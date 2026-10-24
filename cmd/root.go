package cmd

import (
	"fmt"
)

// CLI action names, the commands typed after the program name.
const (
	ActionDiscPlay   = "disc play"
	ActionDiscStop   = "disc stop"
	ActionDiscEject  = "disc eject"
	ActionStreamPlay = "stream play"
	ActionStop       = "stop"
)

// Action is a CLI action. Device is the drive used by disc play and eject,
// URL the stream played by stream play.
type Action struct {
	Name   string
	Device string
	URL    string
}

// ExecuteAction runs a CLI action.
func (player *Player) ExecuteAction(action Action) error {
	switch action.Name {
	case ActionDiscPlay:
		return player.PlayDisc(action.Device)
	case ActionDiscStop:
		return player.StopDisc()
	case ActionDiscEject:
		return player.EjectDisc(action.Device)
	case ActionStreamPlay:
		return player.PlayURL(action.URL)
	case ActionStop:
		return player.Stop()
	default:
		return fmt.Errorf("unknown action: %s", action.Name)
	}
}
