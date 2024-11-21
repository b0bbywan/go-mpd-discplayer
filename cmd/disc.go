package cmd

import (
	"fmt"
	"log"
	"sync"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

func newDiscHandlers(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient) []*hwcontrol.EventHandler {
	// Use VeryNewBasicDiscHandler to create the event handlers
	handlers := hwcontrol.NewBasicDiscHandlers()

	// Define action for the "add" event (handler[0])
	handlers[0].SetProcessor(wg, "Starting Disc Playback", func(device *udev.Device) error {
		if err := mpdClient.StartDiscPlayback(device.Devnode()); err != nil {
			log.Printf("Error starting playback: %w", err)
			return fmt.Errorf("Error starting playback: %w", err)
		}
		return nil
	})

	// Define action for the "remove" event (handler[1])
	handlers[1].SetProcessor(wg, "Stopping Disc Playback", func(device *udev.Device) error {
		if err := mpdClient.StopDiscPlayback(); err != nil {
			log.Printf("Error stopping playback: %w", err)
			return fmt.Errorf("Error stopping playback: %w", err)
		}
		return nil
	})

	return handlers
}
