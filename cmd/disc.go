package cmd

import (
	"fmt"
	"log"
	"sync"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

func newDiscHandlers(wg *sync.WaitGroup, player *Player) []*hwcontrol.EventHandler {
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

	// Define action for the "add" event (handler[0])
	handlers[0].SetProcessor(
		wg,
		fmt.Sprintf("[%s] Starting Disc Playback", handlers[0].Name()),
		startDiscPlayback,
		player.Notifier,
		notifications.EventAdd,
	)

	// Define action for the "remove" event (handler[1])
	handlers[1].SetProcessor(
		wg,
		fmt.Sprintf("[%s] Stopping Disc Playback", handlers[1].Name()),
		stopDiscPlayback,
		player.Notifier,
		notifications.EventRemove,
	)

	return handlers
}
