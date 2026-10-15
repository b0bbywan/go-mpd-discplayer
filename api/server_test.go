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
	"testing"
	"time"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

type fakePlayer struct {
	calls []string
	err   error
}

func (f *fakePlayer) PlayDisc(drive string) error {
	f.calls = append(f.calls, "play")
	return f.err
}

func (f *fakePlayer) Stop() error {
	f.calls = append(f.calls, "stop")
	return f.err
}

func (f *fakePlayer) EjectDisc(drive string) error {
	f.calls = append(f.calls, "eject")
	return f.err
}

func serve(t *testing.T, player Player, method, path string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
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
		{"/player/disc/eject", "eject"},
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
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "POST, OPTIONS" {
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
