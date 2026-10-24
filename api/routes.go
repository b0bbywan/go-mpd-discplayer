package api

import (
	"errors"
	"net/http"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

type mountResponse struct {
	Name    string `json:"name"`
	Storage string `json:"storage"`
}

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

	s.mux.HandleFunc(
		"GET /player/mounts",
		withJSON(p.Mounts, newMountsResponse),
	)
	s.mux.HandleFunc(
		"POST /player/mounts/{name}/play",
		withMount(p.PlayMount),
	)
	s.mux.HandleFunc(
		"POST /player/mounts/{name}/stop",
		withMount(p.StopMount),
	)
	s.mux.HandleFunc(
		"POST /player/mounts/{name}/eject",
		withMount(p.EjectMount),
	)

	s.mux.HandleFunc(
		"POST /player/stream/play",
		withBody(validateStream, func(w http.ResponseWriter, r *http.Request, req *streamRequest) {
			handleActionError(w, p.PlayURL(req.URL))
		}),
	)
}

// handleActionError writes the HTTP response of a player action.
func handleActionError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusAccepted)
	case errors.Is(err, mpdplayer.ErrUnknownMount):
		http.Error(w, err.Error(), http.StatusNotFound)
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

// withJSON wraps a player read into an http.HandlerFunc writing its result,
// turned into a response by toResponse, as JSON.
func withJSON[T, R any](fn func() (T, error), toResponse func(T) R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := fn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, toResponse(v))
	}
}

// newMountsResponse lists the mounts, as an empty list rather than null when
// there is none.
func newMountsResponse(mounts []mpdplayer.Mount) []mountResponse {
	res := make([]mountResponse, 0, len(mounts))
	for _, m := range mounts {
		res = append(res, mountResponse{Name: m.Name, Storage: m.Storage})
	}
	return res
}

// withMount wraps a player action on the mount named in the path.
func withMount(fn func(name string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleActionError(w, fn(r.PathValue("name")))
	}
}
