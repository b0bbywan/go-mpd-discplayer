package hwcontrol

import (
	"log"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/mounts"
)

func NewBasicUSBHandlers() []*EventHandler {
	addHandler := newBasicUSBHandler("addUSB", onAddUSBChecker)
	removeHandler := newBasicUSBHandler("removeUSB", onRemoveUSBChecker)

	return []*EventHandler{addHandler, removeHandler}
}

func newBasicUSBHandler(name string, actionChecker func(*udev.Device, string) bool) *EventHandler {
	return newBasicHandler(
		name,
		func(device *udev.Device) bool {
			return usbPreChecker(device)
		},
		func(device *udev.Device, action string) bool {
			return actionChecker(device, action)
		},
	)
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
