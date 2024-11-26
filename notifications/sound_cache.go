package notifications

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/hajimehoshi/go-mp3"
)

type SoundEntry struct {
	decoder *mp3.Decoder
	file    *os.File
}

type SoundCache struct {
	sounds map[string]*SoundEntry // Preloaded audio data
	mu     sync.Mutex
}

func NewSoundCache(soundsPath map[string]string) (*SoundCache, error) {
	sc := &SoundCache{
		sounds: make(map[string]*SoundEntry),
	}

	// Preload audio files
	for name, path := range soundsPath {
		if err := sc.loadAudioFile(name, path); err != nil {
			return nil, fmt.Errorf("failed to load sound %s from %s: %w", name, path, err)
		}
	}

	return sc, nil
}

func (sc *SoundCache) Get(name string) (*mp3.Decoder, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	data, exists := sc.sounds[name]
	if !exists {
		return nil, fmt.Errorf("sound %s not found", name)
	}
	return data.decoder, nil
}

func (sc *SoundCache) Close() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for name, entry := range sc.sounds {
		if err := entry.file.Close(); err != nil {
			log.Printf("failed to close file for sound %s: %v\n", name, err)
		}
	}
	sc.sounds = nil // Clear the map for garbage collection
}

func (sc *SoundCache) loadAudioFile(name, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}

	decodedMp3, err := mp3.NewDecoder(file)
	if err != nil {
		return fmt.Errorf("failed to decode MP3 file: %w", err)
	}
	data := &SoundEntry{
		decoder: decodedMp3,
		file:    file,
	}
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.sounds[name] = data
	return nil
}
