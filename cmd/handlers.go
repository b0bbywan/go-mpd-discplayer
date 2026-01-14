package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
)

// Handler defines a stateless handler capable of handling one type of  device.
type Handler interface {
	// Handles returns true if the handler can handle that type of device
	Handles(kind detect.DeviceKind) bool
	OnAdd(ctx context.Context, dev detect.Device) error
	OnRemove(ctx context.Context, dev detect.Device) error
}

type BasicHandler struct {
	kind detect.DeviceKind
	// processFunc est défini par l’utilisateur
	processAdd    func(context.Context, detect.Device) error
	processRemove func(context.Context, detect.Device) error
}

func (h *BasicHandler) Handles(kind detect.DeviceKind) bool {
	return h.kind == kind
}

func (h *BasicHandler) OnAdd(ctx context.Context, dev detect.Device) error {
	if h.processAdd != nil {
		return h.processAdd(ctx, dev)
	}
	return nil
}

func (h *BasicHandler) OnRemove(ctx context.Context, dev detect.Device) error {
	if h.processRemove != nil {
		return h.processRemove(ctx, dev)
	}
	return nil
}

// Handler Constructor
func NewBasicHandler(kind detect.DeviceKind,
	processAdd, processRemove func(context.Context, detect.Device) error) *BasicHandler {

	return &BasicHandler{
		kind:          kind,
		processAdd:    processAdd,
		processRemove: processRemove,
	}
}

func (player *Player) newDiscHandler() {
	discHandler := NewBasicHandler(
		detect.DeviceDisc,
		// processAdd
		func(ctx context.Context, dev detect.Device) error {
			if err := hwcontrol.SetDiscSpeed(dev.Path(), player.discSpeed); err != nil {
				log.Printf("[%s] Error setting disc speed on %s: %v", detect.DeviceDisc, dev.Path(), err)
			}
			if err := player.Client.StartDiscPlayback(dev.Path()); err != nil {
				return fmt.Errorf("[%s] Error starting %s playback: %w", detect.DeviceDisc, dev.Path(), err)
			}
			return nil
		},
		// processRemove
		func(ctx context.Context, dev detect.Device) error {
			if err := player.Client.StopDiscPlayback(); err != nil {
				return fmt.Errorf("[%s] Error stopping %s playback: %w", detect.DeviceDisc, dev.Path(), err)
			}
			return nil
		},
	)

	player.handlers = append(player.handlers, discHandler)
}

func (player *Player) newUSBHandler() {
	usbHandler := NewBasicHandler(
		detect.DeviceUSB,
		// processAdd
		func(ctx context.Context, dev detect.Device) error {
			relPath, err := player.Mounter.Mount(dev.Udev())
			if err != nil {
				return fmt.Errorf("[%s] Error getting mount point for %s: %w", detect.DeviceUSB, dev.Path(), err)
			}
			if err = player.Client.StartUSBPlayback(relPath); err != nil {
				return fmt.Errorf("[%s] Error starting %s:%s USB playback: %w", detect.DeviceUSB, dev.Path(), relPath, err)
			}
			return nil
		},
		// processRemove
		func(ctx context.Context, dev detect.Device) error {
			relPath, err := player.Mounter.Unmount(dev.Udev())
			if err != nil {
				return fmt.Errorf("[%s] Error getting mount point for %s: %w", detect.DeviceUSB, dev.Path(), err)
			}
			if err = player.Client.StopPlayback(relPath); err != nil {
				return fmt.Errorf("[%s] Error stopping %s USB playback: %w", detect.DeviceUSB, dev.Path(), err)
			}
			return nil
		},
	)

	player.handlers = append(player.handlers, usbHandler)
}
