package hwcontrol

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

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
	if device == nil ||
	device.PropertyValue("ID_USB_DRIVER") != "usb-storage" ||
	device.PropertyValue("DEVTYPE") != "partition" ||
	!validFsType(device.PropertyValue("ID_PART_ENTRY_TYPE")) {
		return false
	}
	sysname := device.Sysname()
	if !usbNameRegex.MatchString(sysname) {
		log.Printf("Device sysname '%s' does not match expected kernel rule pattern\n", sysname)
		return false
	}
	return true
}

func FindMountPoint(device string) (string, error) {
	timeout := 3 * time.Second
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	timeoutChan := time.After(timeout)
	for {
		select {
		case <-ticker.C:
			log.Printf("Polling for %s mount point...", device)
			mountPoint, err := SeekMountPoint(device)
			if err == nil {
				return mountPoint, nil
			}
		case <-timeoutChan:
			return "", fmt.Errorf("Device %s not found within timeout", device)
		}
	}
}

func SeekMountPoint(device string) (string, error) {
	mountFile := "/proc/mounts"
	file, err := os.Open(mountFile)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %v", mountFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue // Malformed line
		}
		if fields[0] == device {
			return fields[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading %s: %v", mountFile, err)
	}

	return "", fmt.Errorf("device %s not found", device)
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
