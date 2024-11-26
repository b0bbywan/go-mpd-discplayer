package notifications

import (
	"log"
)

type Notifier interface {
	PlaySuccess()
	PlayError()
}

type SoundNotifier struct {
	successSoundPath string
	errorSoundPath   string
}

// NewSoundNotifier creates a new instance of SoundNotifier.
func NewSoundNotifier(successSoundPath, errorSoundPath string) *SoundNotifier {
	return &SoundNotifier{
		successSoundPath: successSoundPath,
		errorSoundPath:   errorSoundPath,
	}
}

// PlaySuccess plays the success notification sound.
func (n *SoundNotifier) PlaySuccess() {
	n.play(n.successSoundPath)
}

// PlayError plays the error notification sound.
func (n *SoundNotifier) PlayError() {
	n.play(n.errorSoundPath)
}

func (n *SoundNotifier) play(soundPath string) {
	go func() {
		if err := PlaySound(soundPath); err != nil {
			log.Printf("Failed to play sound (%s): %v", soundPath, err)
		}
	}()
}
