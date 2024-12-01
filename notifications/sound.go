package notifications

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
)

type Sound struct {
	Streamer beep.StreamSeekCloser
	Format   beep.Format
}

type SoundCache struct {
	sounds map[string]*Sound
	sync.Mutex
}

func NewSoundCache(soundsPath map[string]string) *SoundCache {
	soundCache := &SoundCache{sounds: make(map[string]*Sound)}
	for k, v := range soundsPath {
		log.Printf("Loading %s in RAM", v)
		if err := soundCache.loadSound(k, v); err != nil {
			log.Printf("Failed to load sound %s:%s", k, v)
		}
	}
	sr := beep.SampleRate(44100)
	speaker.Init(sr, sr.N(time.Second/10))
	return soundCache
}

func (sc *SoundCache) loadSound(name, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open sound file %s: %w", path, err)
	}
	streamer, format, err := wav.Decode(f)
	if err != nil {
		return fmt.Errorf("failed to decode sound file %s: %w", path, err)
	}
	sc.Lock()
	sc.sounds[name] = &Sound{Streamer: streamer, Format: format}
	sc.Unlock()
	return nil
}

func (sc *SoundCache) Play(name string) error {
	sc.Lock()
	sound, exists := sc.sounds[name]
	sc.Unlock()
	if !exists {
		return fmt.Errorf("sound %s not found", name)
	}
	sc.Lock()
	log.Printf("Playing sound %s", name)
	done := make(chan bool)
	speaker.Play(beep.Seq(sound.Streamer, beep.Callback(func() { done <- true })))
	<-done
	if err := sound.Streamer.Seek(0); err != nil {
		return fmt.Errorf("Failed to reset %s sound: %w", name, err)
	}
	sc.Unlock()
	log.Printf("Played sound %s", name)
	return nil
}
