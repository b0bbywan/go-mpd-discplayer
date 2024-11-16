package disc

import (
	"context"
	"fmt"
	"log"
	"github.com/jochenvg/go-udev"
	"go.uploadedlobster.com/discid"
)

type EventHandler struct {
	OnAdd    func()
	OnRemove func()
}

func GetTrackCount() (int, error) {
	disc, err := discid.Read("")
	if err != nil {
		return 0, err
	}
	defer disc.Close()

	return disc.LastTrackNumber(), nil
}

// StartMonitor begins monitoring udev events for the specified device and executes the appropriate event handlers.
func StartMonitor(ctx context.Context, targetSysname string, handler EventHandler) error {
	if targetSysname == "" {
		log.Println("Target system name must not be empty.")
		return fmt.Errorf("target system name must not be empty")
	}
	u := udev.Udev{}
	monitor := u.NewMonitorFromNetlink("udev")
	if err := monitor.FilterAddMatchSubsystem("block"); err != nil {
		log.Printf("Failed to add filter: %v", err)
		return fmt.Errorf("Failed to add filter: %w", err)
	}

	// Start the monitor and get the device channel
	deviceChan, errChan, err := monitor.DeviceChan(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device channel: %w", err)
	}

	log.Println("Listening for udev events...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping from disc.")
			return nil
		case device := <-deviceChan:
			handleDeviceEvent(device, targetSysname, handler)
		case err := <-errChan:
			if err != nil {
				log.Printf("Received error from monitor: %v\n", err)
			}
		}
	}
}

func handleDeviceEvent(device *udev.Device, targetSysname string, handler EventHandler) {
	// Get the sysname of the device and check if it matches the target
	if device == nil || device.Sysname() != targetSysname {
		return
	}

	action := device.Action()
	switch action {
	case "change":
		handleDeviceChange(device, handler)
	default:
		log.Printf("Unknown event: %s\n", action)
	}
}

func handleDeviceChange(device *udev.Device, handler EventHandler) {
	trackCount := device.PropertyValue("ID_CDROM_MEDIA_TRACK_COUNT_AUDIO")
	ejectRequest := device.PropertyValue("DISK_EJECT_REQUEST")

	if ejectRequest == "1" && handler.OnRemove != nil {
		handler.OnRemove()
	} else if trackCount != "" && handler.OnAdd != nil {
		handler.OnAdd()
	}
}