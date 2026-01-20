package notifications

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/hajimehoshi/go-mp3"
)

type SoundEntry struct {
	Data       []byte
	SampleRate int
}

type SoundCache struct {
	sounds map[string]*SoundEntry
	mu     sync.RWMutex // RWMutex pour permettre des lectures concurrentes
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
		log.Printf("Loaded sound: %s (%d bytes, %d Hz)", name, len(sc.sounds[name].Data), sc.sounds[name].SampleRate)
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

	// Plus besoin de fermer les fichiers, tout est en mémoire
	sc.sounds = nil
	log.Println("SoundCache closed")
}

func (sc *SoundCache) loadAudioFile(name, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close() // On ferme tout de suite après avoir lu

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return fmt.Errorf("failed to decode MP3 file: %w", err)
	}

	// Décoder tout le MP3 en mémoire
	data, err := io.ReadAll(decoder)
	if err != nil {
		return fmt.Errorf("failed to read decoded MP3 data: %w", err)
	}

	entry := &SoundEntry{
		Data:       data,
		SampleRate: decoder.SampleRate(),
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.sounds[name] = entry

	return nil
}
