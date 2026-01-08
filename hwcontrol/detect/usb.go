package detect

import (
	"log"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/mounts"
)

type USBDevice struct {
	path string
	udev *udev.Device
}

func (d *USBDevice) Path() string {
	return d.path
}

func (d *USBDevice) Kind() DeviceKind {
	return DeviceUSB
}

func (d *USBDevice) Udev() *udev.Device {
	return d.udev
}

func (d *USBDevice) DetectEvent() EventType {
	action := d.Udev().Action()
	if onRemoveUSBChecker(d.Udev(), action) {
		return DeviceRemoved
	}
	if onAddUSBChecker(d.Udev(), action) {
		return DeviceAdded
	}
	return InvalidEvent
}

func onRemoveUSBChecker(device *udev.Device, action string) bool {
	return action == EventRemove
}

func onAddUSBChecker(device *udev.Device, action string) bool {
	return action == EventAdd
}

func usbPreChecker(device *udev.Device) bool {
	if device == nil ||
		device.PropertyValue("ID_USB_DRIVER") != "usb-storage" ||
		device.PropertyValue("DEVTYPE") != "partition" ||
		!validFsType(device.PropertyValue("ID_PART_ENTRY_TYPE")) {
		return false
	}
	sysname := device.Sysname()
	if !mounts.USBNameRegex.MatchString(sysname) {
		log.Printf("Device sysname '%s' does not match expected kernel rule pattern\n", sysname)
		return false
	}
	return true
}

func validFsType(fsType string) bool {
	if fsType == "" ||
		fsType == "0x0" || // empty partition
		fsType == "0xef" || // efi system partition
		fsType == "0x82" { // linux swap
		return false
	}
	return true
}
