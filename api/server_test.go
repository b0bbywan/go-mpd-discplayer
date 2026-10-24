package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

type fakePlayer struct {
	calls  []string
	err    error
	mounts []mpdplayer.Mount
}

func (f *fakePlayer) Mounts() ([]mpdplayer.Mount, error) {
	f.calls = append(f.calls, "mounts")
	return f.mounts, f.err
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

func TestMountsError(t *testing.T) {
	player := &fakePlayer{err: errors.New("mpd down")}
	if rec := serve(t, player, http.MethodGet, "/player/mounts", nil); rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
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
