package mpdplayer

import (
	"fmt"
	"log"

	"github.com/fhs/gompd/v2/mpd"
	"github.com/b0bbywan/go-disc-cuer/cue"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
)

type PlaybackAction func(client *mpd.Client, device string) error

func (rc *ReconnectingMPDClient) StartDiscPlayback(device string) error {
	return rc.startPlayback(attemptToLoadCD, device)
}

// StartDiscPlayback now accepts a custom playback function
func (rc *ReconnectingMPDClient) startPlayback(playbackFunc PlaybackAction, device string) error {
	return rc.execute(func(client *mpd.Client) error {
		if err := clearQueue(client); err != nil {
			return fmt.Errorf("failed to clear MPD queue: %w", err)
		}
		// Use the provided playback function
		if err := playbackFunc(client, device); err != nil {
			return fmt.Errorf("failed to load playlist: %w", err)
		}
		return client.Play(-1)
	})
}

func (rc *ReconnectingMPDClient) StopDiscPlayback() error {
	return rc.execute(func(client *mpd.Client) error {
		if err := client.Stop(); err != nil {
			return fmt.Errorf("error: Failed to stop MPD playback: %w", err)
		}
		return clearQueue(client)
	})
}

// attemptToLoadCD tries to load the CD by first attempting to load a CUE file.
// If loading the CUE file fails, it falls back to loading individual CDDA tracks,
func attemptToLoadCD(client *mpd.Client, device string) error {
	var err error
	if err = loadCue(client, device); err == nil {
		return nil
	}

	log.Printf("info: No valid CUE file, trying to load CDDA tracks: %w", err)
	// Try loading individual tracks if CUE file loading failed
	if err = loadCDDATracks(client, device); err == nil {
		return nil
	}
    return fmt.Errorf("failed to load CD, no valid CUE file and unable to load CDDA tracks: %w", err)
}

// clearQueue clears the MPD playlist.
func clearQueue(client *mpd.Client) error {
	if err := client.Clear(); err != nil {
		return fmt.Errorf("failed to clear MPD playlist: %w", err)
	}
	log.Println("info: MPD queue cleared")
	return nil
}

// loadCDDATracks adds individual CDDA tracks to the MPD playlist based on the track count.
// It does not handle fallback logic, leaving it to the caller.
func loadCDDATracks(client *mpd.Client, device string) error {
	trackCount, err := hwcontrol.GetTrackCount(device)
	if err != nil {
		return fmt.Errorf("failed to get track count: %w", err)
	}

	return addTracks(client, trackCount)
}

func loadCue(client *mpd.Client, device string) error {
	cueFilePath, err := cue.GenerateDefaultFromDisc(device)
	if err != nil || cueFilePath == "" {
		fmt.Printf("failed to generate CUE file: %v", err)
		return fmt.Errorf("failed to generate CUE file: %w", err)
	}
	log.Printf("info: Loading playlist from %s\n", cueFilePath)
	if err := client.PlaylistLoad(cueFilePath, -1, -1); err != nil {
		return fmt.Errorf("failed to load CUE playlist: %w", err)
	}
	return nil
}

// addTracks adds individual CDDA tracks to the MPD playlist based on the specified track count.
func addTracks(client *mpd.Client, trackCount int) error {
	for track := 1; track <= trackCount; track++ {
		if err := client.Add(fmt.Sprintf("cdda:///%d", track)); err != nil {
			return fmt.Errorf("failed to add track %d: %w", track, err)
		}
	}
	log.Printf("info: Added %d tracks to the playlist", trackCount)
	return nil
}
