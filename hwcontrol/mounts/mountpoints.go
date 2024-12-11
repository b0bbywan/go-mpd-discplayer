package mounts

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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
	mountPoints map[string]string
	mu          sync.RWMutex
	mounter     Mounter
}

type Mounter interface {
	validate(device *udev.Device, mountpoint string) (string, error)
	clear(device *udev.Device, target string) (string, error)
}

func NewMountManager(client *mpdplayer.ReconnectingMPDClient) (*MountManager, error) {
	var mounter Mounter

	switch config.MountConfig {
	case "fuse":
		mounter = newFuseFinder()
	case "symlink":
		mounter = newSymlinkFinder()
	default:
		return nil, fmt.Errorf("unsupported mount type: %s", config.MountConfig)
	}
	m := &MountManager{
		mountPoints: make(map[string]string),
		mounter:     mounter,
	}
	populateMountPointCache(m)
	return m, nil
}

func (m *MountManager) Mount(device *udev.Device) (string, error) {
	return m.FindRelPath(device, m.FindDevicePathAndCache)
}

func (m *MountManager) Unmount(device *udev.Device) (string, error) {
	return m.FindRelPath(device, m.SeekMountPointAndClearCache)
}

// Add a device-to-mount-point association
func (m *MountManager) AddCache(device, mountPoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mountPoints[device] = mountPoint
}

// Remove a device's mount point association
func (m *MountManager) RemoveCache(device string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.mountPoints, device)
}

// Retrieve a device's mount point
func (m *MountManager) GetCache(device string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if mountPoint, exists := m.mountPoints[device]; exists {
		return mountPoint, nil
	}
	return "", fmt.Errorf("%s mount point does not exist in cache", device)
}

func (m *MountManager) SeekMountPointAndClearCache(device *udev.Device) (string, error) {
	defer m.RemoveCache(device.Devnode())
	mountPoint, err := m.seekMountPointWithCacheFallback(device.Devnode())
	if err != nil {
		return "", fmt.Errorf("Unknown Device %s: %w", device.Devnode(), err)
	}

	if strings.HasPrefix(mountPoint, config.MPDLibraryFolder) {
		return mountPoint, nil
	}

	if mountPoint, err = m.mounter.clear(device, mountPoint); err != nil {
		return "", fmt.Errorf("Failed to unmount: %w", err)
	}
	return mountPoint, nil
}

func (m *MountManager) FindRelPath(device *udev.Device, callback func(*udev.Device) (string, error)) (string, error) {
	mountPoint, err := callback(device)
	if err != nil {
		return "", fmt.Errorf("Error finding mountpoint for device %s: %w", device.Devnode(), err)
	}
	relPath, err := filepath.Rel(config.MPDLibraryFolder, mountPoint)
	if err != nil {
		return "", fmt.Errorf("Found mountpoint %s for device %s not in MPDLibraryFolder: %w", mountPoint, device, err)
	}
	return relPath, nil
}

func (m *MountManager) FindDevicePathAndCache(device *udev.Device) (string, error) {
	mountPoint, err := m.findMountPointWithRetry(device.Devnode(), RetryTimeout, RetryInterval)
	if err != nil {
		return "", fmt.Errorf("Error finding mountpoint for device %s: %w", device, err)
	}
	m.AddCache(device.Devnode(), mountPoint)
	validatedPath, err := m.mounter.validate(device, mountPoint)
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
			m.AddCache(device, mountPoint)
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
	if path, err := m.GetCache(device); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("Failed to find %s in mount file and cache", device)
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
			m.AddCache(device, mountPoint)
		}
	}); err != nil {
		log.Printf("Failed to populate mount point cache: %v", err)
	}
}
