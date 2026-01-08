package detect

import (
	"context"
	"fmt"
	"log"

	"github.com/jochenvg/go-udev"
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
