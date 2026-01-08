package detect

import (
	"log"

	"github.com/jochenvg/go-udev"
)

type DiscDevice struct {
	path string
	udev *udev.Device
}

func (d *DiscDevice) Path() string {
	return d.path
}

func (d *DiscDevice) Kind() DeviceKind {
	return DeviceDisc
}

func (d *DiscDevice) Udev() *udev.Device {
	return d.udev
}

func (d *DiscDevice) DetectEvent() EventType {
	if !checkDiscChange(d.Udev().Action()) {
		return InvalidEvent
	}
	if onRemoveDiscChecker(d.Udev()) {
		return DeviceRemoved
	}
	if onAddDiscChecker(d.Udev()) {
		return DeviceAdded
	}
	return InvalidEvent
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
