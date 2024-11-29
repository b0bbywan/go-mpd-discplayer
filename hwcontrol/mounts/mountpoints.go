package mounts

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

const (
	RetryTimeout  = 3 * time.Second
	RetryInterval = 300 * time.Millisecond
)

type MountManager struct {
	mountPoints map[string]string
	fuseMounts  map[string]*fuse.Server
	mu          sync.RWMutex
	mounter     Mounter
}

type Mounter interface {
	validate(source string) (string, error)
	clear(source string) (error)
}

/*
func (m *MountManager) Mount(device string) (string, error) {
	return FindDevicePathAndCache(device, m.mounter.validate)
}
*/
func (m *MountManager) Unmount(device, target string) {
	return 
}

func NewMountManager() (*MountManager, error) {
    var mounter Mounter

    switch config.MountConfig {
    case "fuse":
        mounter = &FuseFinder{}
    case "symlink":
        mounter = &SymlinkFinder{}
    default:
        return nil, fmt.Errorf("unsupported mount type: %s", config.MountConfig)
    }

    return &MountManager{
        mountPoints: make(map[string]string),
        mounter:     mounter,
        fuseMounts:  initializeFuseMap(MountConfig),
    }, nil
}

func initializeFuseMap(mountType string) map[string]*fuse.Server {
    if mountType == "fuse" {
        return make(map[string]*fuse.Server)
    }
    return nil // Avoid allocating unnecessary resources
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

func SeekMountPointAndClearCache(device string) (string, error) {
	path, err := MountPointsCache.Get(device)
	if err == nil {
		go clearCache(device, path)
		return path, nil
	}
	return "", fmt.Errorf("Device %s not in cache: %w", device, err)
}

func clearCache(device, path string, cleaner func(string, string)) {
	cleaner(device, path)
	MountPointsCache.Remove(device)
}

func (m *MountManager) FindRelPath(device string) (string, error) {
	mountPoint, err := m.FindDevicePathAndCache(device)
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
	mountPoint, err := findMountPointWithRetry(device, RetryTimeout, RetryInterval)
	if err != nil {
		return "", fmt.Errorf("Error finding mountpoint for device %s: %w", device, err)
	}
	validatedPath := m.mounter.validate(mountPoint)
	MountPointsCache.Add(device, validatedPath)
	return validatedPath, nil
}

func findMountPointWithRetry(device string, timeout, interval time.Duration) (string, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timeoutChan := time.After(timeout)
	for {
		if mountPoint, err := seekMountPoint(device); err == nil {
			MountPointsCache.Add(device, mountPoint)
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

func isRemovablePath(devnode, mountPoint string) bool {
	if !strings.HasPrefix(devnode, "/dev") {
		return false
	}
	if !usbNameRegex.MatchString(filepath.Base(devnode)) {
		return false
	}
	if mountPoint == "/" ||
		strings.HasPrefix(mountPoint, "/home") ||
		strings.HasPrefix(mountPoint, "/var") ||
		strings.HasPrefix(mountPoint, "/boot") ||
		strings.HasPrefix(mountPoint, "/proc") ||
		strings.HasPrefix(mountPoint, "/dev") {
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
		return fmt.Errorf("error reading %s: %v", mountFile, err)
	}

	return nil
}

func seekMountPoint(device string) (string, error) {
	var mountPoint string
	if err := readMountsFile(func(d, mp string) {
		if d == device {
			mountPoint = mp
		}
	}); err != nil {
		return "", err
	}

	if mountPoint == "" {
		return "", fmt.Errorf("device %s not found", device)
	}
	return mountPoint, nil
}

func populateMountPointCache(m *MountPointCache) {
	if err := readMountsFile(func(device, mountPoint string) {
		if isRemovablePath(device, mountPoint) {
			m.AddCache(device, mountPoint)
		}
	}); err != nil {
		log.Printf("Failed to populate mount point cache: %v", err)
	}
}

func validateAndPreparePath(source string, callback func(string, string) error) string {
	if strings.HasPrefix(source, config.MPDLibraryFolder) {
		return source // Already valid
	}

	target := filepath.Join(config.MPDLibraryFolder, filepath.Base(source))
	if err := callback(source, target); err != nil {
		log.Printf("Failed to create symlink for %s: %v", source, err)
		return source // Fallback to the original path
	}

	return target
}
