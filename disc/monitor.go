package disc

import (
	"context"
	"fmt"
	"log"

	"github.com/jochenvg/go-udev"
)

type EventHandler struct {
	PreCheckFunc	func(*udev.Device) bool
	AddCheckFunc	func(*udev.Device, string) bool
	RemoveCheckFunc	func(*udev.Device, string) bool
	OnAddFunc		func()
	OnRemoveFunc	func()
}

func (h *EventHandler) PreCheck(device *udev.Device) bool {
	if h.PreCheckFunc != nil && device != nil {
		return h.PreCheckFunc(device)
	}
	return false
}

func (h *EventHandler) AddCheck(device *udev.Device, action string) bool {
	if h.AddCheckFunc != nil && device != nil {
		return h.AddCheckFunc(device, action)
	}
	return false
}
func (h *EventHandler) RemoveCheck(device *udev.Device, action string) bool {
	if h.RemoveCheckFunc != nil && device != nil {
		return h.RemoveCheckFunc(device, action)
	}
	return false
}

func (h *EventHandler) OnAdd() {
	if h.OnAddFunc != nil {
		h.OnAddFunc()
	}
}

func (h *EventHandler) OnRemove() {
	if h.OnRemoveFunc != nil {
		h.OnRemoveFunc()
	}
}

type CompositeEventHandler struct {
	Handlers []*EventHandler
}

// StartMonitor begins monitoring udev events for the specified device and executes the appropriate event handlers.
func (c *CompositeEventHandler) StartMonitor(ctx context.Context) error {
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
			c.HandleDevice(device)
		case err := <-errChan:
			if err != nil {
				log.Printf("Received error from monitor: %v\n", err)
			}
		}
	}
}

func (c *CompositeEventHandler) HandleDevice(device *udev.Device) {
	for _, handler := range c.Handlers {
		// Only process handlers passing the PreCheck
		if handler.PreCheck(device) {
			action := device.Action()
			if handler.RemoveCheck(device, action) {
				handler.OnRemove()
			} else if handler.AddCheck(device, action) {
				handler.OnAdd()
			}
		}
	}
}
