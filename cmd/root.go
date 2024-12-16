package cmd

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
)

const (
	ActionPlay = "play"
	ActionStop = "stop"
)

// executeAction handles the main logic for each action (add or remove).
func ExecuteAction(player *Player, device, action string) error {
	switch action {
	case ActionPlay:
		if err := player.Client.StartDiscPlayback(device); err != nil {
			return fmt.Errorf("Error adding tracks: %w", err)
		}
		return nil
	case ActionStop:
		if err := player.Client.StopDiscPlayback(); err != nil {
			return fmt.Errorf("Error adding tracks: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("Unknown action: %s", action)
	}

	return nil
}

func Run(player *Player) error {
	var handlers []*hwcontrol.EventHandler
	var wg sync.WaitGroup
	// Create event handlers (subscribers) passing the context
	handlers = append(handlers, newDiscHandlers(&wg, player)...)
	handlers = append(handlers, newUSBHandlers(&wg, player)...)
	for _, handler := range handlers {
		handler.StartSubscriber(&wg, player.Ctx()) // Use the passed context
	}

	// Start event monitoring (publish events to handlers)
	wg.Add(1)
	go loop(&wg, player.Ctx(), handlers)
	<-player.Ctx().Done()
	wg.Wait()
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
