package cmd

import (
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/config"
)

const (
	// Should be start and stop ?
	ActionAdd    = "add"
	ActionRemove = "remove"
)

var (
	MPDConnection   = config.MPDConnection
)

// executeAction handles the main logic for each action (add or remove).
func ExecuteAction(action string) error {
	client, err := mpd.Dial(MPDConnection.Type, MPDConnection.Address)
	if err != nil {
		return fmt.Errorf("failed to connect to MPD: %w", err)
	}
	defer client.Close()

	switch action {
	case ActionAdd:
		if err := doMount(client); err != nil {
			return fmt.Errorf("failed to add CD: %w", err)
		}
	case ActionRemove:
		if err := doUnmount(client); err != nil {
			return fmt.Errorf("failed to remove CD: %w", err)
		}
	default:
		return fmt.Errorf("invalid ACTION '%s'", action)
	}
	return nil
}

func doMount(client *mpd.Client) error {
	return mpdplayer.CleanAndStart(client)
}

func doUnmount(client *mpd.Client) error {
	return mpdplayer.StopAndClean(client)
}
