package mounts

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

type SymlinkFinder struct {
	ctx          context.Context
	symLinkCache map[string]string
	mu           sync.RWMutex // Protects access to cache
}

func newSymlinkFinder(ctx context.Context) *SymlinkFinder {
	return &SymlinkFinder{
		ctx:          ctx,
		symLinkCache: make(map[string]string),
	}
}

func (s *SymlinkFinder) AddCache(source, target string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.symLinkCache[source] = target
}

func (s *SymlinkFinder) RemoveCache(source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.symLinkCache, source)
}

func (s *SymlinkFinder) GetCache(source string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if symLink, exists := s.symLinkCache[source]; exists {
		return symLink, nil
	}
	return "", fmt.Errorf("%s mount point does not exist in cache", source)
}

func (s *SymlinkFinder) validate(device, mountpoint string) (string, error) {
	return validateAndPreparePath(device, mountpoint, s.createSymlink)
}

func (s *SymlinkFinder) clear(source, mountpoint string) (string, error) {
	return s.clearSymlinkCache(source, mountpoint)
}

// Helper function to create a symbolic link
func (s *SymlinkFinder) createSymlink(device, mountpoint, target string) (string, error) {
	// Ensure the target directory exists
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", fmt.Errorf("Error creating target directory: %w", err)
	}

	// Create the symbolic link
	if err := os.Symlink(mountpoint, target); err != nil {
		return "", fmt.Errorf("Error creating symlink from %s to %s: %w", mountpoint, target, err)
	}
	s.AddCache(device, target)
	return target, nil
}

func (s *SymlinkFinder) clearSymlinkCache(device, mountpoint string) (string, error) {
	path, err := s.GetCache(device)
	if err != nil {
		return "", fmt.Errorf("Unknown device %s", device)
	}
	// Check if the path is a symlink
	if err = checkSymlink(mountpoint, path); err != nil {
		return "", fmt.Errorf("Invalid cached symlink %s for [%s:%s]: %w", path, device, mountpoint,err)
	}

	if err = os.Remove(path); err != nil {
		return "", fmt.Errorf("Failed to remove symlink %s: %w", path, err)
	}
	log.Printf("Successfully cleared %s cache", path)
	s.RemoveCache(device)
	return path, nil
}

func checkSymlink(mountpoint, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("Failed to stat path %s: %w", path, err)
	}

	// Only remove if it's a symlink
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("Path %s is not a symlink, skipping cleanup", path)
	}
	symlinkTarget, err := os.Readlink(path)
	if err == nil {
		log.Printf("%s symlink not dead: %s still exists", path, symlinkTarget)
	}
	if symlinkTarget != mountpoint {
		return fmt.Errorf("%s symlink do not match %s mountpoint", symlinkTarget, mountpoint)
	}
	return nil
}