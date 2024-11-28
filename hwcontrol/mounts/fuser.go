package hwcontrol

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type FuseFinder struct {}

func (s *FuseFinder) Find(device string) (string, error) {
	return FindDevicePathAndCache(device, fuseValidator)
}

func fuseValidator(source string) string {
	return validateAndPreparePath(source, createFuseMountAndCache)
}

func (m *MountPointCache) AddServer(device string, server *fuse.Server) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fuseMounts[device] = server
}

// Retrieve a device's FUSE server
func (m *MountPointCache) GetServer(device string) (*fuse.Server, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    server, exists := m.fuseMounts[device]
    if !exists {
        return nil, fmt.Errorf("No FUSE server for device %s", device)
    }
    return server, nil
}

func createFuseMountAndCache(source, target string) error {
	server, err := createFuseMount(source, target)
	if err != nil {
		return fmt.Errorf("failed to create fuse mount for %s:%s: %w", source, target, err)
	}
	MountPointsCache.AddServer(source, server)
	return nil
}

func createFuseMount(source, target string) (*fuse.Server, error) {
	// Ensure the target directory exists
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, fmt.Errorf("Error creating target directory: %w", err)
	}

	// Create a loopback filesystem
	loopbackRoot, err := fs.NewLoopbackRoot(source)
	if err != nil {
		return nil, fmt.Errorf("Error creating loopback root: %w", err)
	}

	opts := &fs.Options{
		MountOptions: fuse.MountOptions{
			FsName:  "usb-loopback",
			Name:    filepath.Base(source),
			Options: []string{"auto_unmount"},
		},
	}

	server, err := fs.Mount(target, loopbackRoot, opts)
	if err != nil {
		return nil, fmt.Errorf("Error mounting FUSE filesystem: %w", err)
	}

	// Wait for the server in a goroutine so it doesn't block the main flow
	go func() {
		server.Wait()
		log.Printf("Unmounted FUSE mount for %s", target)
	}()

	log.Printf("Created FUSE mount for %s at %s", source, target)
	return server, nil
}

func clearFuseMount(device, target string) {
	server, err := MountPointsCache.GetServer(device)
	if err != nil {
		log.Printf("Failed to find %s fuse server in cache: %v", device, err)
		return
	}
	if err = server.Unmount(); err != nil {
		log.Printf("Failed to unmount %s fuse server in cache: %v", device, err)
		return
	}
}