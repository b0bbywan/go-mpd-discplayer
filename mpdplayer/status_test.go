package mpdplayer

import (
	"reflect"
	"testing"
	"time"
)

func ptr(f float64) *float64 { return &f }

func TestStatus(t *testing.T) {
	queue := []string{"cdda:///1", "cdda:///2", "MYUSB/a.flac", "https://radio.example/live", "Albums/b.flac"}
	usb := Mount{Name: "MYUSB", Storage: "udisks://by-uuid-AAAA"}
	tests := []struct {
		name     string
		current  int
		tags     map[string]string
		elapsed  string
		duration string
		want     Playback
	}{
		{"stopped", -1, nil, "", "", Playback{State: "stop", Length: 5}},
		{
			"disc", 1, map[string]string{"Title": "Bruxelles", "AlbumArtist": "Benabar", "Album": "Reprise des negociations"}, "102.3", "224.000",
			Playback{State: "play", Source: SourceDisc, Title: "Bruxelles", Artist: "Benabar",
				Album: "Reprise des negociations", Position: 2, Length: 5, Elapsed: ptr(102.3), Duration: ptr(224)},
		},
		{"mount", 2, nil, "1.5", "60", Playback{State: "play", Source: SourceMount, Mount: "MYUSB",
			Position: 3, Length: 5, Elapsed: ptr(1.5), Duration: ptr(60)}},
		{"stream", 3, map[string]string{"Name": "Radio", "Title": "Live"}, "12.0", "",
			Playback{State: "play", Source: SourceStream, Title: "Live", Name: "Radio", URL: "https://radio.example/live",
				Position: 4, Length: 5, Elapsed: ptr(12)}},
		{"library", 4, map[string]string{"Artist": "Feat", "AlbumArtist": "Band"}, "", "",
			Playback{State: "play", Source: SourceLibrary, Artist: "Feat", Position: 5, Length: 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeMPD()
			f.queue, f.current = queue, tt.current
			f.tags, f.elapsed, f.duration = tt.tags, tt.elapsed, tt.duration
			f.mounts = []fakeMount{{usb.Name, usb.Storage}, {"NAS", "nfs://nas/music"}}
			client := f.start(t)

			before := time.Now()
			got, err := client.Status()
			if err != nil {
				t.Fatal(err)
			}
			if got.Playback.At.Before(before) || got.Playback.At.After(time.Now()) {
				t.Errorf("At = %v, want the time of the call", got.Playback.At)
			}
			got.Playback.At = time.Time{}
			if !reflect.DeepEqual(got.Playback, tt.want) {
				t.Errorf("Playback = %+v, want %+v", got.Playback, tt.want)
			}
			if !reflect.DeepEqual(got.Mounts, []Mount{usb}) {
				t.Errorf("Mounts = %v, want [%v]", got.Mounts, usb)
			}
		})
	}
}

func TestSongSource(t *testing.T) {
	mounts := []Mount{{Name: "MYUSB"}}
	tests := []struct {
		file, source, mount string
	}{
		{"MYUSB2/a.flac", SourceLibrary, ""},
		{"MYUSB/a.flac", SourceMount, "MYUSB"},
		{"http://radio.example/live", SourceStream, ""},
		{"cdda:///1", SourceDisc, ""},
		// A disc track added from the CUE sheet in the library
		{".disc-cuer/6507c409/playlist.cue/track0001", SourceDisc, ""},
		{".disc-cuer2/a.flac", SourceLibrary, ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		if source, mount := songSource(tt.file, discPrefixes(".disc-cuer"), mounts); source != tt.source || mount != tt.mount {
			t.Errorf("songSource(%q) = %q, %q, want %q, %q", tt.file, source, mount, tt.source, tt.mount)
		}
	}
}

func TestQueuedDisc(t *testing.T) {
	f := newFakeMPD()
	f.playlists = map[string][]string{"/cache/6507c409/playlist.cue": {"cdda:///1", "cdda:///2"}}
	f.songTags = map[string]map[string]string{
		"cdda:///1": {"Title": "Retroaction", "Album": "A Noisy Blast", "AlbumArtist": "Mat3r Dolorosa", "Date": "2016-09-23"},
		"cdda:///2": {"Title": "Paradise", "Album": "A Noisy Blast", "AlbumArtist": "Mat3r Dolorosa", "Date": "2016-09-23"},
	}
	client := f.start(t)
	if err := client.StartDiscPlayback("/dev/sr0", "/cache/6507c409/playlist.cue"); err != nil {
		t.Fatal(err)
	}
	// Other songs added by hand are left out
	f.mu.Lock()
	f.queue = append(f.queue, "Albums/b.flac")
	f.mu.Unlock()

	got, err := client.QueuedDisc()
	if err != nil {
		t.Fatal(err)
	}
	want := QueuedDisc{
		CuePath: "/cache/6507c409/playlist.cue",
		Album:   "A Noisy Blast", Artist: "Mat3r Dolorosa", Date: "2016-09-23",
		Tracks: []string{"Retroaction", "Paradise"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("QueuedDisc() = %+v, want %+v", got, want)
	}
}

func TestQueuedDiscAddedFromLibrary(t *testing.T) {
	f := newFakeMPD()
	tracks := []string{".disc-cuer/6507c409/playlist.cue/track0001", ".disc-cuer/6507c409/playlist.cue/track0002"}
	f.queue = tracks
	f.songTags = map[string]map[string]string{
		tracks[0]: {"Title": "Retroaction", "Album": "A Noisy Blast", "AlbumArtist": "Mat3r Dolorosa"},
		tracks[1]: {"Title": "Paradise", "Album": "A Noisy Blast", "AlbumArtist": "Mat3r Dolorosa"},
	}
	client := f.start(t)

	got, err := client.QueuedDisc()
	if err != nil {
		t.Fatal(err)
	}
	// Added rather than loaded: no CUE sheet path, the tags only
	want := QueuedDisc{Album: "A Noisy Blast", Artist: "Mat3r Dolorosa", Tracks: []string{"Retroaction", "Paradise"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("QueuedDisc() = %+v, want %+v", got, want)
	}
}

func TestQueuedDiscWithoutDisc(t *testing.T) {
	f := newFakeMPD()
	f.queue = []string{"Albums/b.flac"}
	client := f.start(t)

	got, err := client.QueuedDisc()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, QueuedDisc{}) {
		t.Errorf("QueuedDisc() = %+v, want none", got)
	}
}
