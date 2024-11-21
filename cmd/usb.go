package cmd

import (
	"fmt"
	"log"
	"sync"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

func newUSBHandlers(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient) []*hwcontrol.EventHandler {
	handlers := hwcontrol.NewBasicUSBHandlers()

	handlers[0].SetProcessor(wg, "Starting USB playback", func (device *udev.Device) error {
		if err := mpdClient.StartDiscPlayback(); err != nil {
			log.Printf("Error starting playback: %w", err)
			return fmt.Errorf("Error starting playback: %w", err)
		}
		return nil
	})
	handlers[1].SetProcessor(wg, "Stopping USB playback", func(device *udev.Device) error {
		if err := mpdClient.StopDiscPlayback(); err != nil {
			log.Printf("Error stopping playback: %w", err)
			return fmt.Errorf("Error stopping playback: %w", err)
		}
		return nil
	})
	return handlers
}
