package mounts

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

var (
	letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

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

func generateTarget(source string) string {
	target := filepath.Join(config.MPDLibraryFolder, config.MPDUSBSubfolder, filepath.Base(source))
	_, err := os.Stat(target)
	if os.IsNotExist(err) {
		return target
	}
	return fmt.Sprintf("%s-%s", target, randomString(5))
}

func randomString(n int) string {
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
