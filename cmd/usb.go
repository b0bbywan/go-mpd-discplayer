package cmd

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

func newUSBHandlers(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient, notifier *notifications.RootNotifier) []*hwcontrol.EventHandler {
	handlers := hwcontrol.NewBasicUSBHandlers()

	startUSBPlayback := func(device *udev.Device) error {
		mountPoint, err := hwcontrol.FindMountPointAndAddtoCache(device.Devnode())
		if err != nil {
			return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[0].Name(), device.Devnode(), err)
		}
		if err := mpdClient.StartUSBPlayback(filepath.Base(mountPoint)); err != nil {
			return fmt.Errorf("[%s] Error starting %s USB playback: %w", handlers[0].Name(), device.Devnode(), err)
		}
		return nil
	}

	stopUSBPlayback := func(device *udev.Device) error {
		mountPoint, err := hwcontrol.SeekMountPointAndClearCache(device.Devnode())
		if err != nil {
			return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[1].Name(), device.Devnode(), err)
		}
		if err := mpdClient.StopPlayback(filepath.Base(mountPoint)); err != nil {
			return fmt.Errorf("[%s] Error stopping %s USB playback: %w", handlers[1].Name(), device.Devnode(), err)
		}
		return nil
	}

	handlers[0].SetProcessor(
		wg,
		fmt.Sprintf("[%s] Starting USB playback", handlers[0].Name()),
		startUSBPlayback,
		notifications.NewAddNotification(notifier),
	)

	handlers[1].SetProcessor(
		wg,
		fmt.Sprintf("[%s] Stopping USB playback", handlers[1].Name()),
		stopUSBPlayback,
		notifications.NewRemoveNotification(notifier),
	)

	return handlers
}
