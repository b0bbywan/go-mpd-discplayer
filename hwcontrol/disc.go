package hwcontrol

import (
	"fmt"
	"log"
	"os"
	"github.com/jochenvg/go-udev"
	"github.com/b0bbywan/go-disc-cuer/utils"
	"golang.org/x/sys/unix"
)

const (
	CDROM_SET_SPEED = 0x5322 // ioctl command for setting speed
	CDROM_GET_SPEED = 0x5323 // ioctl command for getting speed
)

func NewBasicDiscHandler(targetDevice string) *EventHandler {
	return &EventHandler{
		PreCheckFunc: func(device *udev.Device) bool {
			return discPreChecker(device, targetDevice)
		},
		AddCheckFunc: onAddDiscChecker,
		RemoveCheckFunc: onRemoveDiscChecker,
	}
}

func onRemoveDiscChecker(device *udev.Device, action string) bool {
	if !checkDiscChange(action) {
		return false
	}
	ejectRequest := device.PropertyValue("DISK_EJECT_REQUEST")
	return ejectRequest == "1"
}

func onAddDiscChecker(device *udev.Device, action string) bool {
	if !checkDiscChange(action) {
		return false
	}
	trackCount := device.PropertyValue("ID_CDROM_MEDIA_TRACK_COUNT_AUDIO")
	return trackCount != ""
}

func discPreChecker(device *udev.Device, targetDevice string) bool {
	if device == nil || device.Sysname() != targetDevice {
		return false
	}
	return true
}

func checkDiscChange(action string) bool {
	if action != "change" {
		log.Printf("Unhandled action: %s\n", action)
		return false
	}
	return true
}

func GetTrackCount(device string) (int, error) {
	return utils.GetTrackCount(device)
}

func SetDiscSpeed(device string, speed int) error {
	// Open the device
	file, err := os.OpenFile(device, os.O_RDWR, 0)
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