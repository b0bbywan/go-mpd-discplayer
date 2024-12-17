package cmd

import (
	"fmt"
	"sync"
)

const (
	ActionPlay = "play"
	ActionStop = "stop"
)

// executeAction handles the main logic for each action (add or remove).
func (player *Player) ExecuteAction(device, action string) error {
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

func (player *Player) Run() error {
	var wg sync.WaitGroup
	// Create event handlers (subscribers) passing the context
	player.newDiscHandlers(&wg)
	player.newUSBHandlers(&wg)

	player.Start(&wg)
	// Start event monitoring (publish events to handlers)
	<-player.Ctx().Done()
	wg.Wait()
	return nil
}
