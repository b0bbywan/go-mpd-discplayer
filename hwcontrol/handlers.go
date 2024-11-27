package hwcontrol

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jochenvg/go-udev"
)

const (
	EventAdd    = "add"
	EventChange = "change"
	EventRemove = "remove"
)

type EventHandler struct {
	name             string
	deviceFilterFunc func(*udev.Device) bool
	actionChan       chan *udev.Device
	processFunc      func(*udev.Device) error // Action to execute
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
				log.Printf("[%s] Failed to process action: %v", h.Name(), err)
				return
			}
		}()
		return nil
	}
}

func (h *EventHandler) Process(ctx context.Context, device *udev.Device) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("[%s] Skipping device %s due to context cancellation.", h.Name(), device.Devnode())
	default:
		if h.processFunc != nil && device != nil {
			if err := h.processFunc(device); err != nil {
				return fmt.Errorf("[%s] Failed to process device %s: %w", h.Name(), device.Devnode(), err)
			}
		}
		return nil
	}
}

func (h *EventHandler) Name() string {
	return h.name
}

func StartMonitor(ctx context.Context, handlers []*EventHandler) error {
	u := udev.Udev{}
	monitor := u.NewMonitorFromNetlink("udev")
	if err := monitor.FilterAddMatchSubsystem("block"); err != nil {
		return fmt.Errorf("Failed to add filter: %w", err)
	}

	// Start the monitor and get the device channel
	deviceChan, errChan, err := monitor.DeviceChan(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device channel: %w", err)
	}

	log.Println("Listening for udev events...")

	startEventPublisher(ctx, deviceChan, errChan, handlers)
	return nil
}

func (h *EventHandler) StartSubscriber(wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[%s] Recovered from panic: %v", h.Name(), r)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				log.Printf("[%s] Subscriber stopping...", h.Name())
				return
			case device := <-h.actionChan:
				// Handle the event
				log.Printf("[%s] Processing device %s...", h.Name(), device.Devnode())
				wg.Add(1)
				go func(*udev.Device) {
					defer wg.Done()
					if err := h.Process(ctx, device); err != nil {
						log.Printf("[%s] Error processing device %s: %w", h.Name(), device.Devnode(), err)
					}
				}(device)
			}
		}
	}()
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
		actionChan: make(chan *udev.Device, 10),
	}
}

func startEventPublisher(ctx context.Context, deviceChan <-chan *udev.Device, errChan <-chan error, handlers []*EventHandler) {
	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			closeEventPublisher(&wg, handlers)
			return
		case device := <-deviceChan:
			for _, handler := range handlers {
				wg.Add(1)
				go filterDevice(&wg, ctx, handler, device)
			}
		case err := <-errChan:
			if err != nil {
				log.Printf("Received error from monitor: %w", err)
			}
		}
	}
}

func closeEventPublisher(wg *sync.WaitGroup, handlers []*EventHandler) {
	for _, handler := range handlers {
		log.Printf("[%s] Publisher stopping...", handler.Name())
		close(handler.actionChan)
	}
	wg.Wait()
}

func filterDevice(wg *sync.WaitGroup, ctx context.Context, h *EventHandler, d *udev.Device) {
	defer wg.Done()
	if h.DeviceFilter(d) {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Device %s filter canceled by context.", h.Name(), d.Devnode())
		case h.actionChan <- d:
			log.Printf("[%s] Device %s passed filter and sent to handler.", h.Name(), d.Devnode())
		}
	}
}
