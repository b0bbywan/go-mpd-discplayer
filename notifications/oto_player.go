package notifications

import (
	"bytes"
	"fmt"
	"time"

	"github.com/hajimehoshi/oto/v2"
)

type OtoPlayer struct {
	sc     *SoundCache
	otoCtx *oto.Context
}

func NewOtoPlayer(sc *SoundCache) (*OtoPlayer, error) {
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
	sound, err := o.sc.Get(name)
	if err != nil {
		return fmt.Errorf("Could not play %s: %w", name, err)
	}
	// Créer un nouveau reader depuis les données en cache
	reader := bytes.NewReader(sound.Data)

	// Create a new player and play the sound
	player := o.otoCtx.NewPlayer(reader)
	defer player.Close()

	player.Play()

	// We can wait for the sound to finish playing using something like this
	for player.IsPlaying() {
		time.Sleep(time.Millisecond)
	}

	return nil
}

func (o *OtoPlayer) Close() {
	o.sc.Close()
}
