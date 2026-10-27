package detect

import (
	"context"
	"fmt"
	"log"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/blockdev"
)

// UdevDetector écoute les events udev et publie des DeviceEvent
type UdevDetector struct {
	// éventuellement des filtres spécifiques
}

func NewUdevDetector() *UdevDetector {
	return &UdevDetector{}
}

func (d *UdevDetector) Run(ctx context.Context, out chan<- DeviceEvent) error {
	u := udev.Udev{}
	monitor := u.NewMonitorFromNetlink("udev")
	if err := monitor.FilterAddMatchSubsystem("block"); err != nil {
		return fmt.Errorf("failed to add filter: %w", err)
	}

	deviceChan, errChan, err := monitor.DeviceChan(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device channel: %w", err)
	}

	log.Println("Listening for udev events...")

	// Monitor is already listening, so nothing plugged in from now on is missed.
	d.coldplug(ctx, out)

	for {
		select {
		case <-ctx.Done():
			log.Println("Detector stopping due to context cancellation")
			return nil
		case device := <-deviceChan:
			ev := d.convert(device)
			if ev != nil {
				out <- *ev
			}
		case err := <-errChan:
			if err != nil {
				log.Printf("udev monitor error: %v", err)
			}
		}
	}
}

// coldplug emits a DevicePresent event for the USB devices and the audio
// discs already there. Handlers do not start playback on it.
func (d *UdevDetector) coldplug(ctx context.Context, out chan<- DeviceEvent) {
	devices, err := blockdev.List()
	if err != nil {
		log.Printf("coldplug: %v", err)
		return
	}
	for _, dev := range devices {
		device := presentDevice(dev)
		if device == nil {
			continue
		}
		log.Printf("coldplug: found %s", dev.Devnode())
		select {
		case <-ctx.Done():
			return
		case out <- DeviceEvent{Type: DevicePresent, Device: device}:
		}
	}
}

// presentDevice returns dev when it is a USB device or a drive holding an
// audio disc, nil otherwise.
func presentDevice(dev *udev.Device) Device {
	switch {
	case usbPreChecker(dev):
		return &USBDevice{path: dev.Devnode(), udev: dev}
	case discPreChecker(dev) && hasAudioTracks(dev):
		return &DiscDevice{path: dev.Devnode(), udev: dev}
	}
	return nil
}

func (d *UdevDetector) convert(dev *udev.Device) *DeviceEvent {
	device := d.detectDevice(dev)
	if device == nil {
		return nil
	}

	evType := device.DetectEvent()
	if evType == InvalidEvent {
		return nil
	}

	return &DeviceEvent{
		Type:   evType,
		Device: device,
	}
}

func (d *UdevDetector) detectDevice(dev *udev.Device) Device {
	if discPreChecker(dev) {
		return &DiscDevice{path: dev.Devnode(), udev: dev}
	}
	if usbPreChecker(dev) {
		return &USBDevice{path: dev.Devnode(), udev: dev}
	}
	return nil
}
