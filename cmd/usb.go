package cmd

import (
	"fmt"
	"log"
	"sync"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/mounts"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

func newUSBHandlers(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient, config *mounts.MountConfig, notifier *notifications.Notifier) []*hwcontrol.EventHandler {
	handlers := hwcontrol.NewBasicUSBHandlers()
	mounter, err := mounts.NewMountManager(config, mpdClient)
	if err != nil {
		log.Printf("USB Playback disabled: Failed to create mount manager: %v\n", err)
		return nil
	}
	startUSBPlayback := func(device *udev.Device) error {
		relPath, err := mounter.Mount(device)
		if err != nil {
			return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[0].Name(), device.Devnode(), err)
		}
		if err = mpdClient.StartUSBPlayback(relPath); err != nil {
			return fmt.Errorf("[%s] Error starting %s:%s USB playback: %w", handlers[0].Name(), device.Devnode(), relPath, err)
		}
		return nil
	}

	stopUSBPlayback := func(device *udev.Device) error {
		relPath, err := mounter.Unmount(device)
		if err != nil {
			return fmt.Errorf("[%s] Error getting mount point for %s: %w", handlers[1].Name(), device.Devnode(), err)
		}
		if err = mpdClient.StopPlayback(relPath); err != nil {
			return fmt.Errorf("[%s] Error stopping %s USB playback: %w", handlers[1].Name(), device.Devnode(), err)
		}
		return nil
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
