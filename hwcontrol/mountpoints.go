package hwcontrol

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type MountPointCache struct {
	mountPoints		map[string]string
	mu			sync.RWMutex
}

var MountPointsCache = newMountPointCache()

func SeekMountPointAndClearCache(device string) (string, error) {
	defer MountPointsCache.Remove(device)
	if mountPoint, err := seekMountPoint(device); err == nil {
		return mountPoint, nil
	}
	return MountPointsCache.Get(device)

}

func FindMountPointAndAddtoCache(device string) (string, error) {
	timeout := 3 * time.Second
	ticker := time.NewTicker(300 * time.Millisecond)
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

func newMountPointCache() *MountPointCache {
    m := &MountPointCache{
        mountPoints: make(map[string]string),
    }
    populateMountPointCache(m)
    return m
}

// Add a device-to-mount-point association
func (m *MountPointCache) Add(device, mountPoint string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.mountPoints[device] = mountPoint
}

// Remove a device's mount point association
func (m *MountPointCache) Remove(device string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    delete(m.mountPoints, device)
}

// Retrieve a device's mount point
func (m *MountPointCache) Get(device string) (string, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if mountPoint, exists := m.mountPoints[device]; exists {
	    return mountPoint, nil
    }
    return "", fmt.Errorf("%s mount point does not exist in cache", device)
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

func populateMountPointCache(m *MountPointCache) {
    err := readMountsFile(func(device, mountPoint string) {
        if isRemovablePath(device, mountPoint) {
            m.Add(device, mountPoint)
        }
    })

    if err != nil {
        log.Printf("Failed to populate mount point cache: %v", err)
    }
}
