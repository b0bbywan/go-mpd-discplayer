package mpdplayer

import (
	"reflect"
	"testing"
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

			got, err := client.Status()
			if err != nil {
				t.Fatal(err)
			}
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
