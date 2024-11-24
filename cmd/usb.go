package cmd

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

func newUSBHandlers(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient) []*hwcontrol.EventHandler {
	handlers := hwcontrol.NewBasicUSBHandlers()

	handlers[0].SetProcessor(wg, "Starting USB playback", func (device *udev.Device) error {
		mountPoint, err := hwcontrol.SeekMountPoint(device.Devnode())
		if err != nil {
			log.Printf("Error getting mount point for %s: %w", device.Devnode(), err)
			return fmt.Errorf("Error getting mount point for %s: %w", device.Devnode(), err)
		}
		if err := mpdClient.StartUSBPlayback(filepath.Base(mountPoint)); err != nil {
			log.Printf("Error starting USB playback: %w", err)
			return fmt.Errorf("Error starting USB playback: %w", err)
		}

		return nil
	})
	handlers[1].SetProcessor(wg, "Stopping USB playback", func(device *udev.Device) error {
		mountPoint, err := hwcontrol.SeekMountPoint(device.Devnode())
		if err != nil {
			log.Printf("Error getting mount point for %s: %w", device.Devnode(), err)
			return fmt.Errorf("Error getting mount point for %s: %w", device.Devnode(), err)
		}
		if err := mpdClient.StopPlayback(filepath.Base(mountPoint)); err != nil {
			log.Printf("Error stopping USB playback: %w", err)
			return fmt.Errorf("Error stopping USB playback: %w", err)
		}
		return nil
	})

	return handlers
}
