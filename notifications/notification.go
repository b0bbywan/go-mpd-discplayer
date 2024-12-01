package notifications

import (
	"log"
	"path/filepath"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

var soundPaths = map[string]string{
	"add":    filepath.Join(config.SoundsLocation, "in.wav"),
	"remove": filepath.Join(config.SoundsLocation, "out.wav"),
	"error":  filepath.Join(config.SoundsLocation, "error.wav"),
}

type Notifier struct {
	notifier *RootNotifier
	success  string
	error    string
}

func newNotifier(notifier *RootNotifier, sucessSound, errorSound string) *Notifier {
	return &Notifier{
		notifier: notifier,
		success:  sucessSound,
		error:    errorSound,
	}
}

func (n *Notifier) PlaySuccess() {
	n.notifier.Play(n.success)
}

func (n *Notifier) PlayError() {
	n.notifier.Play(n.error)
}

func NewAddNotification(notifier *RootNotifier) *Notifier {
	return newNotifier(notifier, "add", "error")
}

func NewRemoveNotification(notifier *RootNotifier) *Notifier {
	return newNotifier(notifier, "remove", "error")
}

type RootNotifier struct {
	sc *SoundCache
}

// NewRootNotifier creates a new instance of RootNotifier.
func NewRootNotifier() *RootNotifier {
	return &RootNotifier{
		sc: NewSoundCache(soundPaths),
	}
}

func (n *RootNotifier) Play(name string) {
	go func() {
		if err := n.sc.Play(name); err != nil {
			log.Printf("Failed to play sound (%s): %v", name, err)
		}
	}()
}
