package cmd

import (
	"fmt"
	"log"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

func (player *Player) newDiscHandlers() {
	// Use VeryNewBasicDiscHandler to create the event handlers
	handlers := hwcontrol.NewBasicDiscHandlers()

	startDiscPlayback := func(device *udev.Device) error {
		if err := hwcontrol.SetDiscSpeed(device.Devnode(), player.GetDiscSpeed()); err != nil {
			log.Printf("[%s] Error setting disc speed on %s: %v", handlers[0].Name(), device.Devnode(), err)
		}
		if err := player.Client.StartDiscPlayback(device.Devnode()); err != nil {
			return fmt.Errorf("[%s] Error starting %s playback: %w", handlers[0].Name(), device.Devnode(), err)
		}
		return nil
	}

	stopDiscPlayback := func(device *udev.Device) error {
		if err := player.Client.StopDiscPlayback(); err != nil {
			return fmt.Errorf("[%s] Error stopping %s playback: %w", handlers[0].Name(), device.Devnode(), err)
		}
		return nil
	}

	player.SetHandlerProcessor(
		handlers[0],
		startDiscPlayback,
		"Starting Disc Playback",
		notifications.EventAdd,
	)

	player.SetHandlerProcessor(
		handlers[1],
		stopDiscPlayback,
		"Stopping Disc Playback",
		notifications.EventRemove,
	)
}
