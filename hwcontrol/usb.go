package hwcontrol

import (
    "log"
    "regexp"
    "github.com/jochenvg/go-udev"
)

var usbNameRegex = regexp.MustCompile(`^sd.*$`)

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
