package notifications

import (
	"bytes"
	"fmt"
	"time"

	"github.com/ebitengine/oto/v3"
)

type OtoPlayer struct {
	sc     *SoundCache
	otoCtx *oto.Context
}

func NewOtoPlayer(sc *SoundCache) (*OtoPlayer, error) {
	op := &oto.NewContextOptions{
		SampleRate:   44100,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	}

	otoCtx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize oto: %w", err)
	}

	// Attendre que le contexte soit prêt
	<-readyChan

	return &OtoPlayer{
		sc:     sc,
		otoCtx: otoCtx,
	}, nil
}

func (o *OtoPlayer) Play(name string) error {
	sound, err := o.sc.Get(name)
	if err != nil {
		return fmt.Errorf("Could not play %s: %w", name, err)
	}

	// Créer un reader depuis les données PCM
	reader := bytes.NewReader(sound.PCM)

	// Créer un player et jouer le son
	player := o.otoCtx.NewPlayer(reader)
	defer player.Close()

	player.Play()

	// Attendre la fin de la lecture
	for player.IsPlaying() {
		time.Sleep(time.Millisecond)
	}

	return nil
}

func (o *OtoPlayer) Close() {
	o.sc.Close()
}
