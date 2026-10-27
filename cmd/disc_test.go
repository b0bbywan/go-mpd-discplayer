package cmd

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/b0bbywan/go-disc-cuer/cue"
	"github.com/b0bbywan/go-disc-cuer/types"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/state"
)

const testDrive = "/dev/sr1"

var testInfo = &types.DiscInfo{
	Artist:       "Benabar",
	Title:        "Reprise des negociations",
	ReleaseDate:  "2005",
	Genre:        "Chanson",
	Tracks:       []string{"Le diner", "Bruxelles"},
	CoverArtPath: "/cache/9c0bf40c/cover.jpg",
}

// readyDisc is the state of the disc of testInfo in testDrive.
var readyDisc = &state.Disc{
	ID:     "9c0bf40c",
	Device: testDrive,
	Status: state.Ready,
	Album:  "Reprise des negociations",
	Artist: "Benabar",
	Date:   "2005",
	Genre:  "Chanson",
	Tracks: []string{"Le diner", "Bruxelles"},
	Cover:  "/cache/9c0bf40c/cover.jpg",
}

// newDiscTestPlayer returns a player with cuer and a disc file in a temporary
// directory, reading 3 tracks from any TOC.
func newDiscTestPlayer(t *testing.T, client *fakeClient, cuer *fakeCuer) *Player {
	t.Helper()
	orig := readTrackCount
	readTrackCount = func(string) (int, error) { return 3, nil }
	t.Cleanup(func() { readTrackCount = orig })

	p := newTestPlayer(client, &fakeMounter{})
	p.Cuer = cuer
	p.discFile = state.DiscFile(filepath.Join(t.TempDir(), "disc.json"))
	return p
}

func assertDisc(t *testing.T, p *Player, want *state.Disc) {
	t.Helper()
	if got := p.store.Load().Disc; !reflect.DeepEqual(got, want) {
		t.Errorf("disc = %+v, want %+v", got, want)
	}
}

// assertDiscFile checks the disc file holds want, or is missing for nil.
func assertDiscFile(t *testing.T, p *Player, want *state.SavedDisc) {
	t.Helper()
	got, err := p.discFile.Load()
	if want == nil {
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("disc file = %+v (error %v), want none", got, err)
		}
		return
	}
	if err != nil || got != *want {
		t.Errorf("disc file = %+v (error %v), want %+v", got, err, *want)
	}
}

func seedDiscFile(t *testing.T, p *Player, saved state.SavedDisc) {
	t.Helper()
	if err := p.discFile.Save(saved); err != nil {
		t.Fatal(err)
	}
}

func TestInsertDisc(t *testing.T) {
	client := &fakeClient{}
	cuer := &fakeCuer{res: &cue.Result{ID: "9c0bf40c", CuePath: "/cache/9c0bf40c/playlist.cue", Info: testInfo}}
	p := newDiscTestPlayer(t, client, cuer)
	cuer.onResolve = func() {
		assertDisc(t, p, &state.Disc{Device: testDrive, Status: state.Resolving})
	}

	if err := p.insertDisc(testDrive); err != nil {
		t.Fatalf("insertDisc() error = %v", err)
	}
	assertDisc(t, p, readyDisc)
	assertDiscFile(t, p, &state.SavedDisc{ID: "9c0bf40c", Device: testDrive})
	if want := []string{"StartDiscPlayback /dev/sr1 /cache/9c0bf40c/playlist.cue"}; !reflect.DeepEqual(client.calls, want) {
		t.Errorf("client calls = %v, want %v", client.calls, want)
	}
}

func TestInsertUnknownDisc(t *testing.T) {
	client := &fakeClient{}
	p := newDiscTestPlayer(t, client, &fakeCuer{err: errFake})
	// The file of a previous disc never stands for this one
	seedDiscFile(t, p, state.SavedDisc{ID: "old", Device: testDrive})

	if err := p.insertDisc(testDrive); err != nil {
		t.Fatalf("insertDisc() error = %v", err)
	}
	assertDisc(t, p, &state.Disc{Device: testDrive, Status: state.Ready, Tracks: []string{"", "", ""}})
	assertDiscFile(t, p, nil)
	if want := []string{"StartDiscPlayback /dev/sr1"}; !reflect.DeepEqual(client.calls, want) {
		t.Errorf("client calls = %v, want %v", client.calls, want)
	}
}

func TestInsertDiscPlaybackError(t *testing.T) {
	client := &fakeClient{fail: map[string]bool{"StartDiscPlayback": true}}
	p := newDiscTestPlayer(t, client, &fakeCuer{res: &cue.Result{ID: "9c0bf40c", Info: testInfo}})

	if err := p.insertDisc(testDrive); err == nil {
		t.Fatal("insertDisc() expected the playback error, got nil")
	}
	// The disc is in the drive all the same
	assertDisc(t, p, readyDisc)
}

func TestRemoveDisc(t *testing.T) {
	p := newDiscTestPlayer(t, &fakeClient{}, &fakeCuer{})
	p.setDisc(readyDisc)
	seedDiscFile(t, p, state.SavedDisc{ID: "9c0bf40c", Device: testDrive})

	if err := callHandler(t, p, detect.DeviceDisc, detect.DeviceRemoved); err != nil {
		t.Fatalf("remove handler error = %v", err)
	}
	assertDisc(t, p, nil)
	assertDiscFile(t, p, nil)
}

func TestPresentDisc(t *testing.T) {
	playingDisc := mpdplayer.Playback{State: "play", Source: mpdplayer.SourceDisc}
	resolved := &cue.Result{ID: "9c0bf40c", Info: testInfo}
	// The queue loaded from the cached sheet of testInfo, the cache being /cache
	loaded := mpdplayer.QueuedDisc{
		CuePath: "/cache/9c0bf40c/playlist.cue",
		Album:   "Reprise des negociations", Artist: "Benabar", Date: "2005", Genre: "Chanson",
		Tracks: []string{"Le diner", "Bruxelles"},
	}
	tagged := loaded
	tagged.CuePath = ""
	taggedDisc := &state.Disc{
		Device: testDrive, Status: state.Ready,
		Album: "Reprise des negociations", Artist: "Benabar", Date: "2005", Genre: "Chanson",
		Tracks: []string{"Le diner", "Bruxelles"},
	}
	edited := loaded
	edited.Tracks = []string{"Le diner"}
	foreign := loaded
	foreign.CuePath = "/elsewhere/9c0bf40c/playlist.cue"
	tests := []struct {
		name       string
		saved      *state.SavedDisc
		cuer       *fakeCuer
		playback   mpdplayer.Playback
		queued     mpdplayer.QueuedDisc
		failStatus bool
		failQueue  bool
		wantDisc   *state.Disc
		wantFile   *state.SavedDisc
		wantCuer   []string
		wantClient []string
		wantErr    bool
	}{
		{
			name:     "restored from the disc file, without reading the TOC nor MPD",
			saved:    &state.SavedDisc{ID: "9c0bf40c", Device: testDrive},
			cuer:     &fakeCuer{info: testInfo},
			wantDisc: readyDisc,
			wantFile: &state.SavedDisc{ID: "9c0bf40c", Device: testDrive},
			wantCuer: []string{"Cached 9c0bf40c"},
		},
		{
			name:       "disc file of another drive, disc not playing: TOC read",
			saved:      &state.SavedDisc{ID: "other", Device: "/dev/sr0"},
			cuer:       &fakeCuer{res: resolved},
			wantDisc:   readyDisc,
			wantFile:   &state.SavedDisc{ID: "9c0bf40c", Device: testDrive},
			wantCuer:   []string{"Resolve /dev/sr1"},
			wantClient: []string{"Status"},
		},
		{
			name:       "cached sheet gone, disc not playing: TOC read",
			saved:      &state.SavedDisc{ID: "9c0bf40c", Device: testDrive},
			cuer:       &fakeCuer{res: resolved, cachedErr: errFake},
			wantDisc:   readyDisc,
			wantFile:   &state.SavedDisc{ID: "9c0bf40c", Device: testDrive},
			wantCuer:   []string{"Cached 9c0bf40c", "Resolve /dev/sr1"},
			wantClient: []string{"Status"},
		},
		{
			name:       "no disc file, disc playing from its cached sheet: TOC left alone, file written",
			cuer:       &fakeCuer{info: testInfo, cacheDir: "/cache"},
			playback:   playingDisc,
			queued:     loaded,
			wantDisc:   readyDisc,
			wantFile:   &state.SavedDisc{ID: "9c0bf40c", Device: testDrive},
			wantCuer:   []string{"Cached 9c0bf40c"},
			wantClient: []string{"Status", "QueuedDisc"},
		},
		{
			name:       "no disc file, disc paused from its cached sheet: TOC left alone",
			cuer:       &fakeCuer{info: testInfo, cacheDir: "/cache"},
			playback:   mpdplayer.Playback{State: "pause", Source: mpdplayer.SourceDisc},
			queued:     loaded,
			wantDisc:   readyDisc,
			wantFile:   &state.SavedDisc{ID: "9c0bf40c", Device: testDrive},
			wantCuer:   []string{"Cached 9c0bf40c"},
			wantClient: []string{"Status", "QueuedDisc"},
		},
		{
			name:       "no disc file, disc playing without sheet: read from the track tags",
			cuer:       &fakeCuer{info: testInfo, cacheDir: "/cache"},
			playback:   playingDisc,
			queued:     tagged,
			wantDisc:   taggedDisc,
			wantClient: []string{"Status", "QueuedDisc"},
		},
		{
			name:       "no disc file, queue edited since the sheet was loaded: read from the track tags",
			cuer:       &fakeCuer{info: testInfo, cacheDir: "/cache"},
			playback:   playingDisc,
			queued:     edited,
			wantDisc:   &state.Disc{Device: testDrive, Status: state.Ready, Album: "Reprise des negociations", Artist: "Benabar", Date: "2005", Genre: "Chanson", Tracks: []string{"Le diner"}},
			wantCuer:   []string{"Cached 9c0bf40c"},
			wantClient: []string{"Status", "QueuedDisc"},
		},
		{
			name:       "no disc file, sheet loaded from outside the cache: read from the track tags",
			cuer:       &fakeCuer{info: testInfo, cacheDir: "/cache"},
			playback:   playingDisc,
			queued:     foreign,
			wantDisc:   taggedDisc,
			wantClient: []string{"Status", "QueuedDisc"},
		},
		{
			name:       "no disc file, queue unreadable: device only",
			cuer:       &fakeCuer{info: testInfo, cacheDir: "/cache"},
			playback:   playingDisc,
			failQueue:  true,
			wantDisc:   &state.Disc{Device: testDrive, Status: state.Ready},
			wantClient: []string{"Status", "QueuedDisc"},
		},
		{
			name:       "no disc file, disc stopped: TOC read",
			cuer:       &fakeCuer{res: resolved},
			playback:   mpdplayer.Playback{State: "stop", Source: mpdplayer.SourceDisc},
			wantDisc:   readyDisc,
			wantFile:   &state.SavedDisc{ID: "9c0bf40c", Device: testDrive},
			wantCuer:   []string{"Resolve /dev/sr1"},
			wantClient: []string{"Status"},
		},
		{
			name:       "no disc file, MPD unreachable: TOC left alone",
			cuer:       &fakeCuer{res: resolved},
			failStatus: true,
			wantDisc:   &state.Disc{Device: testDrive, Status: state.Ready},
			wantClient: []string{"Status"},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{playback: tt.playback, queued: tt.queued,
				fail: map[string]bool{"Status": tt.failStatus, "QueuedDisc": tt.failQueue}}
			p := newDiscTestPlayer(t, client, tt.cuer)
			if tt.saved != nil {
				seedDiscFile(t, p, *tt.saved)
			}

			if err := p.presentDisc(testDrive); (err != nil) != tt.wantErr {
				t.Errorf("presentDisc() error = %v, wantErr %v", err, tt.wantErr)
			}
			assertDisc(t, p, tt.wantDisc)
			assertDiscFile(t, p, tt.wantFile)
			if !reflect.DeepEqual(tt.cuer.calls, tt.wantCuer) {
				t.Errorf("cuer calls = %v, want %v", tt.cuer.calls, tt.wantCuer)
			}
			// Playback never starts at startup
			if !reflect.DeepEqual(client.calls, tt.wantClient) {
				t.Errorf("client calls = %v, want %v", client.calls, tt.wantClient)
			}
		})
	}
}
