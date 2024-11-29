package cmd

import (
	"fmt"
	"sync"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/mounts"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

func newUSBHandlers(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient, notifier *notifications.Notifier) []*hwcontrol.EventHandler {
	handlers := hwcontrol.NewBasicUSBHandlers()
	mounter := mounts.NewMountManager()
	startUSBPlayback := func(device *udev.Device) error {
		relPath, err := mounter.FindRelPath(device.Devnode())
		if err != nil {
			return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[0].Name(), device.Devnode(), err)
		}
		if err := mpdClient.StartUSBPlayback(relPath); err != nil {
			return fmt.Errorf("[%s] Error starting %s:%s USB playback: %w", handlers[0].Name(), device.Devnode(), relPath, err)
		}
		return nil
	}

	stopUSBPlayback := func(device *udev.Device) error {
/*		relPath, err := hwcontrol.FindRelPath(device.Devnode(), hwcontrol.SeekMountPointAndClearCache)
		if err != nil {
			return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[1].Name(), device.Devnode(), err)
		}
		if err := mpdClient.StopPlayback(relPath); err != nil {
			return fmt.Errorf("[%s] Error stopping %s USB playback: %w", handlers[1].Name(), device.Devnode(), err)
		}
*/		return nil
	}

	handlers[0].SetProcessor(
		wg,
		fmt.Sprintf("[%s] Starting USB playback", handlers[0].Name()),
		startUSBPlayback,
		notifier,
		notifications.EventAdd,
	)

	handlers[1].SetProcessor(
		wg,
		fmt.Sprintf("[%s] Stopping USB playback", handlers[1].Name()),
		stopUSBPlayback,
		notifier,
		notifications.EventRemove,
	)

	return handlers
}
