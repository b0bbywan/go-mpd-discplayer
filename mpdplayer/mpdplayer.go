package mpdplayer

import (
	"fmt"
	"log"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/b0bbywan/go-cd-cuer/cue"
	"github.com/b0bbywan/go-mpd-discplayer/disc"
)

var (

)

// loadCDDATracks adds individual CDDA tracks to the MPD playlist based on the track count.
// It does not handle fallback logic, leaving it to the caller.
func loadCDDATracks(client *mpd.Client) error {
	trackCount, err := disc.GetTrackCount()
	if err != nil {
		return err
	}

	if err := addTracks(client, trackCount); err != nil {
		return err
	}

	return nil
}

func loadCue(client *mpd.Client) error {
    cueFilePath, err := cue.GenerateFromDisc()
    if err != nil || cueFilePath == "" {
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

// attemptToLoadCD tries to load the CD by first attempting to load a CUE file.
// If loading the CUE file fails, it falls back to loading individual CDDA tracks,
// and if that also fails, it adds the whole CD.
func AttemptToLoadCD(client *mpd.Client) error {
	var err error
	if err = loadCue(client); err == nil {
		return nil
	}

	log.Printf("info: No valid CUE file, trying to load CDDA tracks: %v", err)
	// Try loading individual tracks if CUE file loading failed
	return loadCDDATracks(client)
}

// clearQueue clears the MPD playlist.
func ClearQueue(client *mpd.Client) error {
	if err := client.Clear(); err != nil {
		return fmt.Errorf("failed to clear MPD playlist: %w", err)
	}
	log.Println("info: MPD queue cleared")
	return nil
}
