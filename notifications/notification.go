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
	Player
}

type Player interface {
	Play(name string) error
	Close()
}

// NewRootNotifier creates a new instance of RootNotifier.
func NewNotifier() *Notifier {
	sc, err := NewSoundCache(soundPaths)
	if err != nil {
		log.Printf("Failed to load sound cache: %v", err)
		return nil
	}

	var player Player
	switch config.AudioBackend {
	case "alsa":
		player, err = NewOtoPlayer(sc)
	case "pulse":
		player, err = NewPulseAudioPlayer(sc)
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

	log.Printf("%s notifier initialized\n", config.AudioBackend)

	return &Notifier{
		player,
	}
}

func (n *Notifier) PlayEvent(event string) {
	n.play(event)
}

func (n *Notifier) PlayError() {
	n.PlayEvent(EventError)
}

func (n *Notifier) Close() {
	if n != nil && n.Player != nil {
		n.Player.Close()
	}
}

func (n *Notifier) play(name string) {
	go func() {
		if err := n.Play(name); err != nil {
			log.Printf("Failed to play sound (%s): %v", name, err)
		}
	}()
}
