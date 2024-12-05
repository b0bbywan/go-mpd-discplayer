package notifications

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto/v2"
)

type SoundCache struct {
	sounds map[string]*mp3.Decoder // Preloaded audio data
	otoCtx *oto.Context
	mu     sync.Mutex
}

func NewSoundCache(soundsPath map[string]string) (*SoundCache, error) {
	otoCtx, ready, err := oto.NewContext(44100, 2, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize oto: %w", err)
	}
	<-ready

	sc := &SoundCache{
		sounds: make(map[string]*mp3.Decoder),
		otoCtx: otoCtx,
	}

	// Preload audio files
	for name, path := range soundsPath {
		data, err := loadAudioFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load sound %s from %s: %w", name, path, err)
		}
		sc.sounds[name] = data
	}

	return sc, nil
}

// Play plays the sound corresponding to the given name.
func (sc *SoundCache) Play(name string) error {
	sc.mu.Lock()
	data, exists := sc.sounds[name]
	sc.mu.Unlock()

	if !exists {
		return fmt.Errorf("sound %s not found", name)
	}

	// Create a new player and play the sound
	player := sc.otoCtx.NewPlayer(data)
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

func loadAudioFile(path string) (*mp3.Decoder, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}

	decodedMp3, err := mp3.NewDecoder(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode MP3 file: %w", err)
	}
	return decodedMp3, nil
}
