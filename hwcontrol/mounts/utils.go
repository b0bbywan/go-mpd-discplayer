package mounts

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

var (
	letters      = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	USBNameRegex = regexp.MustCompile(`^sd.*$`)
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

func isRemovableNode(devnode, mountPoint string) bool {
	if !strings.HasPrefix(devnode, "/dev") {
		return false
	}
	if !USBNameRegex.MatchString(filepath.Base(devnode)) {
		return false
	}
	if mountPoint == "/" ||
		mountPoint == "/home" ||
		mountPoint == "/var" ||
		strings.HasPrefix(mountPoint, "/var/lib/docker") ||
		strings.HasPrefix(mountPoint, "/boot") ||
		strings.HasPrefix(mountPoint, "/proc") ||
		strings.HasPrefix(mountPoint, "/dev") {
		return false
	}
	return true
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
