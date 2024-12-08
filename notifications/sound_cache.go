package notifications

import (
	"fmt"
	"os"
	"sync"

	"github.com/hajimehoshi/go-mp3"
)

type SoundCache struct {
	sounds map[string]*mp3.Decoder // Preloaded audio data
	mu     sync.Mutex
}

func NewSoundCache(soundsPath map[string]string) (*SoundCache, error) {
	sc := &SoundCache{
		sounds: make(map[string]*mp3.Decoder),
	}

	// Preload audio files
	for name, path := range soundsPath {
		data, err := loadAudioFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load sound %s from %s: %w", name, path, err)
		}
		sc.mu.Lock()
		sc.sounds[name] = data
		sc.mu.Unlock()
	}

	return sc, nil
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
