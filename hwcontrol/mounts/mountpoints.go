package mounts

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

const (
	RetryTimeout  = 3 * time.Second
	RetryInterval = 300 * time.Millisecond
)

var (
	USBNameRegex = regexp.MustCompile(`^sd.*$`)
	letters      = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

type MountManager struct {
	ctx         context.Context
	mountPoints map[string]string
	mu          sync.RWMutex
	mounter     Mounter
}

type Mounter interface {
	validate(source string) (string, error)
	clear(source, target string) error
}

func (m *MountManager) Mount(device string) (string, error) {
	return m.FindRelPath(device, m.FindDevicePathAndCache)
}

func (m *MountManager) Unmount(device string) (string, error) {
	return m.FindRelPath(device, m.SeekMountPointAndClearCache)
}

func NewMountManager(ctx context.Context) (*MountManager, error) {
	var mounter Mounter

	switch config.MountConfig {
	case "fuse":
		mounter = newFuseFinder(ctx)
	case "symlink":
		mounter = newSymlinkFinder(ctx)
	default:
		return nil, fmt.Errorf("unsupported mount type: %s", config.MountConfig)
	}

	return &MountManager{
		ctx:         ctx,
		mountPoints: make(map[string]string),
		mounter:     mounter,
	}, nil
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

func (m *MountManager) SeekMountPointAndClearCache(device string) (string, error) {
	defer m.RemoveCache(device)
	mountPoint, err := seekMountPointWithCacheFallback(device)
	if err != nil {
		return "", fmt.Errorf("Unknown Device %s: %w", device, err)
	}

	if strings.HasPrefix(mountPoint, config.MPDLibraryFolder) {
		return mountPoint, nil
	}

	if mountPoint, err = m.mounter.clear(device, mountPoint); err != nil {
		return "", fmt.Errorf("Failed to unmount: %w", err)
	}
	return mountPoint, nil
}

func (m *MountManager) FindRelPath(device string, callback func(string) (string, error)) (string, error) {
	mountPoint, err := callback(device)
	if err != nil {
		return "", fmt.Errorf("Error finding mountpoint for device %s: %w", device, err)
	}
	relPath, err := filepath.Rel(mountPoint, config.MPDLibraryFolder)
	if err != nil {
		return "", fmt.Errorf("Found mountpoint %s for device %s not in MPDLibraryFolder: %w", mountPoint, device, err)
	}
	return relPath, nil
}

func (m *MountManager) FindDevicePathAndCache(device string) (string, error) {
	mountPoint, err := m.findMountPointWithRetry(device, RetryTimeout, RetryInterval)
	if err != nil {
		return "", fmt.Errorf("Error finding mountpoint for device %s: %w", device, err)
	}
	m.AddCache(device, mountPoint)
	validatedPath, err := m.mounter.validate(mountPoint)
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

func isMPDDir(devnode, mountPoint string) bool {
	if !strings.HasPrefix(devnode, "/dev") {
		return false
	}
	if !USBNameRegex.MatchString(filepath.Base(devnode)) {
		return false
	}
	if !strings.HasPrefix(mountPoint, config.MPDLibraryFolder) {
		return false
	}
	return true
}

func readMountsFile(callback func(device, mountPoint string)) error {
	mountFile := "/proc/mounts"
	file, err := os.Open(mountFile)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", mountFile, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue // Malformed line
		}
		callback(fields[0], fields[1]) // Call the provided callback with device and mount point
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading %s: %w", mountFile, err)
	}
	return nil
}

func seekMountPoint(device string) (string, error) {
	var mountPoint string
	err := readMountsFile(func(d, mp string) {
		if d == device {
			mountPoint = mp
		}
	})
	if err != nil {
		return "", err
	}
	if mountPoint == "" {
		return "", fmt.Errorf("device %s not found", device)
	}
	return mountPoint, nil
}

func populateMountPointCache(m *MountManager) {
	if err := readMountsFile(func(device, mountPoint string) {
		if isMPDDir(device, mountPoint) {
			m.AddCache(device, mountPoint)
		}
	}); err != nil {
		log.Printf("Failed to populate mount point cache: %v", err)
	}
}

func validateAndPreparePath(source string, callback func(string, string) error) (string, error) {
	if strings.HasPrefix(source, config.MPDLibraryFolder) {
		return source, nil // Already valid
	}

	target := generateTarget(source)
	if err := callback(source, target); err != nil {
		return "", fmt.Errorf("Failed to create bind on %s for %s: %w", target, source, err)
	}
	return target, nil
}

func generateTarget(source string) string {
	target := filepath.Join(config.MPDLibraryFolder, config.MPDUSBSubfolder, filepath.Base(source))
	folderInfo, err := os.Stat(target)
	if os.IsNotExist(err) {
		return target
	}
	return fmt.SprintF("%s-%s", target, randomString(5))
}

func randomString(n int) string {
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
