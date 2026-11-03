package api

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/state"
)

// Player is the disc player driven by the API.
type Player interface {
	PlayDisc(drive string) error  // play the disc in drive, "" for the drive holding an audio disc
	StopDisc() error              // stop playback when the disc is playing
	EjectDisc(drive string) error // eject the disc in drive, "" for the drive holding one

	PlayMount(name string) error  // play a USB storage
	StopMount(name string) error  // stop playback when the USB storage is playing
	EjectMount(name string) error // unmount a USB storage from MPD

	PlayURL(url string) error // play a stream
	Stop() error              // stop playback, whatever is playing

	Status() (mpdplayer.Status, error) // playback state and USB storages mounted in MPD
	State() state.State                // disc in the drive and USB storages mounted in MPD

	DiscCover(id string) (string, error) // path of the cached cover of a disc, fs.ErrNotExist without one
}

type Config struct {
	Bind   string // TCP listen address
	Port   int    // TCP listen port, 0 → no TCP listener
	Socket string // Unix socket for a reverse proxy, "" → none
}

type Server struct {
	mux    *http.ServeMux
	config Config
}

func NewServer(cfg Config, player Player) *Server {
	server := &Server{
		mux:    http.NewServeMux(),
		config: cfg,
	}
	server.registerPlayerRoutes(player)
	return server
}

func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

func (s *Server) Run(ctx context.Context) error {
	listeners, err := s.listen()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	// Shutdown closes every listener, the Unix socket file included
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[api] server shutdown error: %v", err)
		}
	}()

	errCh := make(chan error, len(listeners))
	var wg sync.WaitGroup
	for _, ln := range listeners {
		wg.Add(1)
		go func(ln net.Listener) {
			defer wg.Done()
			log.Printf("[api] http server running on %s", ln.Addr())
			if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("server %s: %w", ln.Addr(), err)
			}
		}(ln)
	}

	wg.Wait()
	close(errCh)
	return <-errCh
}

func (s *Server) listen() ([]net.Listener, error) {
	var listeners []net.Listener
	if s.config.Port > 0 {
		ln, err := net.Listen("tcp", net.JoinHostPort(s.config.Bind, strconv.Itoa(s.config.Port)))
		if err != nil {
			return nil, fmt.Errorf("failed to listen on %s:%d: %w", s.config.Bind, s.config.Port, err)
		}
		listeners = append(listeners, ln)
	}
	if s.config.Socket != "" {
		ln, err := listenUnix(s.config.Socket)
		if err != nil {
			closeAll(listeners)
			return nil, fmt.Errorf("failed to listen on %s: %w", s.config.Socket, err)
		}
		listeners = append(listeners, ln)
	}
	if len(listeners) == 0 {
		return nil, errors.New("no listener configured: set API.Port or API.Socket")
	}
	return listeners, nil
}

// listenUnix binds path, replacing a socket a crashed run left behind; any
// other file there is not ours to remove.
func listenUnix(path string) (net.Listener, error) {
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&fs.ModeSocket != 0 {
		_ = os.Remove(path)
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil { // the reverse proxy runs as us
		_ = ln.Close()
		return nil, err
	}
	return ln, nil
}

func closeAll(listeners []net.Listener) {
	for _, ln := range listeners {
		_ = ln.Close()
	}
}

// corsMiddleware allows any origin: the API is reached from browser apps
// served elsewhere, directly or through a reverse proxy.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
