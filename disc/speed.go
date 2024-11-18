package disc

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

const (
	CDROM_SET_SPEED = 0x5322 // ioctl command for setting speed
	CDROM_GET_SPEED = 0x5323 // ioctl command for getting speed
)

func SetDiscSpeed(device string, speed int) error {
	// Open the device
	file, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}
	defer file.Close()

	// Perform the ioctl call
	err = unix.IoctlSetInt(int(file.Fd()), CDROM_SET_SPEED, speed)
	if err != nil {
		return fmt.Errorf("failed to set speed: %w", err)
	}

	return nil
}