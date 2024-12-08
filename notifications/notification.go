package notifications

import (
	"log"
	"path/filepath"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

var soundPaths = map[string]string{
	"add":    filepath.Join(config.SoundsLocation, "in.mp3"),
	"remove": filepath.Join(config.SoundsLocation, "out.mp3"),
	"error":  filepath.Join(config.SoundsLocation, "error.mp3"),
}

type Notifier struct {
	notifier *RootNotifier
	success  string
	error    string
}

type Player interface {
	Play(name string) error
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
	player Player
}

// NewRootNotifier creates a new instance of RootNotifier.
func NewRootNotifier() *RootNotifier {
	otoPlayer, err := NewOtoPlayer(soundPaths)
	if err != nil {
		log.Printf("Failed to create RootNotifier: %w", err)
		return nil
	}
	log.Printf("Root notifier initialized")
	return &RootNotifier{
		player: otoPlayer,
	}
}

func (n *RootNotifier) Play(name string) {
	go func() {
		if err := n.player.Play(name); err != nil {
			log.Printf("Failed to play sound (%s): %v", name, err)
		}
	}()
}
