package mpdplayer

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fhs/gompd/v2/mpd"

	"github.com/b0bbywan/go-disc-cuer/config"
	"github.com/b0bbywan/go-disc-cuer/cue"
)

const CDDAPathPrefix = "cdda://"

type PlaybackAction func(client *mpd.Client, device string) error

func (rc *ReconnectingMPDClient) StartDiscPlayback(device string) error {
	return rc.startPlayback(rc.attemptToLoadCD, device)
}

func (rc *ReconnectingMPDClient) StartUSBPlayback(device string) error {
	return rc.startPlayback(addUSBToQueue, device)
}

func (rc *ReconnectingMPDClient) StartPlayback(uri string) error {
	return rc.startPlayback(addUri, uri)
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
	return rc.StopPlayback(CDDAPathPrefix)
}

func (rc *ReconnectingMPDClient) StopPlayback(label string) error {
	return rc.execute(func(client *mpd.Client) error {
		if checkPathPlaying(client, label) {
			if err := client.Stop(); err != nil {
				return fmt.Errorf("error: Failed to stop MPD playback: %w", err)
			}
		}
		return deleteFromPlaylist(client, label)
	})
}

func (rc *ReconnectingMPDClient) Mount(neighbor, label string) error {
	neighborDiskId := fmt.Sprintf("udisks://by-uuid-%s", neighbor)
	return rc.execute(func(client *mpd.Client) error {
		if err := findNeighbor(client, neighborDiskId); err != nil {
			return fmt.Errorf("Failed to find %s in neighbors list: %w", neighbor, err)
		}
		if err := mount(client, neighborDiskId, label); err != nil {
			return fmt.Errorf("Failed to mount %s -> %s: %w", neighbor, label, err)
		}
		return nil
	})
}

func (rc *ReconnectingMPDClient) Unmount(label string) error {
	return rc.execute(func(client *mpd.Client) error {
		if err := unmount(client, label); err != nil {
			return fmt.Errorf("Failed to unmount %s: %w", label, err)
		}
		return nil
	})
}

func (rc *ReconnectingMPDClient) ClearMounts() error {
	return rc.execute(func(client *mpd.Client) error {
		mounts, err := listMounts(client)
		if err != nil {
			return fmt.Errorf("Failed to list mounts: %w", err)
		}

		if err = checkMountsOrUnmount(client, mounts); err != nil {
			return fmt.Errorf("Failed to check or unmount: %w", err)
		}
		return nil
	})
}

func (rc *ReconnectingMPDClient) GetConfig() (string, error) {
	var config mpd.Attrs
	var err error
	if err = rc.execute(func(client *mpd.Client) error {
		config, err = client.Command("config").Attrs()
		return err
	}); err != nil {
		return "", fmt.Errorf("Failed to get MPD Config from server: %w", err)
	}
	musicDirectory, ok := config["music_directory"]
	if !ok {
		return "", fmt.Errorf("music_directory not found in config")
	}
	return musicDirectory, nil
}

// attemptToLoadCD tries to load the CD by first attempting to load a CUE file.
// If loading the CUE file fails, it falls back to loading individual CDDA tracks,
func (rc *ReconnectingMPDClient) attemptToLoadCD(client *mpd.Client, device string) error {
	var err error
	if err = loadCue(client, rc.mpcConfig.CuerConfig, device); err == nil {
		return nil
	}

	log.Printf("info: No valid CUE file, trying to load CDDA tracks: %v", err)
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
func loadCDDATracks(client *mpd.Client, device string) error {
	trackCount, err := getTrackCount(device)
	if err != nil {
		return fmt.Errorf("failed to get track count: %w", err)
	}

	return addTracks(client, trackCount)
}

func loadCue(client *mpd.Client, cuerConfig *config.Config, device string) error {
	if cuerConfig == nil {
		return fmt.Errorf("No Cuer config to generate from")
	}
	cueFilePath, err := cue.GenerateDefaultFromDisc(device, cuerConfig)
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
		if err := addUri(client, fmt.Sprintf("%s/%d", CDDAPathPrefix, track)); err != nil {
			return fmt.Errorf("failed to add track %d: %w", track, err)
		}
	}
	log.Printf("info: Added %d tracks to the playlist", trackCount)
	return nil
}

func checkPathPlaying(client *mpd.Client, checkPath string) bool {
	song, err := client.CurrentSong()
	if err != nil {
		return true
	}

	// Check if a song is currently playing
	if len(song) == 0 {
		return true
	}

	return checkSongPath(song, checkPath)
}

func checkSongPath(song mpd.Attrs, checkPath string) bool {
	if path, ok := song["file"]; ok {
		return strings.HasPrefix(path, checkPath)
	}
	return false
}

// addUSBToQueue adds the specified label to the playlist.
func addUSBToQueue(client *mpd.Client, label string) error {
	if err := UpdateDBAndWait(client, label); err != nil {
		return fmt.Errorf("Database update failed: %w", err)
	}
	log.Printf("Adding %s files to queue...", label)
	return addUri(client, label)
}

func addUri(client *mpd.Client, uri string) error {
	if err := client.Add(uri); err != nil {
		return fmt.Errorf("failed to add uri %s: %w", uri, err)
	}
	return nil
}

func deleteFromPlaylist(client *mpd.Client, checkPath string) error {
	playlist, err := client.PlaylistInfo(-1, -1)
	if err != nil {
		return fmt.Errorf("failed to fetch MPD playlist: %w", err)
	}

	start := -1 // Initialize start index to -1 to indicate no active range

	for i := len(playlist) - 1; i >= -1; i-- {
		if i >= 0 && checkSongPath(playlist[i], checkPath) {
			// Start a new range if not already started
			if start == -1 {
				start = i
			}
		} else if start != -1 {
			// End the current range and delete it
			if err := client.Delete(i+1, start+1); err != nil {
				return fmt.Errorf("failed to delete playlist range (%d, %d): %w", i+1, start, err)
			}
			log.Printf("info: Deleted songs from position %d to %d", i+1, start)
			start = -1 // Reset start index
		}
	}
	return nil
}

func UpdateDBAndWait(client *mpd.Client, label string) error {
	if _, err := client.Update(label); err != nil {
		return fmt.Errorf("Failed to udpate Database: %w", err)
	}
	timeout := 30 * time.Second
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timeoutChan := time.After(timeout)
	for {
		if !DbUpdating(client) {
			log.Println("Database update finished.")
			return nil
		}
		select {
		case <-ticker.C:
		case <-timeoutChan:
			return fmt.Errorf("Database did not finish update within timeout")
		}
	}
}

func DbUpdating(client *mpd.Client) bool {
	status, err := client.Status()
	if err != nil {
		return true
	}
	_, updating := status["updating_db"]
	return updating
}

func findNeighbor(client *mpd.Client, neighbor string) error {
	res, err := client.Command("listneighbors").AttrsList("neighbor")
	if err != nil {
		return fmt.Errorf("Failed to list neighbors, is udisk neighbor plugin enabled?: %w", err)
	}
	for _, v := range res {
		if neighbor == v["neighbor"] {
			return nil
		}
	}
	return fmt.Errorf("neighbor %s not found", neighbor)
}

func mount(client *mpd.Client, neighbor, label string) error {
	if err := client.Command("mount %s %s", label, neighbor).OK(); err != nil {
		return fmt.Errorf("Mount %s -> %s failed: %w", neighbor, label, err)
	}
	return nil
}

func unmount(client *mpd.Client, label string) error {
	if err := client.Command("unmount %s", label).OK(); err != nil {
		return fmt.Errorf("Unmount %s failed: %w", label, err)
	}
	return nil
}

func listMounts(client *mpd.Client) ([]mpd.Attrs, error) {
	res, err := client.Command("listmounts").AttrsList("mount")
	if err != nil {
		return nil, fmt.Errorf("Failed to list MPD mounts: %w", err)
	}
	return res, nil
}

func checkMountsOrUnmount(client *mpd.Client, mounts []mpd.Attrs) error {
	var errors []error
	for _, v := range mounts {
		if err := checkMountOrUnmount(client, v["storage"], v["mount"]); err != nil {
			errors = append(errors, fmt.Errorf("Failed to unmount %s: %w", v["mount"], err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors encountered while clearing mounts: %v", errors)
	}
	return nil
}

func checkMount(client *mpd.Client, storage, mount string) error {
	if mount == "" {
		return nil
	}
	if err := findNeighbor(client, storage); err != nil {
		return fmt.Errorf("%s mount does not exists: %w", mount, err)
	}
	return nil
}

func checkMountOrUnmount(client *mpd.Client, storage, mount string) error {
	if err := checkMount(client, storage, mount); err == nil {
		return nil
	}
	if err := unmount(client, mount); err != nil {
		return fmt.Errorf("Failed to unmount %s: %w", mount, err)
	}
	return nil
}
