package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// SavedDisc is the disc kept in the disc file.
type SavedDisc struct {
	ID     string `json:"id"`
	Device string `json:"device"`
}

// DiscFile keeps the disc in the drive across restarts of the daemon, so it
// does not read the TOC again, which would interrupt playback. It lives in
// the runtime directory, emptied at boot, when the drive is idle.
type DiscFile string

// RuntimeDiscFile is the disc file in the user runtime directory.
func RuntimeDiscFile(appName string) (DiscFile, error) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return "", errors.New("XDG_RUNTIME_DIR is not set")
	}
	return DiscFile(filepath.Join(dir, appName, "disc.json")), nil
}

// Save writes the disc in the file, replacing the previous one.
func (f DiscFile) Save(disc SavedDisc) error {
	data, err := json.Marshal(disc)
	if err != nil {
		return fmt.Errorf("failed to encode the disc: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(string(f)), 0o700); err != nil {
		return fmt.Errorf("failed to create the disc file directory: %w", err)
	}
	// Write then rename, so a crash never leaves a partial file
	tmp := string(f) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("failed to write the disc file: %w", err)
	}
	if err := os.Rename(tmp, string(f)); err != nil {
		return fmt.Errorf("failed to write the disc file: %w", err)
	}
	return nil
}

// Load reads the disc in the file. The error wraps fs.ErrNotExist when there
// is none.
func (f DiscFile) Load() (SavedDisc, error) {
	var disc SavedDisc
	data, err := os.ReadFile(string(f))
	if err != nil {
		return disc, fmt.Errorf("failed to read the disc file: %w", err)
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return disc, fmt.Errorf("failed to decode the disc file: %w", err)
	}
	return disc, nil
}

// Remove deletes the file. A file already gone is not an error.
func (f DiscFile) Remove() error {
	if err := os.Remove(string(f)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove the disc file: %w", err)
	}
	return nil
}
