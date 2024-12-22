package notifications

import (
	"fmt"
	"io"
	"time"

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
	data, err := p.sc.Get(name)
	if err != nil {
		return fmt.Errorf("Could not play %s: %w", name, err)
	}

	reader := pulse.NewReader(data, proto.FormatInt16LE)
	// Use PulseAudio's NewPlayback with the sound data as a reader
	stream, err := p.client.NewPlayback(reader, pulse.PlaybackStereo)
	if err != nil {
		return fmt.Errorf("failed to create PulseAudio playback stream: %w", err)
	}
	defer stream.Close()
/*	p.sc.mu.Lock() // Ensure no concurrent writes to the stream
	defer p.sc.mu.Unlock()
*/
	// Start the stream and wait for it to finish
	done := make(chan struct{})
	go func() {
		stream.Start()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
	}

	stream.Drain()

	// Reset the audio stream to the beginning for future playback
	if _, err := data.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to reset %s after playing: %w", name, err)
	}

	return nil
}

// Close cleans up the PulseAudio connection.
func (p *PulseAudioPlayer) Close() {
	p.client.Close()
	p.sc.Close()
}
