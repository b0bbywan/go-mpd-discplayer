package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
)

// Handler defines a stateless handler capable of handling one type of  device.
type Handler interface {
	// Handles returns true if the handler can handle that type of device
	Handles(kind detect.DeviceKind) bool
	OnAdd(ctx context.Context, dev detect.Device) error
	OnRemove(ctx context.Context, dev detect.Device) error
	// OnPresent handles a device already plugged in at startup
	OnPresent(ctx context.Context, dev detect.Device) error
}

type BasicHandler struct {
	kind detect.DeviceKind
	// processFunc est défini par l’utilisateur
	processAdd     func(context.Context, detect.Device) error
	processRemove  func(context.Context, detect.Device) error
	processPresent func(context.Context, detect.Device) error
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

func (h *BasicHandler) OnPresent(ctx context.Context, dev detect.Device) error {
	if h.processPresent != nil {
		return h.processPresent(ctx, dev)
	}
	return nil
}

// Handler Constructor
func NewBasicHandler(kind detect.DeviceKind,
	processAdd, processRemove, processPresent func(context.Context, detect.Device) error) *BasicHandler {

	return &BasicHandler{
		kind:           kind,
		processAdd:     processAdd,
		processRemove:  processRemove,
		processPresent: processPresent,
	}
}

func (player *Player) newDiscHandler() {
	discHandler := NewBasicHandler(
		detect.DeviceDisc,
		// processAdd
		func(ctx context.Context, dev detect.Device) error {
			if err := player.insertDisc(dev.Path()); err != nil {
				return fmt.Errorf("[%s] Error starting %s playback: %w", detect.DeviceDisc, dev.Path(), err)
			}
			return nil
		},
		// processRemove
		func(ctx context.Context, dev detect.Device) error {
			player.removeDisc()
			if err := player.Client.StopAndRemove(player.Client.DiscPrefixes()...); err != nil {
				return fmt.Errorf("[%s] Error stopping %s playback: %w", detect.DeviceDisc, dev.Path(), err)
			}
			return nil
		},
		// processPresent: keep the disc in the state, no autoplay at startup
		func(ctx context.Context, dev detect.Device) error {
			if err := player.presentDisc(dev.Path()); err != nil {
				return fmt.Errorf("[%s] Error reading %s at startup: %w", detect.DeviceDisc, dev.Path(), err)
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
			relPath, err := player.mountUSB(dev)
			if err != nil {
				return err
			}
			player.refreshMounts()
			if err = player.Client.StartUSBPlayback(relPath); err != nil {
				return fmt.Errorf("[%s] Error starting %s:%s USB playback: %w", detect.DeviceUSB, dev.Path(), relPath, err)
			}
			return nil
		},
		// processRemove
		func(ctx context.Context, dev detect.Device) error {
			relPath, err := player.Mounter.Unmount(dev.Udev())
			player.refreshMounts()
			if err != nil {
				return fmt.Errorf("[%s] Error getting mount point for %s: %w", detect.DeviceUSB, dev.Path(), err)
			}
			if err = player.Client.StopAndRemove(relPath); err != nil {
				return fmt.Errorf("[%s] Error stopping %s USB playback: %w", detect.DeviceUSB, dev.Path(), err)
			}
			return nil
		},
		// processPresent: mount and refresh the library, without touching the queue
		func(ctx context.Context, dev detect.Device) error {
			relPath, err := player.mountUSB(dev)
			if err != nil {
				return err
			}
			player.refreshMounts()
			if err = player.Client.UpdateDB(relPath); err != nil {
				return fmt.Errorf("[%s] Error updating database for %s:%s: %w", detect.DeviceUSB, dev.Path(), relPath, err)
			}
			log.Printf("[%s] %s available at %s", detect.DeviceUSB, dev.Path(), relPath)
			return nil
		},
	)

	player.handlers = append(player.handlers, usbHandler)
}

// mountUSB exposes dev to MPD and returns its path in the library.
func (player *Player) mountUSB(dev detect.Device) (string, error) {
	relPath, err := player.Mounter.Mount(dev.Udev())
	if err != nil {
		return "", fmt.Errorf("[%s] Error getting mount point for %s: %w", detect.DeviceUSB, dev.Path(), err)
	}
	return relPath, nil
}
