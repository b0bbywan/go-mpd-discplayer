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
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
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
			if err := mpdClient.StartDiscPlayback(); err != nil {
				log.Printf("Error adding tracks: %w", err)
				return fmt.Errorf("Error adding tracks: %w", err)
			}
			return nil
		case ActionStop:
			if err := mpdClient.StopDiscPlayback(); err != nil {
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

	// Initialize MPD client
	mpdClient := mpdplayer.NewReconnectingMPDClient(config.MPDConnection)

	var handlers []*hwcontrol.EventHandler
	// Create event handlers (subscribers) passing the context
	handlers = append(handlers, newDiscHandlers(&wg, mpdClient)...)
	handlers = append(handlers, newUSBHandlers(&wg, mpdClient)...)
	for _, handler := range handlers {
		handler.StartSubscriber(ctx) // Use the passed context
	}

	// Start event monitoring (publish events to handlers)
	wg.Add(1)
	go loop(&wg, ctx, handlers)


	// Signal handling goroutine to cleanly stop the program
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


func loop(wg *sync.WaitGroup, ctx context.Context, handlers []*hwcontrol.EventHandler) {
	defer wg.Done()
	for {
	select {
		case <-ctx.Done():
			log.Println("Stopping from cmd.")
			return
		default:
			if err := hwcontrol.StartMonitor(ctx, handlers); err != nil {
				log.Printf("Error starting monitor: %w\n", err)
				time.Sleep(time.Second) // Retry after some delay
				continue
			}
		}
	}
}
