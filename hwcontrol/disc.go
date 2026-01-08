package hwcontrol

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/unix"
)

const (
	CDROM_SET_SPEED      = 0x5322 // ioctl command for setting speed
	CDROM_PROC_FILE_INFO = "/proc/sys/dev/cdrom/info"
)

func SetDiscSpeed(device string, speed int) error {
	// Open the device
	file, err := os.OpenFile(device, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("failed to close device file: %w", closeErr)
		}
	}()

	// Perform the ioctl call
	err = unix.IoctlSetInt(int(file.Fd()), CDROM_SET_SPEED, speed)
	if err != nil {
		log.Printf("failed to set speed: %w", err)
		return fmt.Errorf("failed to set speed: %w", err)
	}

	return nil
}
