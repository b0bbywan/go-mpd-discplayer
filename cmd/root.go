package cmd

import (
	"fmt"
)

const (
	ActionPlay  = "play"
	ActionStop  = "stop"
	ActionEject = "eject"
)

// ExecuteAction runs a CLI action. device is the drive used by play and eject.
func (player *Player) ExecuteAction(device, action string) error {
	switch action {
	case ActionPlay:
		return player.PlayDisc(device)
	case ActionStop:
		return player.Stop()
	case ActionEject:
		return player.EjectDisc(device)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
