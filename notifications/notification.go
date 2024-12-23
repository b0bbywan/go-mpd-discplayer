package notifications

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
)

const (
	EventAdd    = "add"
	EventRemove = "remove"
	EventError  = "error"
)

type Notifier struct {
	Player
	queue       chan string
	stop        chan struct{}
	wg          sync.WaitGroup
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
	n := &Notifier{
		Player: player,
		queue: make(chan string, 3), // Adjust buffer size as needed
		stop:  make(chan struct{}),
	}
	n.wg.Add(1)
	go n.processQueue()

	return n

}

func (n *Notifier) processQueue() {
	defer n.wg.Done()
	for {
		select {
		case sound := <-n.queue:
			log.Printf("Playing %s", sound)
			if err := n.Play(sound); err != nil {
				log.Printf("Failed to play sound (%s): %v", sound, err)
			}
		case <-n.stop:
			return
		}
	}
}

func (n *Notifier) PlayEvent(event string) {
	select {
	case n.queue <- event:
		// Sound successfully queued
	default:
		log.Printf("Sound queue is full; dropping event: %s", event)
	}
}

func (n *Notifier) PlayError() {
	n.PlayEvent(EventError)
}

func (n *Notifier) Close() {
	if n != nil {
		close(n.stop)
		n.wg.Wait() // Wait for the processing goroutine to finish
		if n.Player != nil {
			n.Player.Close()
		}
	}
}
