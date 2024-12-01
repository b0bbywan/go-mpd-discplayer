package cmd

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

const (
	ActionPlay = "play"
	ActionStop = "stop"
)

// executeAction handles the main logic for each action (add or remove).
func ExecuteAction(mpdClient *mpdplayer.ReconnectingMPDClient, device, action string) error {
	switch action {
	case ActionPlay:
		if err := mpdClient.StartDiscPlayback(device); err != nil {
			return fmt.Errorf("Error adding tracks: %w", err)
		}
		return nil
	case ActionStop:
		if err := mpdClient.StopDiscPlayback(); err != nil {
			return fmt.Errorf("Error adding tracks: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("Unknown action: %s", action)
	}

	return nil
}

func Run(wg *sync.WaitGroup, ctx context.Context, mpdClient *mpdplayer.ReconnectingMPDClient) error {
	notifier := notifications.NewRootNotifier()
	var handlers []*hwcontrol.EventHandler
	// Create event handlers (subscribers) passing the context
	handlers = append(handlers, newDiscHandlers(wg, mpdClient, notifier)...)
	handlers = append(handlers, newUSBHandlers(wg, mpdClient, notifier)...)
	for _, handler := range handlers {
		handler.StartSubscriber(wg, ctx) // Use the passed context
	}

	// Start event monitoring (publish events to handlers)
	wg.Add(1)
	go loop(wg, ctx, handlers)

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
