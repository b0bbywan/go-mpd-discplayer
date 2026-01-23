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

func (p *PulseAudioPlayer) Play(name string) error {
	entry, err := p.sc.Get(name)
	if err != nil {
		return fmt.Errorf("Could not play %s: %w", name, err)
	}

	reader := pulse.NewReader(
		bytes.NewReader(entry.PCM),
		proto.FormatInt16LE,
	)

	stream, err := p.client.NewPlayback(
		reader,
		pulse.PlaybackStereo,
		pulse.PlaybackSampleRate(44100),
		pulse.PlaybackLatency(0.5),
	)

	if err != nil {
		return fmt.Errorf("failed to create PulseAudio playback stream: %w", err)
	}
	defer stream.Close()

	stream.Start()
	stream.Drain()

	if stream.Underflow() {
		log.Printf("Audio underflow detected for %s", name)
	}

	return nil
}

func (p *PulseAudioPlayer) Close() {
	p.client.Close()
	p.sc.Close()
}
