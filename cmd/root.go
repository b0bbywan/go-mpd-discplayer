package cmd

import (
	"fmt"
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
