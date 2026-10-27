// Package state holds what the player knows about its media: the disc in the
// drive and the USB storages mounted in MPD. Playback is not part of it, MPD
// keeps it.
package state

import (
	"sync"
	"sync/atomic"

	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

// DiscStatus tells whether the disc metadata is known yet.
type DiscStatus string

const (
	// Resolving is a disc just inserted, whose CUE sheet is being resolved.
	// Only its device is known.
	Resolving DiscStatus = "resolving"
	// Ready is a disc whose CUE sheet is resolved, or that has none.
	Ready DiscStatus = "ready"
)

// Disc is the audio disc in the drive.
type Disc struct {
	ID     string // FreeDB disc id, also the cache directory name, "" until resolved
	Device string
	Status DiscStatus
	Album  string
	Artist string
	Date   string
	Genre  string
	Tracks []string // track titles, in order
	Cover  string   // path of the cover in the cache, "" without one
}

// State is a snapshot of the player media. The values it reaches (the disc,
// the mounts) are never modified in place: writers replace them.
type State struct {
	Disc   *Disc // nil without an audio disc in the drive
	Mounts []mpdplayer.Mount
}

// Store holds the current State. Reads are lock-free; writes go one at a
// time. The zero value is an empty store.
type Store struct {
	mu  sync.Mutex
	ptr atomic.Pointer[State]
}

// Load returns the current state.
func (s *Store) Load() State {
	if st := s.ptr.Load(); st != nil {
		return *st
	}
	return State{}
}

// Update replaces the state with what fn returns from the current one.
// Updates are serialized, so fn always sees the result of the previous one.
func (s *Store) Update(fn func(State) State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := fn(s.Load())
	s.ptr.Store(&next)
}
