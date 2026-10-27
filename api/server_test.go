package api

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/state"
)

type fakePlayer struct {
	calls  []string
	err    error
	disc   *state.Disc
	mounts []mpdplayer.Mount
	status mpdplayer.Status
	// coverPath and coverErr are what DiscCover returns
	coverPath string
	coverErr  error
}

func (f *fakePlayer) DiscCover(id string) (string, error) {
	f.calls = append(f.calls, "disc_cover "+id)
	return f.coverPath, f.coverErr
}

func (f *fakePlayer) Status() (mpdplayer.Status, error) {
	f.calls = append(f.calls, "status")
	return f.status, f.err
}

func (f *fakePlayer) State() state.State {
	f.calls = append(f.calls, "state")
	return state.State{Disc: f.disc, Mounts: f.mounts}
}

func (f *fakePlayer) PlayMount(name string) error {
	f.calls = append(f.calls, "play_mount "+name)
	return f.err
}

func (f *fakePlayer) StopMount(name string) error {
	f.calls = append(f.calls, "stop_mount "+name)
	return f.err
}

func (f *fakePlayer) EjectMount(name string) error {
	f.calls = append(f.calls, "eject_mount "+name)
	return f.err
}

func (f *fakePlayer) PlayDisc(drive string) error {
	f.calls = append(f.calls, "play")
	return f.err
}

func (f *fakePlayer) PlayURL(url string) error {
	f.calls = append(f.calls, "play_url "+url)
	return f.err
}

func (f *fakePlayer) Stop() error {
	f.calls = append(f.calls, "stop")
	return f.err
}

func (f *fakePlayer) StopDisc() error {
	f.calls = append(f.calls, "stop_disc")
	return f.err
}

func (f *fakePlayer) EjectDisc(drive string) error {
	f.calls = append(f.calls, "eject")
	return f.err
}

func serve(t *testing.T, player Player, method, path string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	return serveBody(t, player, method, path, header, "")
}

func serveBody(t *testing.T, player Player, method, path string, header http.Header, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	NewServer(Config{}, player).Handler().ServeHTTP(rec, req)
	return rec
}

func TestPlayerRoutes(t *testing.T) {
	tests := []struct {
		path string
		call string
	}{
		{"/player/stop", "stop"},
		{"/player/disc/play", "play"},
		{"/player/disc/stop", "stop_disc"},
		{"/player/disc/eject", "eject"},
		{"/player/mounts/MYUSB/play", "play_mount MYUSB"},
		{"/player/mounts/MYUSB/stop", "stop_mount MYUSB"},
		{"/player/mounts/MYUSB/eject", "eject_mount MYUSB"},
		{"/player/mounts/MY%20USB/play", "play_mount MY USB"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			player := &fakePlayer{}
			rec := serve(t, player, http.MethodPost, tt.path, nil)
			if rec.Code != http.StatusAccepted {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusAccepted)
			}
			if len(player.calls) != 1 || player.calls[0] != tt.call {
				t.Errorf("calls = %v, want [%s]", player.calls, tt.call)
			}
		})
	}
}

func TestStream(t *testing.T) {
	jsonType := http.Header{"Content-Type": {"application/json"}}
	tests := []struct {
		name   string
		header http.Header
		body   string
		code   int
	}{
		{"webradio", jsonType, `{"url": "https://stream.example.com/radio.mp3"}`, http.StatusAccepted},
		{"no content type", nil, `{"url": "https://stream.example.com/radio.mp3"}`, http.StatusUnsupportedMediaType},
		{"invalid json", jsonType, `{"url":`, http.StatusBadRequest},
		{"missing url", jsonType, `{}`, http.StatusBadRequest},
		{"local file", jsonType, `{"url": "file:///etc/passwd"}`, http.StatusBadRequest},
		{"relative path", jsonType, `{"url": "Music/album"}`, http.StatusBadRequest},
		{"too large", jsonType, `{"url": "https://` + strings.Repeat("a", maxRequestBodySize) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &fakePlayer{}
			rec := serveBody(t, player, http.MethodPost, "/player/stream/play", tt.header, tt.body)
			if rec.Code != tt.code {
				t.Errorf("status = %d, want %d (%s)", rec.Code, tt.code, rec.Body.String())
			}
			wantCalls := 0
			if tt.code == http.StatusAccepted {
				wantCalls = 1
			}
			if len(player.calls) != wantCalls {
				t.Errorf("calls = %v, want %d", player.calls, wantCalls)
			}
		})
	}
}

func TestStreamPassesURL(t *testing.T) {
	player := &fakePlayer{}
	serveBody(t, player, http.MethodPost, "/player/stream/play",
		http.Header{"Content-Type": {"application/json"}}, `{"url": "http://radio.example.com:8000/live"}`)
	if len(player.calls) != 1 || player.calls[0] != "play_url http://radio.example.com:8000/live" {
		t.Errorf("calls = %v", player.calls)
	}
}

func TestPlayerRouteError(t *testing.T) {
	player := &fakePlayer{err: errors.New("no disc")}
	rec := serve(t, player, http.MethodPost, "/player/disc/eject", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if body := rec.Body.String(); body != "no disc\n" {
		t.Errorf("body = %q, want %q", body, "no disc\n")
	}
}

func TestPlayerRouteNoDisc(t *testing.T) {
	for _, err := range []error{detect.ErrNoDisc, detect.ErrSeveralDiscs} {
		player := &fakePlayer{err: fmt.Errorf("looking for a disc: %w", err)}
		if rec := serve(t, player, http.MethodPost, "/player/disc/eject", nil); rec.Code != http.StatusConflict {
			t.Errorf("%v: status = %d, want %d", err, rec.Code, http.StatusConflict)
		}
	}
}

func TestPlayerRouteMPDError(t *testing.T) {
	player := &fakePlayer{err: fmt.Errorf("error stopping playback: %w", mpdplayer.ErrMPD)}
	if rec := serve(t, player, http.MethodPost, "/player/stop", nil); rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestPlayerRouteUnknownMount(t *testing.T) {
	player := &fakePlayer{err: fmt.Errorf("NAS: %w", mpdplayer.ErrUnknownMount)}
	if rec := serve(t, player, http.MethodPost, "/player/mounts/NAS/play", nil); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMounts(t *testing.T) {
	tests := []struct {
		name   string
		mounts []mpdplayer.Mount
		want   string
	}{
		{"mounts", []mpdplayer.Mount{{Name: "MYUSB", Storage: "udisks://a"}},
			`[{"name":"MYUSB","storage":"udisks://a"}]` + "\n"},
		{"none", nil, "[]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, &fakePlayer{mounts: tt.mounts}, http.MethodGet, "/player/mounts", nil)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q", got)
			}
			if got := rec.Body.String(); got != tt.want {
				t.Errorf("body = %q, want %q", got, tt.want)
			}
		})
	}
}

func ptr(f float64) *float64 { return &f }

func TestStatus(t *testing.T) {
	usb := []mpdplayer.Mount{{Name: "MUSIC", Storage: "udisks://by-uuid-3A4F-1C2B"}}
	at := time.Date(2026, 10, 24, 17, 46, 2, 345e6, time.UTC)
	const atJSON = `"at":"2026-10-24T17:46:02.345Z"`
	stopped := mpdplayer.Playback{At: at, State: "stop"}
	const stoppedJSON = `{` + atJSON + `,"state":"stop","source":null,"mount":null,"title":null,"artist":null,` +
		`"album":null,"name":null,"url":null,"position":null,"length":0,"elapsed":null,"duration":null}`
	tests := []struct {
		name     string
		playback mpdplayer.Playback
		disc     *state.Disc
		mounts   []mpdplayer.Mount
		want     string
	}{
		{"disc", mpdplayer.Playback{
			At: at, State: "play", Source: mpdplayer.SourceDisc,
			Title: "Bruxelles", Artist: "Benabar", Album: "Reprise des negociations",
			Position: 4, Length: 12, Elapsed: ptr(102.3), Duration: ptr(224),
		}, &state.Disc{
			ID: "9c0bf40c", Device: "/dev/sr0", Status: state.Ready,
			Album: "Reprise des negociations", Artist: "Benabar", Date: "2005", Genre: "Chanson",
			Tracks: []string{"Le diner", "Bruxelles"}, Cover: "/cache/9c0bf40c/cover.jpg",
		}, usb, `{"playback":{` + atJSON + `,"state":"play","source":"disc","mount":null,"title":"Bruxelles","artist":"Benabar",` +
			`"album":"Reprise des negociations","name":null,"url":null,"position":4,"length":12,"elapsed":102.3,"duration":224},` +
			`"disc":{"id":"9c0bf40c","device":"/dev/sr0","status":"ready","album":"Reprise des negociations","artist":"Benabar",` +
			`"date":"2005","genre":"Chanson","tracks":["Le diner","Bruxelles"],"cover":null},` +
			`"mounts":[{"name":"MUSIC","storage":"udisks://by-uuid-3A4F-1C2B"}]}`},
		{"mount", mpdplayer.Playback{
			At: at.In(time.FixedZone("CET", 3600)), State: "pause", Source: mpdplayer.SourceMount, Mount: "MUSIC",
			Position: 1, Length: 1, Elapsed: ptr(0), Duration: ptr(60),
		}, nil, usb, `{"playback":{` + atJSON + `,"state":"pause","source":"mount","mount":"MUSIC","title":null,"artist":null,` +
			`"album":null,"name":null,"url":null,"position":1,"length":1,"elapsed":0,"duration":60},` +
			`"disc":null,"mounts":[{"name":"MUSIC","storage":"udisks://by-uuid-3A4F-1C2B"}]}`},
		{"stream", mpdplayer.Playback{
			At: at, State: "play", Source: mpdplayer.SourceStream, URL: "https://radio.example/live",
			Title: "Live", Name: "Radio", Position: 1, Length: 1, Elapsed: ptr(12),
		}, nil, nil, `{"playback":{` + atJSON + `,"state":"play","source":"stream","mount":null,"title":"Live","artist":null,` +
			`"album":null,"name":"Radio","url":"https://radio.example/live","position":1,"length":1,"elapsed":12,"duration":null},` +
			`"disc":null,"mounts":[]}`},
		{"stopped", stopped, nil, nil, `{"playback":` + stoppedJSON + `,"disc":null,"mounts":[]}`},
		{"disc resolving", stopped, &state.Disc{Device: "/dev/sr0", Status: state.Resolving}, nil,
			`{"playback":` + stoppedJSON + `,"disc":{"id":null,"device":"/dev/sr0","status":"resolving",` +
				`"album":null,"artist":null,"date":null,"genre":null,"tracks":null,"cover":null},"mounts":[]}`},
		{"unknown disc", stopped, &state.Disc{Device: "/dev/sr0", Status: state.Ready, Tracks: []string{"", ""}}, nil,
			`{"playback":` + stoppedJSON + `,"disc":{"id":null,"device":"/dev/sr0","status":"ready",` +
				`"album":null,"artist":null,"date":null,"genre":null,"tracks":["",""],"cover":null},"mounts":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &fakePlayer{status: mpdplayer.Status{Playback: tt.playback}, disc: tt.disc, mounts: tt.mounts}
			rec := serve(t, player, http.MethodGet, "/player/status", nil)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q", got)
			}
			if got := rec.Body.String(); got != tt.want+"\n" {
				t.Errorf("body = %s\nwant   %s", got, tt.want)
			}
		})
	}
}

func TestStatusCoverURL(t *testing.T) {
	cover := filepath.Join(t.TempDir(), "cover.jpg")
	if err := os.WriteFile(cover, []byte("jpeg"), 0o644); err != nil {
		t.Fatal(err)
	}
	modTime := time.Date(2026, 10, 24, 17, 46, 2, 345e6, time.UTC)
	if err := os.Chtimes(cover, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		disc *state.Disc
		want string
	}{
		{"cover", &state.Disc{ID: "9c0bf40c", Cover: cover}, fmt.Sprintf(`"cover":"discs/9c0bf40c/cover?v=%d"`, modTime.UnixMilli())},
		{"no cover", &state.Disc{ID: "9c0bf40c"}, `"cover":null`},
		{"cover file gone", &state.Disc{ID: "9c0bf40c", Cover: cover + ".gone"}, `"cover":null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, &fakePlayer{disc: tt.disc}, http.MethodGet, "/player/status", nil)
			if got := rec.Body.String(); !strings.Contains(got, tt.want) {
				t.Errorf("body = %s, want it to hold %s", got, tt.want)
			}
		})
	}
}

func TestDiscCover(t *testing.T) {
	cover := filepath.Join(t.TempDir(), "cover.jpg")
	if err := os.WriteFile(cover, []byte("jpeg"), 0o644); err != nil {
		t.Fatal(err)
	}
	player := &fakePlayer{coverPath: cover}
	rec := serve(t, player, http.MethodGet, "/discs/9c0bf40c/cover?v=1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "jpeg" {
		t.Errorf("body = %q, want the cover", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q", got)
	}
	if want := []string{"disc_cover 9c0bf40c"}; !reflect.DeepEqual(player.calls, want) {
		t.Errorf("calls = %v, want %v", player.calls, want)
	}
}

func TestDiscCoverErrors(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		player   *fakePlayer
		want     int
		wantCall bool
	}{
		{"no cover", "/discs/9c0bf40c/cover", &fakePlayer{coverErr: fs.ErrNotExist}, http.StatusNotFound, true},
		{"cover file gone", "/discs/9c0bf40c/cover", &fakePlayer{coverPath: "/nonexistent/cover.jpg"}, http.StatusNotFound, true},
		{"cache unreadable", "/discs/9c0bf40c/cover", &fakePlayer{coverErr: fs.ErrPermission}, http.StatusInternalServerError, true},
		// Only a disc id reaches the player
		{"too short", "/discs/9c0bf40/cover", &fakePlayer{}, http.StatusNotFound, false},
		{"not hexadecimal", "/discs/9c0bf40g/cover", &fakePlayer{}, http.StatusNotFound, false},
		{"upper case", "/discs/9C0BF40C/cover", &fakePlayer{}, http.StatusNotFound, false},
		{"escaped path", "/discs/..%2F..%2Fetc/cover", &fakePlayer{}, http.StatusNotFound, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, tt.player, http.MethodGet, tt.path, nil)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
			if called := len(tt.player.calls) > 0; called != tt.wantCall {
				t.Errorf("calls = %v, want a call: %v", tt.player.calls, tt.wantCall)
			}
		})
	}
}

func TestStatusError(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{fmt.Errorf("error reading status: %w", mpdplayer.ErrMPD), http.StatusBadGateway},
		{errors.New("other failure"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		if rec := serve(t, &fakePlayer{err: tt.err}, http.MethodGet, "/player/status", nil); rec.Code != tt.want {
			t.Errorf("%v: status = %d, want %d", tt.err, rec.Code, tt.want)
		}
	}
}

func TestPlayerRoutesArePostOnly(t *testing.T) {
	player := &fakePlayer{}
	if rec := serve(t, player, http.MethodGet, "/player/disc/play", nil); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if rec := serve(t, player, http.MethodPost, "/", nil); rec.Code != http.StatusNotFound {
		t.Errorf("POST / status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if len(player.calls) != 0 {
		t.Errorf("calls = %v, want none", player.calls)
	}
}

func TestCORS(t *testing.T) {
	origin := http.Header{"Origin": {"https://pwa.odio.love"}}

	rec := serve(t, &fakePlayer{}, http.MethodPost, "/player/stop", origin)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}

	rec = serve(t, &fakePlayer{}, http.MethodPost, "/player/stop", nil)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin without Origin = %q, want none", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	player := &fakePlayer{}
	rec := serve(t, player, http.MethodOptions, "/player/disc/eject", http.Header{"Origin": {"https://pwa.odio.love"}})
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
	if len(player.calls) != 0 {
		t.Errorf("preflight reached the player: %v", player.calls)
	}
}

func TestRunUnixSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "api.sock")
	// A socket left behind by a crashed run is replaced
	stale, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	stale.(*net.UnixListener).SetUnlinkOnClose(false)
	_ = stale.Close()

	player := &fakePlayer{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- NewServer(Config{Socket: socket}, player).Run(ctx) }()

	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}}
	var resp *http.Response
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if resp, err = client.Post("http://localhost/player/disc/eject", "", nil); err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("POST over unix socket: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted || len(player.calls) != 1 {
		t.Errorf("status = %d, calls = %v", resp.StatusCode, player.calls)
	}
	if fi, err := os.Stat(socket); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("socket mode = %v, err = %v, want 0600", fi.Mode().Perm(), err)
	}

	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run() = %v, want nil after shutdown", err)
	}
	if _, err := os.Stat(socket); !os.IsNotExist(err) {
		t.Errorf("socket still present after shutdown: %v", err)
	}
}

func TestRunWithoutListener(t *testing.T) {
	if err := NewServer(Config{}, &fakePlayer{}).Run(context.Background()); err == nil {
		t.Error("Run() without port nor socket succeeded")
	}
}
