package mounts

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type SymlinkFinder struct {}
/*
func (s *SymlinkFinder) Mount(source string) (string, error) {
	return FindDevicePathAndCache(source, symlinkValidator)
}
*/
func (s *SymlinkFinder) validate(source string) string {
	return validateAndPreparePath(source, createSymlink)
}

func (s *SymlinkFinder) clear(source, target string) {
	return clearSymlinkCache(source, target)
}

// Helper function to create a symbolic link
func createSymlink(source, target string) error {
	// Ensure the target directory exists
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("Error creating target directory: %w", err)
	}

	// Create the symbolic link
	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("Error creating symlink from %s to %s: %w", source, target)
	}
	return nil
}

func clearSymlinkCache(device, path string) {
	// Check if the path is a symlink
	info, err := os.Lstat(path)
	if err != nil {
		log.Printf("Failed to stat path %s: %v", path, err)
		return
	}

	// Only remove if it's a symlink
	if info.Mode()&os.ModeSymlink == 0 {
		log.Printf("Path %s is not a symlink, skipping cleanup", path)
		return
	}
	if err := os.Remove(path); err != nil {
		log.Printf("Failed to remove symlink %s: %v", path, err)
		return
	}
	log.Printf("Successfully cleared %s cache", path)
}
