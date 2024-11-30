package hwcontrol

import (
	"fmt"
	"log"
	"os"

	"github.com/jochenvg/go-udev"
	"golang.org/x/sys/unix"

	"github.com/b0bbywan/go-disc-cuer/utils"
)

const (
	CDROM_SET_SPEED      = 0x5322 // ioctl command for setting speed
	CDROM_PROC_FILE_INFO = "/proc/sys/dev/cdrom/info"
)

// NewBasicDiscHandlers initializes two event handlers for "add" and "remove" disc events.
func NewBasicDiscHandlers() []*EventHandler {
	addHandler := newBasicDiscHandler("addDisc", onAddDiscChecker)
	removeHandler := newBasicDiscHandler("removeDisc", onRemoveDiscChecker)

	return []*EventHandler{addHandler, removeHandler}
}

// newBasicDiscHandler creates a reusable event handler for disc-related actions.
func newBasicDiscHandler(name string, actionChecker func(*udev.Device) bool) *EventHandler {
	return newBasicHandler(
		name,
		func(device *udev.Device) bool {
			return discPreChecker(device)
		},
		func(device *udev.Device, action string) bool {
			return discActionChecker(device, device.Action(), actionChecker)
		},
	)
}

// discActionChecker validates a disc action and runs additional custom checks.
func discActionChecker(device *udev.Device, action string, checker func(*udev.Device) bool) bool {
	if !checkDiscChange(action) {
		return false
	}
	return checker(device)
}

// onRemoveDiscChecker checks if a disc removal was requested.
func onRemoveDiscChecker(device *udev.Device) bool {
	if device.Action() == EventRemove {
		return true
	}
	ejectRequest := device.PropertyValue("DISK_EJECT_REQUEST")
	return ejectRequest == "1"
}

// onAddDiscChecker verifies that the inserted disc has audio tracks.
func onAddDiscChecker(device *udev.Device) bool {
	if device.Action() == EventRemove {
		return false
	}
	trackCount := device.PropertyValue("ID_CDROM_MEDIA_TRACK_COUNT_AUDIO")
	return trackCount != ""
}

// discPreChecker ensures the device is valid and matches the target device.
func discPreChecker(device *udev.Device) bool {
	if device == nil ||
		device.PropertyValue("ID_CDROM") != "1" ||
		device.PropertyValue("ID_FS_TYPE") != "" {
		return false
	}
	return true
}

// checkDiscChange validates that the action is handled.
func checkDiscChange(action string) bool {
	if action == EventChange || action == EventRemove || action == EventAdd {
		return true
	}
	log.Printf("Unhandled action: %s\n", action)
	return false
}

func GetTrackCount(device string) (int, error) {
	return utils.GetTrackCount(device)
}

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
