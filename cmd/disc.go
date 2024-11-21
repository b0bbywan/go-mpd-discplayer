package cmd

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/config"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

func newDiscHandlers(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient) []*hwcontrol.EventHandler {
	// Use VeryNewBasicDiscHandler to create the event handlers
	handlers := hwcontrol.NewBasicDiscHandlers(filepath.Base(config.TargetDevice))

	// Define action for the "add" event (handler[0])
	handlers[0].SetProcessor(wg, "Starting Disc Playback", func(device *udev.Device) error {
/*		if err := hwcontrol.SetDiscSpeed(config.TargetDevice, config.DiscSpeed); err != nil {
			log.Printf("Failed to set disc speed: %w", err)
		}
*/		if err := mpdClient.StartDiscPlayback(); err != nil {
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
