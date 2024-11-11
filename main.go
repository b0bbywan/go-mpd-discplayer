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

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("error: No ACTION specified")
	}
	action := os.Args[1]

	client, err := mpd.Dial("unix", socketPath)
	if err != nil {
		log.Fatalf("error: Failed to connect to MPD: %v", err)
	}
	defer client.Close()

	switch action {
	case ActionAdd:
		if err := doMount(client); err != nil {
			log.Fatalf("Failed to add CD: %v", err)
		}
	case ActionRemove:
		if err := doUnmount(client); err != nil {
			log.Fatalf("Failed to remove CD: %v", err)
		}
	default:
		log.Fatalf("error: Invalid ACTION '%s'", action)
	}
}

func getDiscID() (int, error) {
	disc, err := discid.Read("")
	if err != nil {
		return 0, err
	}
	defer disc.Close()

	return disc.LastTrackNumber(), nil
}

func addCDDATracks(client *mpd.Client) error {
	numTracks, err := getDiscID()
	if err != nil {
		log.Printf("warning: Failed to detect CD tracks, adding whole CD: %v", err)
		return client.Add("cdda://")
	}

	for i := 1; i <= numTracks; i++ {
		if err := client.Add(fmt.Sprintf("cdda:///%d", i)); err != nil {
			return err
		}
	}
	return nil
}

func doMount(client *mpd.Client) error {
	fmt.Println("info: Clearing MPD queue")
	if err := client.Clear(); err != nil {
		return fmt.Errorf("failed to clear MPD queue: %w", err)
	}

	cueFilePath, err := cue.Generate("", "", false)
	if err != nil || cueFilePath == "" {
		log.Printf("info: No CUE file generated, falling back to basic CDDA loading")
		if err := addCDDATracks(client); err != nil {
			return fmt.Errorf("failed to add CDDA tracks: %w", err)
		}
	} else {
		fmt.Printf("info: Loading playlist from %s\n", cueFilePath)
		if err := client.PlaylistLoad(cueFilePath, -1, -1); err != nil {
			return fmt.Errorf("error: Failed to load CUE file: %w", err)
		}
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

	if err := client.Clear(); err != nil {
		return fmt.Errorf("error: Failed to clear MPD playlist: %w", err)
	}

	return nil
}
