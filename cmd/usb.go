package cmd

import (
	"sync"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

func newUSBHandlers(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient) []*hwcontrol.EventHandler {
	handlers := hwcontrol.NewBasicUSBHandlers()

	handlers[0].SetProcessor(wg, "Starting USB playback", func (device *udev.Device) error {
		return nil
	})
	handlers[1].SetProcessor(wg, "Stopping USB playback", func(device *udev.Device) error {
		return nil
	})

	return handlers
}
