package notifications

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
)

func PlaySound(soundPath string) error {
	f, err := os.Open(soundPath)
	if err != nil {
		return fmt.Errorf("Failed to open wav file: %w", err)
	}
	defer f.Close()

	streamer, format, err := wav.Decode(f)
	if err != nil {
		f.Close()
		return fmt.Errorf("Failed to decode wav file: %w", err)
	}
	defer streamer.Close()
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		return fmt.Errorf("Failed to init speaker: %w", err)
	}
	done := make(chan bool)
	log.Printf("Playing %s sound...", soundPath)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done
	return nil
}
