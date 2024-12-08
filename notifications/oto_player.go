package notifications

import (
	"fmt"
	"io"
	"time"

	"github.com/hajimehoshi/oto/v2"
)

type OtoPlayer struct {
	sc     *SoundCache
	otoCtx *oto.Context
}

func NewOtoPlayer(soundsPath map[string]string) (*OtoPlayer, error) {
	sc, err := NewSoundCache(soundsPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to load sound cache: %w", err)
	}
	otoCtx, ready, err := oto.NewContext(44100, 2, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize oto: %w", err)
	}
	<-ready

	return &OtoPlayer{
		sc:     sc,
		otoCtx: otoCtx,
	}, nil
}

// Play plays the sound corresponding to the given name.
func (o *OtoPlayer) Play(name string) error {
	o.sc.mu.Lock()
	data, exists := o.sc.sounds[name]
	o.sc.mu.Unlock()

	if !exists {
		return fmt.Errorf("sound %s not found", name)
	}

	// Create a new player and play the sound
	player := o.otoCtx.NewPlayer(data)
	defer player.Close()

	player.Play()

	// We can wait for the sound to finish playing using something like this
	for player.IsPlaying() {
		time.Sleep(time.Millisecond)
	}

	if _, err := player.(io.Seeker).Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("Failed to reset %s after playing: %w", name, err)
	}
	return nil
}
