package mounts

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jochenvg/go-udev"
)

type SymlinkFinder struct {
	symlinkCache     *protectedCache
	mpdLibraryFolder string
	mpdUSBFolder     string
}

func newSymlinkFinder(mpdLibraryFolder, mpdUSBFolder string) *SymlinkFinder {
	s := &SymlinkFinder{
		symlinkCache:     newCache(),
		mpdLibraryFolder: mpdLibraryFolder,
		mpdUSBFolder:     mpdUSBFolder,
	}
	populateSymlinkCache(s)
	return s
}

func (s *SymlinkFinder) validate(device *udev.Device, mountpoint, target string) (string, error) {
	return s.createSymlink(device, mountpoint, target)
}

func (s *SymlinkFinder) clear(device *udev.Device, mountpoint string) (string, error) {
	return s.clearSymlinkCache(device, mountpoint)
}

// Helper function to create a symbolic link
func (s *SymlinkFinder) createSymlink(device *udev.Device, mountpoint, target string) (string, error) {
	// Ensure the target directory exists
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", fmt.Errorf("Error creating target directory: %w", err)
	}

	// Create the symbolic link
	if err := os.Symlink(mountpoint, target); err != nil {
		return "", fmt.Errorf("Error creating symlink from %s to %s: %w", mountpoint, target, err)
	}
	s.symlinkCache.AddCache(device.Devnode(), target)
	return target, nil
}

func (s *SymlinkFinder) clearSymlinkCache(device *udev.Device, mountpoint string) (string, error) {
	devnode := device.Devnode()
	path, err := s.symlinkCache.GetCache(devnode)
	if err != nil {
		return "", fmt.Errorf("Unknown device %s", devnode)
	}
	// Check if the path is a symlink
	if err = checkSymlink(mountpoint, path); err != nil {
		return "", fmt.Errorf("Invalid cached symlink %s for [%s:%s]: %w", path, devnode, mountpoint, err)
	}

	if err = os.Remove(path); err != nil {
		return "", fmt.Errorf("Failed to remove symlink %s: %w", path, err)
	}
	log.Printf("Successfully cleared %s cache", path)
	s.symlinkCache.RemoveCache(devnode)
	return path, nil
}

func validateSymlink(path string, info os.FileInfo) (string, error) {
	if info.Mode()&os.ModeSymlink == 0 {
		return "", fmt.Errorf("Path %s is not a symlink, skipping cleanup", path)
	}
	symlinkTarget, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("Path %s:%s is not a symlink: %w", path, symlinkTarget, err)
	}
	if _, err := os.Stat(symlinkTarget); err != nil {
		return "", fmt.Errorf("Dead Symlink %s: %s does not exists %w", path, symlinkTarget, err)
	}
	return strings.TrimSuffix(symlinkTarget, "/"), nil
}

func checkSymlink(mountpoint, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("Failed to stat path %s: %w", path, err)
	}
	symlinkTarget, err := validateSymlink(path, info)
	if err != nil {
		return fmt.Errorf("Invalid symlink %s: %w", path, err)
	}
	if symlinkTarget != strings.TrimSuffix(mountpoint, "/") {
		return fmt.Errorf("%s symlink do not match %s mountpoint", symlinkTarget, mountpoint)
	}
	return nil
}

func checkSymlinkPopulation(entry fs.DirEntry, path string) (string, error) {
	if entry.IsDir() {
		return "", nil
	}

	info, err := entry.Info()
	if err != nil {
		return "", nil
	}

	symlinkTarget, err := validateSymlink(path, info)
	if err != nil {
		if strings.HasPrefix(err.Error(), "Dead Symlink") {
			return "", err
		}
		return "", nil
	}
	device, err := isSymlinkMountpoint(symlinkTarget)
	if err != nil {
		return "", fmt.Errorf("Device for %s not found: %w", symlinkTarget, err)
	}
	log.Printf("Symlink %s -> %s valid", path, symlinkTarget)

	return device, nil
}

func isSymlinkMountpoint(path string) (string, error) {
	var device string
	deviceFinder := func(d, mp string) {
		if mp == path {
			device = d
			return
		}
	}

	if err := readMountsFile(deviceFinder); err != nil {
		return "", fmt.Errorf("Failed to read mount file while checking symlink %s: %w", path, err)
	}

	if device == "" {
		return "", fmt.Errorf("Path %s not found in mountfile, not a mount", path)
	}
	if !isRemovableNode(device, path) {
		return "", fmt.Errorf("%s:%s is not a removable mountpoint", device, path)
	}
	fmt.Printf("Found device %s for %s path", device, path)
	return device, nil
}

func populateSymlinkCache(s *SymlinkFinder) {
	mpdUSBFolder := filepath.Join(s.mpdLibraryFolder, s.mpdUSBFolder)
	if _, err := os.Stat(mpdUSBFolder); err != nil && os.IsNotExist(err) {
		log.Println("MPD USB Folder does not exist for now")
		return
	}

	entries, err := os.ReadDir(mpdUSBFolder)
	if err != nil {
		log.Printf("Failed to read directory %s: %v", mpdUSBFolder, err)
		return
	}

	for _, entry := range entries {
		path := filepath.Join(mpdUSBFolder, entry.Name())
		trimmedPath := strings.TrimSuffix(path, "/")
		device, err := checkSymlinkPopulation(entry, trimmedPath)
		if err != nil {
			if err = os.Remove(path); err != nil {
				log.Printf("Failed to remove symlink %s: %v", path, err)
			}
		}
		s.symlinkCache.AddCache(device, trimmedPath)
	}
	log.Printf("Symlink populated")
}
