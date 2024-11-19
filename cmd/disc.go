package cmd

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"github.com/b0bbywan/go-mpd-discplayer/config"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

func newDiscHandler(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient) *hwcontrol.EventHandler {
	handler := hwcontrol.NewBasicDiscHandler(filepath.Base(config.TargetDevice))

	handler.OnAddFunc = func() {
		log.Println("Adding tracks to MPD...")
		wg.Add(1) // Increment the counter before starting the task
		go func() {
			defer wg.Done() // Decrement the counter once the task is done
			if err := hwcontrol.SetDiscSpeed(config.GetDevicePath(), config.DiscSpeed); err != nil {
				fmt.Printf("Failed to set disc speed: %w", err)
			}
			if err := mpdClient.StartDiscPlayback(); err != nil {
				log.Printf("Error adding tracks: %v\n", err)
				return
			}
		}()
	}
	handler.OnRemoveFunc = func() {
		log.Println("Stopping playback...")
		wg.Add(1) // Increment the counter before starting the task
		go func() {
			defer wg.Done() // Decrement the counter once the task is done
			if err := mpdClient.StopDiscPlayback(); err != nil {
				log.Printf("Error removing tracks: %v\n", err)
				return
			}
		}()
	}
	return handler
}
