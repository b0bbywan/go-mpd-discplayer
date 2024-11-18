package disc

import (
	"log"
	"github.com/jochenvg/go-udev"
	"go.uploadedlobster.com/discid"
)

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

func NewBasicDiscHandler(targetDevice string) *EventHandler {
	return &EventHandler{
		PreCheckFunc: func(device *udev.Device) bool {
            return discPreChecker(device, targetDevice)
        },
		AddCheckFunc: onAddDiscChecker,
		RemoveCheckFunc: onRemoveDiscChecker,
	}
}

func GetTrackCount() (int, error) {
	disc, err := discid.Read("")
	if err != nil {
		return 0, err
	}
	defer disc.Close()

	return disc.LastTrackNumber(), nil
}
