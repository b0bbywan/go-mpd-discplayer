package api

import (
	"errors"
	"net/http"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
)

func (s *Server) registerPlayerRoutes(p Player) {
	s.mux.HandleFunc(
		"POST /player/stop",
		withAction(p.Stop),
	)

	s.mux.HandleFunc(
		"POST /player/disc/play",
		withDisc(p.PlayDisc),
	)
	s.mux.HandleFunc(
		"POST /player/disc/stop",
		withAction(p.StopDisc),
	)
	s.mux.HandleFunc(
		"POST /player/disc/eject",
		withDisc(p.EjectDisc),
	)
}

// handleActionError writes the HTTP response of a player action.
func handleActionError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusAccepted)
	case errors.Is(err, detect.ErrNoDisc), errors.Is(err, detect.ErrSeveralDiscs):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// withDisc wraps a disc action on the drive holding the disc.
func withDisc(fn func(drive string) error) http.HandlerFunc {
	return withAction(func() error { return fn("") })
}

// withAction wraps a no-arg player action into an http.HandlerFunc.
func withAction(fn func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleActionError(w, fn())
	}
}
