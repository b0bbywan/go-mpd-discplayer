package notifications

import (
	"fmt"
	"log"
	"path/filepath"
)

const (
	EventAdd    = "add"
	EventRemove = "remove"
	EventError  = "error"
)

type Notifier struct {
	Player
}

type NotificationConfig struct {
	AudioBackend string
	SoundPaths   map[string]string
	PulseServer  string
}

type Player interface {
	Play(name string) error
	Close()
}

func NewNotificationConfig(audioBackend, pulseServer, soundsLocation string) *NotificationConfig {
	return &NotificationConfig{
		AudioBackend: audioBackend,
		PulseServer:  pulseServer,
		SoundPaths: map[string]string{
			EventAdd:    filepath.Join(soundsLocation, "in.mp3"),
			EventRemove: filepath.Join(soundsLocation, "out.mp3"),
			EventError:  filepath.Join(soundsLocation, "error.mp3"),
		},
	}
}

// NewRootNotifier creates a new instance of RootNotifier.
func NewNotifier(config *NotificationConfig) *Notifier {
	sc, err := NewSoundCache(config.SoundPaths)
	if err != nil {
		log.Printf("Failed to load sound cache: %v", err)
		return nil
	}

	var player Player
	switch config.AudioBackend {
	case "alsa":
		player, err = NewOtoPlayer(sc)
	case "pulse":
		player, err = NewPulseAudioPlayer(sc, config.PulseServer)
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
	if n != nil {
		n.play(event)
	}
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
