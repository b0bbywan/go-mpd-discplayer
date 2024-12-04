package mounts

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

type SymlinkFinder struct {
	ctx          context.Context
	symLinkCache map[string]string
	mu           sync.RWMutex // Protects access to cache
}

func newSymlinkFinder(ctx context.Context) *SymlinkFinder {
	s := &SymlinkFinder{
		ctx:          ctx,
		symLinkCache: make(map[string]string),
	}
	populateSymlinkCache(s)
	return s
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
		return "", fmt.Errorf("Invalid cached symlink %s for [%s:%s]: %w", path, device, mountpoint, err)
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
	if strings.TrimSuffix(symlinkTarget, "/") != strings.TrimSuffix(mountpoint, "/") {
		return fmt.Errorf("%s symlink do not match %s mountpoint", symlinkTarget, mountpoint)
	}
	return nil
}

func checkSymlinkPopulation(path string, info os.FileInfo) (string, error) {
	if info.Mode()&os.ModeSymlink == 0 {
		return "", fmt.Errorf("Path %s is not a symlink", path)
	}

	// TODO: fix: dead symlink not detected
	symlinkTarget, err := os.Readlink(path)
	if err != nil {
		// TODO: Delete dead symlink before returning error
		log.Printf("dead symlink")
		return "", fmt.Errorf("Symlink %s is dead: %w", path, err)
	}
	device, err := isSymlinkMountpoint(strings.TrimSuffix(symlinkTarget, "/"))
	if err != nil {
		return "", fmt.Errorf("device for %s not found: %w", symlinkTarget, err)
	}
	log.Printf("symlink %s:%s valid", path, symlinkTarget)

	return device, nil
}

func isSymlinkMountpoint(path string) (string,error) {
	log.Printf("checking %s mountpoint", path)
	var device string
	if err := readMountsFile(func(d, mp string) {
		if mp == path {
			device = d
		}
	}); err != nil {
		return "", fmt.Errorf("failed to read mount file while checking symlink %s: %w", path, err)
	}

	if device == "" {
		return "", fmt.Errorf("Path %s not found in mountfile, not a mount", path)
	}
	fmt.Printf("found device %s for %s path", device, path)
	return device, nil
}

func populateSymlinkCache(s *SymlinkFinder) {
	err := filepath.Walk(filepath.Join(config.MPDLibraryFolder, config.MPDUSBSubfolder), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("path Error for %s(%v): %v", path, info, err)
			return err
		}
		if info.Name() == config.MPDUSBSubfolder {
			return nil
		}

		trimmedPath := strings.TrimSuffix(path, "/")
		device, err := checkSymlinkPopulation(trimmedPath, info)
		if  err != nil {
			log.Printf("invalid symlink %s: %v", trimmedPath, err)
		}
		s.AddCache(device, trimmedPath)
		return nil
	})
	if err != nil {
		log.Printf("walk failed")
	}
	log.Printf("symlink populated")
}
