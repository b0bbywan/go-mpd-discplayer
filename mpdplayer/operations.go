package mpdplayer

import (
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/fhs/gompd/v2/mpd"

	"github.com/b0bbywan/go-disc-cuer/config"
	"github.com/b0bbywan/go-disc-cuer/cue"

	"github.com/b0bbywan/go-mpd-discplayer/internal/retry"
)

const (
	CDDAPathPrefix = "cdda://"
	udisksScheme   = "udisks://"
)

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

// Stop stops playback, whatever is playing. The queue is left untouched.
func (rc *ReconnectingMPDClient) Stop() error {
	return rc.execute(func(client *mpd.Client) error {
		if err := client.Stop(); err != nil {
			return fmt.Errorf("error: Failed to stop MPD playback: %w", err)
		}
		return nil
	})
}

// DiscPrefixes are the path prefixes of the disc tracks: CDDA tracks, and
// the tracks of the disc CUE sheets when MPD lists them in its library.
func (rc *ReconnectingMPDClient) DiscPrefixes() []string {
	return discPrefixes(rc.mpcConfig.CueFolder)
}

func discPrefixes(cueFolder string) []string {
	if cueFolder == "" {
		return []string{CDDAPathPrefix}
	}
	return []string{CDDAPathPrefix, cueFolder + "/"}
}

// StopIfPlaying stops playback when the current song path starts with one
// of prefixes. The queue is left untouched.
func (rc *ReconnectingMPDClient) StopIfPlaying(prefixes ...string) error {
	return rc.execute(func(client *mpd.Client) error {
		return stopIfPlaying(client, prefixes)
	})
}

// StopAndRemove stops playback when the current song path starts with one of
// prefixes, then removes the songs under them from the queue.
func (rc *ReconnectingMPDClient) StopAndRemove(prefixes ...string) error {
	return rc.execute(func(client *mpd.Client) error {
		if err := stopIfPlaying(client, prefixes); err != nil {
			return err
		}
		return deleteFromPlaylist(client, prefixes)
	})
}

func (rc *ReconnectingMPDClient) Mount(identifiers []string, label string) error {
	return rc.execute(func(client *mpd.Client) error {
		// MPD may have restored the mount from its state file
		if isMounted(client, identifiers, label) {
			return nil
		}
		neighborURI, err := findNeighbor(client, identifiers)
		if err != nil {
			return fmt.Errorf("failed to find neighbor for %v: %w", identifiers, err)
		}
		if err := mount(client, neighborURI, label); err != nil {
			return fmt.Errorf("failed to mount %s -> %s: %w", neighborURI, label, err)
		}
		return nil
	})
}

// UpdateDB triggers a database update of uri without waiting for it.
func (rc *ReconnectingMPDClient) UpdateDB(uri string) error {
	return rc.execute(func(client *mpd.Client) error {
		if _, err := client.Update(uri); err != nil {
			return fmt.Errorf("failed to update database: %w", err)
		}
		return nil
	})
}

// Mount is a USB storage mounted in MPD.
type Mount struct {
	Name    string // mount path in the MPD library
	Storage string // udisks URI
}

// USBMounts lists the USB storages mounted in MPD. Other storages (nfs,
// smb...) are left out.
func (rc *ReconnectingMPDClient) USBMounts() ([]Mount, error) {
	var usb []Mount
	err := rc.execute(func(client *mpd.Client) error {
		var err error
		usb, err = usbMounts(client)
		return err
	})
	return usb, err
}

// MountPrefix is the path prefix of the files of the mount named name, the
// trailing slash keeping MYUSB from matching MYUSB2.
func MountPrefix(name string) string {
	return name + "/"
}

// Unmount unmounts label from MPD. A label already unmounted, by an eject
// from the API for instance, is not an error.
func (rc *ReconnectingMPDClient) Unmount(label string) error {
	return rc.execute(func(client *mpd.Client) error {
		mounts, err := listMounts(client)
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(mounts, func(m mpd.Attrs) bool { return m["mount"] == label }) {
			return nil
		}
		if err := unmount(client, label); err != nil {
			return fmt.Errorf("failed to unmount %s: %w", label, err)
		}
		return nil
	})
}

// ClearStaleMounts unmounts udisks mounts whose storage matches none of the
// identifiers of the devices currently plugged in. Other storages (nfs, smb...)
// are left untouched.
func (rc *ReconnectingMPDClient) ClearStaleMounts(present []string) error {
	return rc.execute(func(client *mpd.Client) error {
		mounts, err := listMounts(client)
		if err != nil {
			return fmt.Errorf("failed to list mounts: %w", err)
		}

		var errs []error
		for _, m := range mounts {
			storage, label := m["storage"], m["mount"]
			if label == "" || !strings.HasPrefix(storage, udisksScheme) || containsAny(storage, present) {
				continue
			}
			log.Printf("Unmounting stale mount %s (%s)", label, storage)
			if err := unmount(client, label); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("errors encountered while clearing mounts: %v", errs)
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
		return "", fmt.Errorf("failed to get MPD config from server: %w", err)
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
		return fmt.Errorf("no Cuer config to generate from")
	}
	cueFilePath, err := cue.New(cuerConfig).Generate(cue.Options{Device: device})
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

func stopIfPlaying(client *mpd.Client, prefixes []string) error {
	if !checkPathPlaying(client, prefixes) {
		return nil
	}
	if err := client.Stop(); err != nil {
		return fmt.Errorf("error: Failed to stop MPD playback: %w", err)
	}
	return nil
}

func checkPathPlaying(client *mpd.Client, prefixes []string) bool {
	song, err := client.CurrentSong()
	if err != nil {
		return true
	}

	// Check if a song is currently playing
	if len(song) == 0 {
		return true
	}

	return checkSongPath(song, prefixes)
}

func checkSongPath(song mpd.Attrs, prefixes []string) bool {
	path, ok := song["file"]
	return ok && hasAnyPrefix(path, prefixes)
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// addUSBToQueue adds the specified label to the playlist.
func addUSBToQueue(client *mpd.Client, label string) error {
	if err := UpdateDBAndWait(client, label); err != nil {
		return fmt.Errorf("database update failed: %w", err)
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

func deleteFromPlaylist(client *mpd.Client, prefixes []string) error {
	playlist, err := client.PlaylistInfo(-1, -1)
	if err != nil {
		return fmt.Errorf("failed to fetch MPD playlist: %w", err)
	}

	start := -1 // Initialize start index to -1 to indicate no active range

	for i := len(playlist) - 1; i >= -1; i-- {
		if i >= 0 && checkSongPath(playlist[i], prefixes) {
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
		return fmt.Errorf("failed to update database: %w", err)
	}
	if err := retry.Until(30*time.Second, 500*time.Millisecond, func() error {
		if DbUpdating(client) {
			return errors.New("database did not finish update within timeout")
		}
		return nil
	}); err != nil {
		return err
	}
	log.Println("Database update finished.")
	return nil
}

func DbUpdating(client *mpd.Client) bool {
	status, err := client.Status()
	if err != nil {
		return true
	}
	_, updating := status["updating_db"]
	return updating
}

func findNeighbor(client *mpd.Client, identifiers []string) (string, error) {
	res, err := client.Command("listneighbors").AttrsList("neighbor")
	if err != nil {
		return "", fmt.Errorf("failed to list neighbors, is udisk neighbor plugin enabled?: %w", err)
	}
	for _, v := range res {
		if uri := v["neighbor"]; containsAny(uri, identifiers) {
			return uri, nil
		}
	}
	var neighbors []string
	for _, v := range res {
		neighbors = append(neighbors, v["neighbor"])
	}
	return "", fmt.Errorf("no neighbor found matching %v in %v", identifiers, neighbors)
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
		return nil, fmt.Errorf("failed to list MPD mounts: %w", err)
	}
	return res, nil
}

func usbMounts(client *mpd.Client) ([]Mount, error) {
	mounts, err := listMounts(client)
	if err != nil {
		return nil, err
	}
	var usb []Mount
	for _, m := range mounts {
		if m["mount"] != "" && strings.HasPrefix(m["storage"], udisksScheme) {
			usb = append(usb, Mount{Name: m["mount"], Storage: m["storage"]})
		}
	}
	return usb, nil
}

// isMounted reports whether label is already mounted on a storage matching
// one of the identifiers.
func isMounted(client *mpd.Client, identifiers []string, label string) bool {
	mounts, err := listMounts(client)
	if err != nil {
		return false
	}
	for _, m := range mounts {
		if m["mount"] == label && containsAny(m["storage"], identifiers) {
			return true
		}
	}
	return false
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if sub != "" && strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
