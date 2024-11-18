package cmd

import (
	"log"
	"sync"
	"github.com/b0bbywan/go-mpd-discplayer/config"
	"github.com/b0bbywan/go-mpd-discplayer/disc"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)


func newDiscHandler(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient) *disc.EventHandler {
	handler := disc.NewBasicDiscHandler(config.TargetDevice)

	handler.OnAddFunc = func() {
		log.Println("Adding tracks to MPD...")
		wg.Add(1) // Increment the counter before starting the task
		go func() {
			defer wg.Done() // Decrement the counter once the task is done
			if err := mpdClient.StartPlayback(); err != nil {
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
			if err := mpdClient.StopPlayback(); err != nil {
				log.Printf("Error removing tracks: %v\n", err)
				return
			}
		}()
	}
	return handler
}
