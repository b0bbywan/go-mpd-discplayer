package cmd

import (
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

const (
	// Should be start and stop ?
	ActionAdd    = "add"
	ActionRemove = "remove"
	socketPath   = "/run/user/1000/mpd/socket"
)


// executeAction handles the main logic for each action (add or remove).
func ExecuteAction(action string) error {
	client, err := mpd.Dial("unix", socketPath)
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
	if err := mpdplayer.ClearQueue(client); err != nil {
		return fmt.Errorf("failed to clear MPD queue: %w", err)
	}

	if err := mpdplayer.AttemptToLoadCD(client); err != nil {
		return err
	}

	// TODO move to attempttoloadcd
	if err := client.Play(-1); err != nil {
		return fmt.Errorf("failed to start playback: %w", err)
	}
	return nil
}

func doUnmount(client *mpd.Client) error {
	// TODO move to mpd
	if err := client.Stop(); err != nil {
		return fmt.Errorf("error: Failed to stop MPD playback: %w", err)
	}

	if err := mpdplayer.ClearQueue(client); err != nil {
		return fmt.Errorf("error: Failed to clear MPD playlist: %w", err)
	}

	return nil
}
