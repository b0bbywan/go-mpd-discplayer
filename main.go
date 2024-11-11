package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fhs/gompd/v2/mpd"
	"go.uploadedlobster.com/discid"
	"github.com/b0bbywan/go-cd-cuer/cue"
)

const (
	ActionAdd    = "add"
	ActionRemove = "remove"
	socketPath   = "/run/user/1000/mpd/socket"
)

var action string

// init handles argument validation and sets the action variable
func init() {
	if len(os.Args) < 2 {
		log.Fatalf("error: No ACTION specified")
	}
	action = os.Args[1]
}

func main() {
	if err := executeAction(action); err != nil {
		log.Fatalf("error: %v", err)
	}
}

// executeAction handles the main logic for each action (add or remove).
func executeAction(action string) error {
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

func getTrackCount() (int, error) {
	disc, err := discid.Read("")
	if err != nil {
		return 0, err
	}
	defer disc.Close()

	return disc.LastTrackNumber(), nil
}

// loadCDDATracks adds individual CDDA tracks to the MPD playlist based on the track count.
// It does not handle fallback logic, leaving it to the caller.
func loadCDDATracks(client *mpd.Client) error {
	trackCount, err := getTrackCount()
	if err != nil {
		return err
	}

	if err := addTracks(client, trackCount); err != nil {
		return err
	}

	return nil
}

func loadCue(client *mpd.Client) error {
    cueFilePath, err := cue.Generate("", "", false)
    if err != nil || cueFilePath == "" {
        return fmt.Errorf("failed to generate CUE file: %w", err)
    }
    log.Printf("info: Loading playlist from %s\n", cueFilePath)
    if err := client.PlaylistLoad(cueFilePath, -1, -1); err != nil {
        return fmt.Errorf("failed to load CUE playlist: %w", err)
    }
    return nil
}


// addWholeCD adds the entire CD as a single item in the MPD playlist.
func addWholeCD(client *mpd.Client) error {
	if err := client.Add("cdda://"); err != nil {
		return fmt.Errorf("failed to add whole CD: %w", err)
	}
	log.Println("info: Added whole CD to the playlist")
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
func attemptToLoadCD(client *mpd.Client) error {
	var err error
	if err = loadCue(client); err == nil {
		return nil
	}

	log.Printf("info: No valid CUE file, trying to load CDDA tracks: %v", err)
	// Try loading individual tracks if CUE file loading failed
	if err = loadCDDATracks(client); err == nil {
		return nil
	}

	log.Printf("warning: Failed to add individual CDDA tracks, falling back to whole CD: %v", err)
	return addWholeCD(client)
}

// clearQueue clears the MPD playlist.
func clearQueue(client *mpd.Client) error {
	if err := client.Clear(); err != nil {
		return fmt.Errorf("failed to clear MPD playlist: %w", err)
	}
	log.Println("info: MPD queue cleared")
	return nil
}

func doMount(client *mpd.Client) error {
	fmt.Println("info: Clearing MPD queue")
	if err := clearQueue(client); err != nil {
		return fmt.Errorf("failed to clear MPD queue: %w", err)
	}

	if err := attemptToLoadCD(client); err != nil {
		return err
	}

	if err := client.Play(-1); err != nil {
		return fmt.Errorf("failed to start playback: %w", err)
	}
	return nil
}

func doUnmount(client *mpd.Client) error {
	if err := client.Stop(); err != nil {
		return fmt.Errorf("error: Failed to stop MPD playback: %w", err)
	}

	if err := clearQueue(client); err != nil {
		return fmt.Errorf("error: Failed to clear MPD playlist: %w", err)
	}

	return nil
}
