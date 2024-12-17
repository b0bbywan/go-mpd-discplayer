package cmd

import (
	"fmt"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

func (player *Player) newUSBHandlers() {
	handlers := hwcontrol.NewBasicUSBHandlers()
	//TODO check mounter nil
	startUSBPlayback := func(device *udev.Device) error {
		relPath, err := player.Mounter.Mount(device)
		if err != nil {
			return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[0].Name(), device.Devnode(), err)
		}
		if err = player.Client.StartUSBPlayback(relPath); err != nil {
			return fmt.Errorf("[%s] Error starting %s:%s USB playback: %w", handlers[0].Name(), device.Devnode(), relPath, err)
		}
		return nil
	}

	stopUSBPlayback := func(device *udev.Device) error {
		relPath, err := player.Mounter.Unmount(device)
		if err != nil {
			return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[1].Name(), device.Devnode(), err)
		}
		if err = player.Client.StopPlayback(relPath); err != nil {
			return fmt.Errorf("[%s] Error stopping %s USB playback: %w", handlers[1].Name(), device.Devnode(), err)
		}
		return nil
	}

	player.SetHandlerProcessor(
		handlers[0],
		startUSBPlayback,
		"Starting USB playback",
		notifications.EventAdd,
	)

	player.SetHandlerProcessor(
		handlers[1],
		stopUSBPlayback,
		"Stopping USB playback",
		notifications.EventRemove,
	)
}
