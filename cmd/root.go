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
	"github.com/fhs/gompd/v2/mpd"
	"github.com/b0bbywan/go-mpd-discplayer/config"
	"github.com/b0bbywan/go-mpd-discplayer/disc"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

var (
	MPDConnection   = config.MPDConnection
)

// executeAction handles the main logic for each action (add or remove).
func Execute() error {
	client, err := mpd.Dial(MPDConnection.Type, MPDConnection.Address)
	if err != nil {
		return fmt.Errorf("failed to connect to MPD: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	handler := makeHandler(&wg, nil)

	// Handle OS signals to gracefully stop the program
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
				case <-ctx.Done():
					log.Println("Stopping from cmd.")
					return
				default:
					handler = makeHandler(&wg, client)

					// Update the handler with the new MPD client
					if err := disc.StartMonitor(ctx, config.TargetDevice, handler); err != nil {
						log.Printf("Error starting monitor: %v\nRestarting...", err)
						// Optionally, sleep or delay before retrying, or stop based on error
						time.Sleep(time.Second) // Retry after some delay if you want
						continue
					}
				}					
			}
	}()

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
	if client != nil {
		log.Println("Closing MPD client.")
		client.Close()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("All tasks completed. Exiting...")
	return nil
}

func doMount(client *mpd.Client) error {
	return mpdplayer.CleanAndStart(client)
}

func doUnmount(client *mpd.Client) error {
	return mpdplayer.StopAndClean(client)
}

func makeHandler(wg *sync.WaitGroup, client *mpd.Client) disc.EventHandler {
	handler := disc.EventHandler{
		OnAdd: func() {
			log.Println("Adding tracks to MPD...")
			wg.Add(1) // Increment the counter before starting the task
			go func() {
				defer wg.Done() // Decrement the counter once the task is done
				if err := doMount(client); err != nil {
					log.Printf("Error adding tracks: %v\n", err)
				}
			}()
		},
		OnRemove: func() {
			log.Println("Stopping playback...")
			wg.Add(1) // Increment the counter before starting the task
			go func() {
				defer wg.Done() // Decrement the counter once the task is done
				if err := doUnmount(client); err != nil {
					log.Printf("Error removing tracks: %v\n", err)
				}
			}()
		},
	}
	return handler
}
