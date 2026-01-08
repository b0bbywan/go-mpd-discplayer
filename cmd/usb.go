package cmd

import (
	"context"
	"fmt"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
)

func (player *Player) newUSBHandler() {
    usbHandler := hwcontrol.NewBasicHandler(
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

    // Stocker le handler dans Player
    player.handlers = append(player.handlers, usbHandler)
}
