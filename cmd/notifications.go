package cmd

import (
	"path/filepath"

	"github.com/b0bbywan/go-mpd-discplayer/config"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

var (
	addSound    = filepath.Join(config.SoundsLocation, "in.wav")
	removeSound = filepath.Join(config.SoundsLocation, "out.wav")
	errorSound  = filepath.Join(config.SoundsLocation, "error.wav")
)

func newAddNotification() *notifications.SoundNotifier {
	return notifications.NewSoundNotifier(addSound, errorSound)
}

func newRemoveNotification() *notifications.SoundNotifier {
	return notifications.NewSoundNotifier(removeSound, errorSound)
}
