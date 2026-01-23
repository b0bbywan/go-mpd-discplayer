package notifications

import (
	"fmt"
	"os"
	"sync"
)

type SoundEntry struct {
	PCM 		[]byte
}

type SoundCache struct {
	sounds map[string]*SoundEntry // Preloaded audio data
	mu     sync.RWMutex
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

func (sc *SoundCache) Get(name string) (*SoundEntry, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	entry, exists := sc.sounds[name]
	if !exists {
		return nil, fmt.Errorf("sound %s not found", name)
	}

	return entry, nil
}

func (sc *SoundCache) Close() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.sounds = nil // Clear the map for garbage collection
}

func (sc *SoundCache) loadAudioFile(name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.sounds[name] = &SoundEntry{
		PCM:        data,
	}

	return nil
}
