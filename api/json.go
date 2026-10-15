package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
)

const maxRequestBodySize = 1 << 20 // 1 MB

type streamRequest struct {
	URL string `json:"url"`
}

// withBody parses and validates a JSON request body, then calls next.
func withBody[T any](
	validate func(*T) error,
	next func(w http.ResponseWriter, r *http.Request, req *T),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("[api] failed to close request body: %v", err)
			}
		}()

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

		var req T
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		if validate != nil {
			if err := validate(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		next(w, r, &req)
	}
}

// validateStream accepts http and https URLs only: the route is meant for
// webradios, not for files on the player.
func validateStream(req *streamRequest) error {
	u, err := url.Parse(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("url must be an http or https URL")
	}
	return nil
}
