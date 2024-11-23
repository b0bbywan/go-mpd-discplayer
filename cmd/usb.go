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

	handlers[0].SetProcessor(
		wg,
		fmt.Sprintf("[%s] Starting USB playback", handlers[0].Name()),
		func (device *udev.Device) error {
			mountPoint, err := hwcontrol.FindMountPointAndAddtoCache(device.Devnode())
			if err != nil {
				log.Printf("[%s] Error getting mount point for %s: %w", handlers[0].Name(), device.Devnode(), err)
				return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[0].Name(), device.Devnode(), err)
			}
			if err := mpdClient.StartUSBPlayback(filepath.Base(mountPoint)); err != nil {
				log.Printf("[%s] Error starting USB playback: %w", handlers[0].Name(), err)
				return fmt.Errorf("[%s] Error starting USB playback: %w", handlers[0].Name(), err)
			}
			return nil
		},
	)

	handlers[1].SetProcessor(
		wg,
		fmt.Sprintf("[%s] Stopping USB playback", handlers[1].Name()),
		func(device *udev.Device) error {
			mountPoint, err := hwcontrol.SeekMountPointAndClearCache(device.Devnode())
			if err != nil {
				log.Printf("[%s] Error getting mount point for %s: %w", handlers[1].Name(), device.Devnode(), err)
				return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[1].Name(), device.Devnode(), err)
			}
			if err := mpdClient.StopPlayback(filepath.Base(mountPoint)); err != nil {
				log.Printf("[%s] Error stopping USB playback: %w", handlers[1].Name(), err)
				return fmt.Errorf("[%s] Error stopping USB playback: %w", handlers[1].Name(), err)
			}
			return nil
		},
	)

	return handlers
}
