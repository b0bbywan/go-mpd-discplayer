package notifications

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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

	if _, err := data.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to reset %s before playing: %w", name, err)
	}

	reader := pulse.NewReader(data, proto.FormatInt16LE)
	// Use PulseAudio's NewPlayback with the sound data as a reader
	stream, err := p.client.NewPlayback(reader, pulse.PlaybackStereo)
	if err != nil {
		return fmt.Errorf("failed to create PulseAudio playback stream: %w", err)
	}
	defer stream.Close()

	// Create an error channel for the playback result
	errChan := make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	// Start the playback and wait for the result
	go p.waitForStreamCompletion(ctx, stream, errChan)

	var streamErr error
	var ok bool
	// Wait for playback to complete or timeout
	select {
	case err, ok = <-errChan: // Receive any error from the channel
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			streamErr = fmt.Errorf("playback error for %s: %w", name, err)
		}
	case <-ctx.Done(): // Timeout or cancellation case
		stream.Stop()
		if err := stream.Error(); err != nil {
			streamErr = fmt.Errorf("playback timedout with error: %w", err)
		}
	}
	if !ok {
		close(errChan)
	}
	if streamErr != nil  {
		return streamErr
	}
	return nil
}

// Close cleans up the PulseAudio connection.
func (p *PulseAudioPlayer) Close() {
	p.client.Close()
	p.sc.Close()
}

func (p *PulseAudioPlayer) waitForStreamCompletion(ctx context.Context, stream *pulse.PlaybackStream, errChan chan<- error) {
	go func() {
		select {
		case <-ctx.Done(): // Listen for context cancellation or timeout
			return
		default:
			stream.Start()

			// Wait for the stream to finish
			stream.Drain()

			if stream.Underflow() {
				log.Printf("Underflow occurred during playback")
			}

			// Send any stream error or nil (on success) to the error channel
			if err := stream.Error(); err != nil {
				errChan <- fmt.Errorf("stream playback error: %w", err)
			} else {
				close(errChan) // Close the channel to indicate successful completion
			}
		}
	}()
}
