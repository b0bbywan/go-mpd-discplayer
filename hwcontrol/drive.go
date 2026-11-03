package hwcontrol

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/unix"
)

const CDROM_SET_SPEED = 0x5322 // ioctl command for setting speed

// SetDiscSpeed sets the read speed of the drive.
func SetDiscSpeed(device string, speed int) error {
	return withDrive(device, openReadOnly, func(fd int) error {
		if err := unix.IoctlSetInt(fd, CDROM_SET_SPEED, speed); err != nil {
			return fmt.Errorf("failed to set speed: %w", err)
		}
		return nil
	})
}

// withDrive opens device with open, runs fn on its file descriptor, then
// closes it.
func withDrive(device string, open func(string) (*os.File, error), fn func(fd int) error) error {
	file, err := open(device)
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("failed to close device file: %s", closeErr)
		}
	}()
	return fn(int(file.Fd()))
}

func openReadOnly(device string) (*os.File, error) {
	return os.OpenFile(device, os.O_RDONLY, 0)
}

// openDrive opens the drive even without a readable disc (O_NONBLOCK). SCSI
// eject commands need write access, so read-only is only a fallback.
func openDrive(device string) (*os.File, error) {
	file, err := os.OpenFile(device, os.O_RDWR|unix.O_NONBLOCK, 0)
	if err == nil {
		return file, nil
	}
	return os.OpenFile(device, os.O_RDONLY|unix.O_NONBLOCK, 0)
}
