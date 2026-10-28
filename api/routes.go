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

type statusResponse struct {
	Playback playbackResponse `json:"playback"`
	Mounts   []mountResponse  `json:"mounts"`
}

// playbackResponse is the playback state, null standing for what is missing.
type playbackResponse struct {
	State    string   `json:"state"`
	Source   *string  `json:"source"`
	Mount    *string  `json:"mount"`
	Title    *string  `json:"title"`
	Artist   *string  `json:"artist"`
	Album    *string  `json:"album"`
	Name     *string  `json:"name"`
	URL      *string  `json:"url"`
	Position *int     `json:"position"`
	Length   int      `json:"length"`
	Elapsed  *float64 `json:"elapsed"`
	Duration *float64 `json:"duration"`
}

func (s *Server) registerPlayerRoutes(p Player) {
	s.mux.HandleFunc(
		"GET /player/status",
		withJSON(p.Status, newStatusResponse),
	)
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
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// writeError writes err as plain text, with the HTTP status telling what
// failed: 502 for MPD, 404 for an unknown mount, 409 for the disc drives.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, mpdplayer.ErrMPD):
		http.Error(w, err.Error(), http.StatusBadGateway)
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
			writeError(w, err)
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

func newStatusResponse(st mpdplayer.Status) statusResponse {
	pb := st.Playback
	res := playbackResponse{
		State:    pb.State,
		Source:   nonZero(pb.Source),
		Mount:    nonZero(pb.Mount),
		Title:    nonZero(pb.Title),
		Artist:   nonZero(pb.Artist),
		Album:    nonZero(pb.Album),
		Name:     nonZero(pb.Name),
		URL:      nonZero(pb.URL),
		Position: nonZero(pb.Position),
		Length:   pb.Length,
		Elapsed:  pb.Elapsed,
		Duration: pb.Duration,
	}
	return statusResponse{Playback: res, Mounts: newMountsResponse(st.Mounts)}
}

// nonZero points to v, nil when v is the zero value.
func nonZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
