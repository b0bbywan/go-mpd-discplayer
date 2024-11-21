package hwcontrol

import (
    "log"
    "regexp"
    "github.com/jochenvg/go-udev"
)

var usbNameRegex = regexp.MustCompile(`^sd.*$`)

func NewBasicUSBHandler() *EventHandler {
    return &EventHandler{
        PreCheckFunc: usbPreChecker,
        AddCheckFunc: onAddUSBChecker,
        RemoveCheckFunc: onRemoveUSBChecker,
    }
}

func onRemoveUSBChecker(device *udev.Device, action string) bool {
    return action == "remove"
}

func onAddUSBChecker(device *udev.Device, action string) bool {
    return action == "add"
}

func usbPreChecker(device *udev.Device) bool {
    if device == nil || device.PropertyValue("SUBSYSTEMS") != "usb" {
        return false
    }
    sysname := device.Sysname()
    if !usbNameRegex.MatchString(sysname) {
        log.Printf("Device sysname '%s' does not match expected kernel rule pattern\n", sysname)
        return false
    }
    return true
}
