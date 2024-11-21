package hwcontrol

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jochenvg/go-udev"
)

type EventHandler struct {
	name				string
	deviceFilterFunc	func(*udev.Device) bool
	actionChan			chan *udev.Device
	processFunc			func(*udev.Device) error // Action to execute
}

func (h *EventHandler) DeviceFilter(device *udev.Device) bool {
	if h.deviceFilterFunc != nil && device != nil {
		return h.deviceFilterFunc(device)
	}
	return false
}

func (h *EventHandler) SetProcessor(wg *sync.WaitGroup, actionLog string, processor func(device *udev.Device) error) {
	h.processFunc = func(device *udev.Device) error {
		log.Println(actionLog)
		wg.Add(1) // Increment the counter before starting the task
		go func() {
			defer wg.Done()
			if err := processor(device); err != nil {
				fmt.Errorf("(%s) Failed to process action: %w", h.Name(), err)
				return
			}
		}()
		return nil
	}
}

func (h *EventHandler) Process(device *udev.Device) error {
	if h.processFunc != nil && device != nil {
		if err := h.processFunc(device); err != nil {
			log.Printf("(%s) Failed to process device %s: %w", h.Name(), device.Sysname(), err)
			return fmt.Errorf("(%s) Failed to process device %s: %w", h.Name(), device.Sysname(), err)
		}
	}
	return nil
}

func (h *EventHandler) Name() string {
	return h.name
}

func StartMonitor(ctx context.Context, handlers []*EventHandler) error {
	u := udev.Udev{}
	monitor := u.NewMonitorFromNetlink("udev")
	if err := monitor.FilterAddMatchSubsystem("block"); err != nil {
		log.Printf("Failed to add filter: %w", err)
		return fmt.Errorf("Failed to add filter: %w", err)
	}

	// Start the monitor and get the device channel
	deviceChan, errChan, err := monitor.DeviceChan(ctx)
	if err != nil {
		log.Printf("failed to create device channel: %w", err)
		return fmt.Errorf("failed to create device channel: %w", err)
	}

	log.Println("Listening for udev events...")

	startEventPublisher(ctx, deviceChan, errChan, handlers)
	return nil
}

func newBasicHandler(
	name string,
	preChecker func(*udev.Device) bool,
	actionChecker func(*udev.Device, string) bool,
) *EventHandler {
	return &EventHandler{
		name: name,
		deviceFilterFunc: func(device *udev.Device) bool {
			return preChecker(device) && actionChecker(device, device.Action())
		},
		actionChan: make(chan *udev.Device),
	}
}

func startEventPublisher(ctx context.Context, deviceChan <-chan *udev.Device, errChan <-chan error, handlers []*EventHandler) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Publisher stopping...")
			return
		case device := <-deviceChan:
			for _, handler := range handlers {
				if handler.DeviceFilter(device) {
					handler.actionChan <- device
				}
			}
		case err := <-errChan:
			if err != nil {
				log.Printf("Received error from monitor: %w", err)
			}
		}
	}
}

func (h *EventHandler) StartSubscriber(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic: %v", r)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				log.Printf("(%s) Subscriber stopping...", h.Name())
				close(h.actionChan)  // Close the channel on shutdown
				return
			case device := <-h.actionChan:
				// Handle the event
				if err := h.Process(device); err != nil {
					log.Printf("(%s) Failed to process device %s: %w", h.Name(), device.Sysname(), err)
					continue
				}
			}
		}
	}()
}
