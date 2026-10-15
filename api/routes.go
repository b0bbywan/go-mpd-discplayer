package api

import (
	"errors"
	"net/http"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
)

func (s *Server) registerPlayerRoutes(p Player) {
	s.mux.HandleFunc(
		"POST /player/play",
		withAction(p.Play),
	)
	s.mux.HandleFunc(
		"POST /player/stream",
		withBody(validateStream, func(w http.ResponseWriter, r *http.Request, req *streamRequest) {
			handleActionError(w, p.PlayURL(req.URL))
		}),
	)
	s.mux.HandleFunc(
		"POST /player/stop",
		withAction(p.Stop),
	)
	s.mux.HandleFunc(
		"POST /player/eject",
		withAction(p.Eject),
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

// withAction wraps a no-arg player action into an http.HandlerFunc.
func withAction(fn func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleActionError(w, fn())
	}
}
