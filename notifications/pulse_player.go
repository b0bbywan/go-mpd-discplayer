package notifications

import (
	"bytes"
	"fmt"
	"log"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

type PulseAudioPlayer struct {
	sc     *SoundCache
	client *pulse.Client
}

func NewPulseAudioPlayer(sc *SoundCache, pulseServerString string) (*PulseAudioPlayer, error) {
	pulseServer := pulse.ClientServerString(pulseServerString)
	client, err := pulse.NewClient(pulseServer)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PulseAudio: %w", err)
	}

	return &PulseAudioPlayer{
		sc:     sc,
		client: client,
	}, nil
}

// Play plays the sound corresponding to the given name.
func (p *PulseAudioPlayer) Play(name string) error {
	sound, err := p.sc.Get(name)
	if err != nil {
		return fmt.Errorf("Could not play %s: %w", name, err)
	}

	// Créer un reader depuis les données en cache
	reader := pulse.NewReader(
		bytes.NewReader(sound.Data),
		proto.FormatInt16LE,
	)

	// Use PulseAudio's NewPlayback with the sound data as a reader
	stream, err := p.client.NewPlayback(
		reader,
		pulse.PlaybackStereo,
		pulse.PlaybackSampleRate(44100),
		pulse.PlaybackLatency(0.05),
	)
	if err != nil {
		return fmt.Errorf("failed to create PulseAudio playback stream: %w", err)
	}
	defer stream.Close()

	// Start the stream and wait for it to finish
	stream.Start()
	stream.Drain()
	if stream.Underflow() {
		log.Println("Audio stream underflow detected")
	}

	return nil
}

// Close cleans up the PulseAudio connection.
func (p *PulseAudioPlayer) Close() {
	p.client.Close()
	p.sc.Close()
}
