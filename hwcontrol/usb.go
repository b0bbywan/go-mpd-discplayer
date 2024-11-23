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
	if action != EventAdd {
		return false
	}
	devnode := device.Devnode()
	mountPoint, err := FindMountPoint(devnode)
	if err != nil {
		log.Printf("Device %s does not have a valid mount point: %v\n", devnode, err)
		return false
	}
	return mountPoint != ""
}

func usbPreChecker(device *udev.Device) bool {
	if device == nil ||
	device.PropertyValue("ID_USB_DRIVER") != "usb-storage" ||
	device.PropertyValue("DEVTYPE") != "partition" {
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
			log.Println("Polling for mount point...")
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
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/mounts: %v", err)
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
		return "", fmt.Errorf("error reading /proc/mounts: %v", err)
	}

	return "", fmt.Errorf("device %s not found", device)
}
