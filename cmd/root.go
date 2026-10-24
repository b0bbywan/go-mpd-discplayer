package cmd

import (
	"fmt"

	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
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
		if err := player.Client.StartDiscPlayback(device); err != nil {
			return fmt.Errorf("error adding tracks: %w", err)
		}
		return nil
	case ActionStop:
		if err := player.Client.StopAndRemove(mpdplayer.CDDAPathPrefix); err != nil {
			return fmt.Errorf("error adding tracks: %w", err)
		}
		return nil
	case ActionEject:
		return player.Eject(device)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
