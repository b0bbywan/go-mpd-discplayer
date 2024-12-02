package notifications

import (
	"fmt"
	"path/filepath"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

var soundPaths = map[string]string{
	"add":    filepath.Join(config.SoundsLocation, "in.wav"),
	"remove": filepath.Join(config.SoundsLocation, "out.wav"),
	"error":  filepath.Join(config.SoundsLocation, "error.wav"),
}

type Notifier struct {
	sc       *SoundCache
	success string
	error   string
}

func newNotifier(sc *SoundCache, sucessSound, errorSound string) *Notifier {
	return &Notifier{
		sc:       sc,
		success:  sucessSound,
		error:    errorSound,
	}
}

func (n *Notifier) PlaySuccess() {
	n.play(n.success)
}

func (n *Notifier) PlayError() {
	n.play(n.error)
}

func (n *Notifier) play(name string) error {
	if err := n.sc.Play(name); err != nil {
		return fmt.Errorf("Failed to play sound (%s): %w", name, err)
	}
	return nil
}

type RootNotifier struct {
    AddNotifier   *Notifier
    RemoveNotifier *Notifier
}

// NewRootNotifier creates a new instance of RootNotifier.
func NewRootNotifier() *RootNotifier {
	sc := NewSoundCache(soundPaths)
	return &RootNotifier{
		AddNotifier:    newNotifier(sc, "add", "error"),
		RemoveNotifier: newNotifier(sc, "remove", "error"),
	}
}
