package mpdplayer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fhs/gompd/v2/mpd"
)

// Playback sources, told from the path of the current song.
const (
	SourceDisc    = "disc"
	SourceMount   = "mount"
	SourceStream  = "stream"
	SourceLibrary = "library"
)

// Playback is the MPD playback state and the song it is on.
type Playback struct {
	State    string // play, pause or stop
	Source   string // one of the Source constants, "" without a current song
	Mount    string // name of the USB storage the song is on, for SourceMount
	Title    string // tags of the current song, "" when missing
	Artist   string
	Album    string
	Name     string   // stream name, sent by webradios
	URL      string   // stream URL, for SourceStream
	Position int      // position of the current song in the queue from 1, 0 without a current song
	Length   int      // number of songs in the queue
	Elapsed  *float64 // seconds played in the current song, nil without a current song
	Duration *float64 // length of the current song in seconds, nil when unknown, for a stream
}

// Status is the playback state and the USB storages mounted in MPD.
type Status struct {
	Playback Playback
	Mounts   []Mount
}

// Status reads the playback state, the current song and the USB mounts.
func (rc *ReconnectingMPDClient) Status() (Status, error) {
	var st Status
	err := rc.execute(func(client *mpd.Client) error {
		status, err := client.Status()
		if err != nil {
			return fmt.Errorf("failed to get MPD status: %w", err)
		}
		song, err := client.CurrentSong()
		if err != nil {
			return fmt.Errorf("failed to get current song: %w", err)
		}
		mounts, err := usbMounts(client)
		if err != nil {
			return err
		}
		st = Status{Playback: newPlayback(status, song, mounts), Mounts: mounts}
		return nil
	})
	return st, err
}

func newPlayback(status, song mpd.Attrs, mounts []Mount) Playback {
	file := song["file"]
	pb := Playback{
		State:    status["state"],
		Title:    song["Title"],
		Artist:   song["Artist"],
		Album:    song["Album"],
		Name:     song["Name"],
		Length:   atoi(status["playlistlength"]),
		Elapsed:  optionalFloat(status, "elapsed"),
		Duration: optionalFloat(status, "duration"),
	}
	if pos, ok := status["song"]; ok && file != "" {
		pb.Position = atoi(pos) + 1
	}
	pb.Source, pb.Mount = songSource(file, mounts)
	if pb.Source == SourceStream {
		pb.URL = file
	}
	return pb
}

// songSource tells where file comes from: a disc, a USB storage (along with
// its name), a stream or the MPD library.
func songSource(file string, mounts []Mount) (source, mount string) {
	switch {
	case file == "":
		return "", ""
	case strings.HasPrefix(file, CDDAPathPrefix):
		return SourceDisc, ""
	case strings.Contains(file, "://"):
		return SourceStream, ""
	}
	for _, m := range mounts {
		if strings.HasPrefix(file, MountPrefix(m.Name)) {
			return SourceMount, m.Name
		}
	}
	return SourceLibrary, ""
}

// atoi reads an MPD number, 0 when missing.
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// optionalFloat reads the MPD number under key, nil when missing.
func optionalFloat(attrs mpd.Attrs, key string) *float64 {
	f, err := strconv.ParseFloat(attrs[key], 64)
	if err != nil {
		return nil
	}
	return &f
}
