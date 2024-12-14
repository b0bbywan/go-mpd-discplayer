package mounts

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/config"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

const (
	RetryTimeout  = 3 * time.Second
	RetryInterval = 300 * time.Millisecond
)

var (
	USBNameRegex = regexp.MustCompile(`^sd.*$`)
)

type MountManager struct {
	mountPoints *protectedCache
	mounter     Mounter
}

type Mounter interface {
	validate(device *udev.Device, mountpoint, target string) (string, error)
	clear(device *udev.Device, target string) (string, error)
}

func NewMountManager(client *mpdplayer.ReconnectingMPDClient) (*MountManager, error) {
	mounter, err := newMounter(client)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Mounter: %w")
	}
	m := &MountManager{
		mountPoints: newCache(),
		mounter:     mounter,
	}
	populateMountPointCache(m)
	return m, nil
}

func (m *MountManager) Mount(device *udev.Device) (string, error) {
	mountPoint, err := m.FindDevicePathAndCache(device)
	if err != nil {
		return "", fmt.Errorf("Failed to find a mountpoint for %s while mounting: %w", device.Devnode(), err)
	}
	return m.FindRelPath(mountPoint)
}

func (m *MountManager) Unmount(device *udev.Device) (string, error) {
	mountPoint, err := m.SeekMountPointAndClearCache(device)
	if err != nil {
		return "", fmt.Errorf("Failed to find a mountpoint for %s while unmounting: %w", device.Devnode(), err)
	}
	return m.FindRelPath(mountPoint)
}

func (m *MountManager) FindRelPath(mountPoint string) (string, error) {
	relPath, err := filepath.Rel(config.MPDLibraryFolder, mountPoint)
	if err != nil {
		return "", fmt.Errorf("Found mountpoint %s not in MPDLibraryFolder: %w", mountPoint, err)
	}
	return relPath, nil
}

func (m *MountManager) SeekMountPointAndClearCache(device *udev.Device) (string, error) {
	defer m.mountPoints.RemoveCache(device.Devnode())
	mountPoint, err := m.seekMountPointWithCacheFallback(device.Devnode())
	if err != nil && config.MountConfig != "mpd" {
		return "", fmt.Errorf("Unknown Device %s: %w", device.Devnode(), err)
	}

	if strings.HasPrefix(mountPoint, config.MPDLibraryFolder) {
		return mountPoint, nil
	}

	if mountPoint, err = m.mounter.clear(device, mountPoint); err == nil {
		return mountPoint, nil
	}

	return "", fmt.Errorf("Failed to unmount: %w", err)
}

func (m *MountManager) FindDevicePathAndCache(device *udev.Device) (string, error) {
	mountPoint, err := m.findMountPointWithRetry(device.Devnode(), RetryTimeout, RetryInterval)
	if err != nil && config.MountConfig != "mpd" {
		return "", fmt.Errorf("Error finding mountpoint for device %s: %w", device, err)
	}
	m.mountPoints.AddCache(device.Devnode(), mountPoint)

	if strings.HasPrefix(mountPoint, config.MPDLibraryFolder) {
		return mountPoint, nil
	}

	target := generateTarget(mountPoint)

	validatedPath, err := m.mounter.validate(device, mountPoint, target)
	if err != nil {
		return "", fmt.Errorf("mounter validation failed: %w", err)
	}
	return validatedPath, nil
}

func (m *MountManager) findMountPointWithRetry(device string, timeout, interval time.Duration) (string, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timeoutChan := time.After(timeout)
	for {
		if mountPoint, err := seekMountPoint(device); err == nil {
			m.mountPoints.AddCache(device, mountPoint)
			return mountPoint, nil
		}
		select {
		case <-ticker.C:
			log.Printf("Polling for %s mount point...", device)
		case <-timeoutChan:
			return "", fmt.Errorf("Device %s not found within timeout", device)
		}
	}
}

func (m *MountManager) seekMountPointWithCacheFallback(device string) (string, error) {
	if mountPoint, err := seekMountPoint(device); err == nil {
		return mountPoint, nil
	}
	if path, err := m.mountPoints.GetCache(device); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("Failed to find %s in mount file and cache", device)
}

func newMounter(client *mpdplayer.ReconnectingMPDClient) (Mounter, error) {
	switch config.MountConfig {
	case "symlink":
		return newSymlinkFinder(), nil
	case "mpd":
		return newMpdFinder(client)
	default:
		return nil, fmt.Errorf("Unsupported mount type: %s", config.MountConfig)
	}
}

func isRemovableNode(devnode, mountPoint string) bool {
	if !strings.HasPrefix(devnode, "/dev") {
		return false
	}
	if !USBNameRegex.MatchString(filepath.Base(devnode)) {
		return false
	}
	if mountPoint == "/" ||
		mountPoint == "/home" ||
		mountPoint == "/var" ||
		strings.HasPrefix(mountPoint, "/var/lib/docker") ||
		strings.HasPrefix(mountPoint, "/boot") ||
		strings.HasPrefix(mountPoint, "/proc") ||
		strings.HasPrefix(mountPoint, "/dev") {
		return false
	}
	return true
}

func seekMountPoint(device string) (string, error) {
	var mountPoint string
	mountPointSeeker := func(d, mp string) {
		if d == device {
			mountPoint = mp
			return
		}
	}

	if err := readMountsFile(mountPointSeeker); err != nil {
		return "", fmt.Errorf("Failed to seek mount point %s: %w", device, err)
	}

	if mountPoint == "" {
		return "", fmt.Errorf("device %s not found", device)
	}
	return mountPoint, nil
}

func populateMountPointCache(m *MountManager) {
	if err := readMountsFile(func(device, mountPoint string) {
		if isRemovableNode(device, mountPoint) {
			m.mountPoints.AddCache(device, mountPoint)
		}
	}); err != nil {
		log.Printf("Failed to populate mount point cache: %v", err)
	}
}
