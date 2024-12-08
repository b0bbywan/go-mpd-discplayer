package notifications

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/b0bbywan/go-mpd-discplayer/config"
)

const (
	EventAdd    = "add"
	EventRemove = "remove"
	EventError  = "error"
)

var soundPaths = map[string]string{
	EventAdd:    filepath.Join(config.SoundsLocation, "in.mp3"),
	EventRemove: filepath.Join(config.SoundsLocation, "out.mp3"),
	EventError:  filepath.Join(config.SoundsLocation, "error.mp3"),
}

type Notifier struct {
	player Player
}

type Player interface {
	Play(name string) error
}

// NewRootNotifier creates a new instance of RootNotifier.
func NewNotifier() *Notifier {
	var player Player
	var err error
	switch config.AudioBackend {
	case "pulse":
		player, err = NewPulseAudioPlayer(soundPaths)
	case "alsa":
		player, err = NewOtoPlayer(soundPaths)
	case "none":
		log.Printf("Notifications disabled\n")
		return nil
	default:
		err = fmt.Errorf("Unsupported option")
	}
	if err != nil {
		log.Printf("Failed to initialize player for backend %s: %v\nNotifications disabled", config.AudioBackend, err)
		return nil
	}
	log.Printf("Root notifier initialized")
	return &Notifier{
		player: player,
	}
}

func (n *Notifier) PlayEvent(event string) {
	n.play(event)
}

func (n *Notifier) PlayError() {
	n.PlayEvent(EventError)
}

func (n *Notifier) play(name string) {
	go func() {
		if err := n.player.Play(name); err != nil {
			log.Printf("Failed to play sound (%s): %v", name, err)
		}
	}()
}
