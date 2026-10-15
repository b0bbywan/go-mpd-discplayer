package cmd

import (
	"fmt"
)

const (
	ActionPlay  = "play"
	ActionStop  = "stop"
	ActionEject = "eject"
)

// executeAction handles the main logic for each action (add or remove).
func (player *Player) ExecuteAction(device, action string) error {
	switch action {
	case ActionPlay:
		return player.PlayDevice(device)
	case ActionStop:
		return player.Stop()
	case ActionEject:
		return player.EjectDevice(device)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
