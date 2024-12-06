package mounts

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/jochenvg/go-udev"
)

type FuseFinder struct {
	ctx        context.Context
	fuseMounts map[string]*fuse.Server
	mu         sync.RWMutex // Protects access to mounts
}

func newFuseFinder(ctx context.Context) *FuseFinder {
	return &FuseFinder{
		ctx:        ctx,
		fuseMounts: make(map[string]*fuse.Server),
	}
}

func (f *FuseFinder) validate(device *udev.Device, mountpoint string) (string, error) {
	return validateAndPreparePath(device, mountpoint, f.createFuseMountAndCache)
}

func (f *FuseFinder) clear(device *udev.Device, target string) (string, error) {
	return f.clearFuseMount(device, target)
}

func (f *FuseFinder) AddServer(device string, server *fuse.Server) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fuseMounts[device] = server
}

func (f *FuseFinder) DeleteServer(device string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.fuseMounts, device)
}

// Retrieve a device's FUSE server
func (f *FuseFinder) GetServer(device string) (*fuse.Server, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	server, exists := f.fuseMounts[device]
	if !exists {
		return nil, fmt.Errorf("No FUSE server for device %s", device)
	}
	return server, nil
}

func (f *FuseFinder) createFuseMountAndCache(device *udev.Device, mountpoint, target string) (string, error) {
	server, err := f.createFuseMount(mountpoint, target)
	if err != nil {
		return "", fmt.Errorf("failed to create fuse mount for %s:%s: %w", mountpoint, target, err)
	}
	f.AddServer(device.Devnode(), server)
	return target, nil
}

func (f *FuseFinder) createFuseMount(source, target string) (*fuse.Server, error) {
	// Ensure the target directory exists
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, fmt.Errorf("Error creating target directory: %w", err)
	}
	log.Printf("Target %s created\n", target)
	// Create a loopback filesystem
	loopbackRoot, err := fs.NewLoopbackRoot(source)
	if err != nil {
		return nil, fmt.Errorf("Error creating loopback root: %w", err)
	}
	log.Printf("loopback %v created\n", loopbackRoot)

	opts := &fs.Options{
		MountOptions: fuse.MountOptions{
			FsName:  source,
			Name:    filepath.Base(source),
			Options: []string{"auto_unmount"},
		},
	}

	server, err := fs.Mount(target, loopbackRoot, opts)
	if err != nil {
		return nil, fmt.Errorf("Error mounting FUSE filesystem: %w", err)
	}
	log.Printf("server %v created\n", server)


	log.Printf("Created FUSE mount for %s at %s", source, target)
	return server, nil
}

func (f *FuseFinder) clearFuseMount(device *udev.Device, target string) (string, error) {
	devnode := device.Devnode()
	server, err := f.GetServer(devnode)
	if err != nil {
		return "", fmt.Errorf("Failed to find %s fuse server in cache: %w", devnode, err)

	}
	if err = server.Unmount(); err != nil {
		return "", fmt.Errorf("Failed to unmount %s fuse server in cache: %w", devnode, err)
	}
	f.DeleteServer(devnode)
	return target, nil
}

func WaitContext(server *fuse.Server) {
	server.Wait()
	log.Printf("Unmounted FUSE mount for %s", target)
}
