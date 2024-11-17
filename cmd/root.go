package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"github.com/b0bbywan/go-mpd-discplayer/config"
	"github.com/b0bbywan/go-mpd-discplayer/disc"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

const (
	ActionPlay = "play"
	ActionStop = "stop"
)

var (
	MPDConnection   = config.MPDConnection
)

// executeAction handles the main logic for each action (add or remove).
func ExecuteAction(action string) error {
	mpdClient := mpdplayer.NewReconnectingMPDClient(config.MPDConnection)
	switch action {
		case ActionPlay:
			if err := mpdClient.StartPlayback(); err != nil {
				log.Printf("Error adding tracks: %w", err)
				return fmt.Errorf("Error adding tracks: %w", err)
			}
			return nil
		case ActionStop:
			if err := mpdClient.StopPlayback(); err != nil {
				log.Printf("Error adding tracks: %w", err)
				return fmt.Errorf("Error adding tracks: %w", err)
			}
			return nil
		default:
			return fmt.Errorf("Unknown action: %s", action)
		}
}

func Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Handle OS signals to gracefully stop the program
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	mpdClient := mpdplayer.NewReconnectingMPDClient(config.MPDConnection)
	handler := makeHandler(&wg, mpdClient)

	wg.Add(1)
	go loop(&wg, ctx, handler)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range sigChan {
			log.Println("Received termination signal. Exiting...")
			cancel()
			return
		}
	}()

	// Wait for termination signal
	<-ctx.Done()

	// Cleanup after receiving the termination signal
	if mpdClient != nil {
		mpdClient.Disconnect()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("All tasks completed. Exiting...")
	return nil
}

func makeHandler(wg *sync.WaitGroup, mpdClient *mpdplayer.ReconnectingMPDClient) disc.EventHandler {
	handler := disc.EventHandler{
		OnAdd: func() {
			log.Println("Adding tracks to MPD...")
			wg.Add(1) // Increment the counter before starting the task
			go func() {
				defer wg.Done() // Decrement the counter once the task is done
				if err := mpdClient.StartPlayback(); err != nil {
					log.Printf("Error adding tracks: %v\n", err)
					return
				}
			}()
		},
		OnRemove: func() {
			log.Println("Stopping playback...")
			wg.Add(1) // Increment the counter before starting the task
			go func() {
				defer wg.Done() // Decrement the counter once the task is done
				if err := mpdClient.StopPlayback(); err != nil {
					log.Printf("Error removing tracks: %v\n", err)
					return
				}
			}()
		},
	}
	return handler
}

func loop(wg *sync.WaitGroup, ctx context.Context, handler disc.EventHandler) {
	defer wg.Done()
	for {
	select {
		case <-ctx.Done():
			log.Println("Stopping from cmd.")
			return
		default:
			// Update the handler with the new MPD client
			if err := disc.StartMonitor(ctx, config.TargetDevice, handler); err != nil {
				log.Printf("Error starting monitor: %v\nRestarting...", err)
				// Optionally, sleep or delay before retrying, or stop based on error
				time.Sleep(time.Second) // Retry after some delay if you want
				continue
			}
		}
	}
}
