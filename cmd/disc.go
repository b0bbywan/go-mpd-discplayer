package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"

	"github.com/b0bbywan/go-disc-cuer/cue"
	"github.com/b0bbywan/go-disc-cuer/types"
	"github.com/b0bbywan/go-disc-cuer/utils"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/state"
)

// readTrackCount reads the number of tracks from the TOC of the disc in drive.
var readTrackCount = utils.GetTrackCount

// cueCache resolves CUE sheets with go-disc-cuer and reads back the cached
// ones from its cache location.
type cueCache struct {
	*cue.Generator
	location string
}

func (c cueCache) Cached(id string) (*types.DiscInfo, error) {
	return cue.Read(c.CuePath(id))
}

func (c cueCache) CuePath(id string) string {
	return utils.CachePlaylistPath(c.location, id)
}

// insertDisc plays the disc just inserted in drive. The disc stays resolving
// in the state until its CUE sheet is resolved and loaded.
func (p *Player) insertDisc(drive string) error {
	p.setDisc(&state.Disc{Device: drive, Status: state.Resolving})
	p.setDiscSpeed(drive)
	res := p.resolveDisc(drive)
	// Built before playback starts: an unknown disc reads its TOC
	disc := newDisc(drive, res)
	err := p.startDisc(drive, res)
	p.keepDisc(disc)
	return err
}

// presentDisc keeps the disc found in drive at startup, without starting
// playback. The disc file spares reading the TOC, which would interrupt
// playback; without it, a disc playing is read from the MPD queue, and the
// TOC only when the disc is not playing.
func (p *Player) presentDisc(drive string) error {
	if disc := p.restoreDisc(drive); disc != nil {
		p.setDisc(disc)
		return nil
	}
	playing, err := p.discPlaying()
	if err != nil {
		p.keepDisc(&state.Disc{Device: drive, Status: state.Ready})
		return err
	}
	if playing {
		p.keepDisc(p.queuedDisc(drive))
		return nil
	}
	p.keepDisc(newDisc(drive, p.resolveDisc(drive)))
	return nil
}

// queuedDisc is the disc in drive as MPD has it in the queue: from the cached
// sheet MPD loaded it from, else from the tags of its tracks, without id.
func (p *Player) queuedDisc(drive string) *state.Disc {
	disc := &state.Disc{Device: drive, Status: state.Ready}
	qd, err := p.Client.QueuedDisc()
	if err != nil {
		log.Printf("warning: failed to read the disc in the queue: %v", err)
		return disc
	}
	if loaded := p.loadedDisc(drive, qd); loaded != nil {
		return loaded
	}
	disc.Album = qd.Album
	disc.Artist = qd.Artist
	disc.Date = qd.Date
	disc.Genre = qd.Genre
	disc.Tracks = qd.Tracks
	return disc
}

// loadedDisc returns the disc of the cached sheet the queue was loaded from,
// nil when it was not loaded from one, or no longer holds its tracks.
func (p *Player) loadedDisc(drive string, qd mpdplayer.QueuedDisc) *state.Disc {
	if p.Cuer == nil || qd.CuePath == "" {
		return nil
	}
	id := filepath.Base(filepath.Dir(qd.CuePath))
	if qd.CuePath != p.Cuer.CuePath(id) {
		return nil
	}
	disc, err := p.cachedDisc(id, drive)
	if err != nil {
		log.Printf("warning: %v", err)
		return nil
	}
	if len(disc.Tracks) != len(qd.Tracks) {
		return nil
	}
	return disc
}

// removeDisc forgets the disc removed from the drive.
func (p *Player) removeDisc() {
	p.setDisc(nil)
	if p.discFile == "" {
		return
	}
	if err := p.discFile.Remove(); err != nil {
		log.Printf("warning: %v", err)
	}
}

// startDisc replaces the queue with the disc in drive and starts playback,
// from the CUE sheet of res when there is one.
func (p *Player) startDisc(drive string, res *cue.Result) error {
	cuePath := ""
	if res != nil {
		cuePath = res.CuePath
	}
	if err := p.Client.StartDiscPlayback(drive, cuePath); err != nil {
		return fmt.Errorf("error adding tracks: %w", err)
	}
	return nil
}

func (p *Player) setDiscSpeed(drive string) {
	if err := hwcontrol.SetDiscSpeed(drive, p.discSpeed); err != nil {
		log.Printf("warning: failed to set disc speed on %s: %v", drive, err)
	}
}

// resolveDisc resolves the CUE sheet of the disc in drive, reading its TOC.
// It returns nil when the disc cannot be resolved, the disc then playing
// without metadata.
func (p *Player) resolveDisc(drive string) *cue.Result {
	if p.Cuer == nil {
		return nil
	}
	res, err := p.Cuer.Resolve(cue.Options{Device: drive})
	if err != nil {
		log.Printf("warning: failed to resolve the CUE sheet of %s: %v", drive, err)
		return nil
	}
	return res
}

// restoreDisc returns the disc of the disc file when it is in drive, with the
// metadata of its cached sheet, nil otherwise.
func (p *Player) restoreDisc(drive string) *state.Disc {
	if p.discFile == "" || p.Cuer == nil {
		return nil
	}
	saved, err := p.discFile.Load()
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Printf("warning: %v", err)
		}
		return nil
	}
	if saved.Device != drive {
		return nil
	}
	disc, err := p.cachedDisc(saved.ID, drive)
	if err != nil {
		log.Printf("warning: %v", err)
		return nil
	}
	return disc
}

// cachedDisc is the ready disc id in drive, with the metadata of its cached
// sheet.
func (p *Player) cachedDisc(id, drive string) (*state.Disc, error) {
	info, err := p.Cuer.Cached(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read the cached sheet of %s: %w", id, err)
	}
	disc := &state.Disc{ID: id, Device: drive, Status: state.Ready}
	setDiscInfo(disc, info)
	return disc, nil
}

// discPlaying reports whether MPD is playing or paused on a disc track.
func (p *Player) discPlaying() (bool, error) {
	st, err := p.Client.Status()
	if err != nil {
		return false, fmt.Errorf("error reading status: %w", err)
	}
	pb := st.Playback
	return pb.Source == mpdplayer.SourceDisc && (pb.State == "play" || pb.State == "pause"), nil
}

// keepDisc sets disc in the state and in the disc file. A disc without id,
// unknown to the metadata sources, is not kept across restarts.
func (p *Player) keepDisc(disc *state.Disc) {
	p.setDisc(disc)
	if p.discFile == "" {
		return
	}
	var err error
	if disc.ID == "" {
		err = p.discFile.Remove()
	} else {
		err = p.discFile.Save(state.SavedDisc{ID: disc.ID, Device: disc.Device})
	}
	if err != nil {
		log.Printf("warning: %v", err)
	}
}

func (p *Player) setDisc(disc *state.Disc) {
	p.store.Update(func(st state.State) state.State {
		st.Disc = disc
		return st
	})
}

// newDisc is the ready disc in drive, resolved as res. An unresolved disc has
// only its number of tracks, read from its TOC, without titles.
func newDisc(drive string, res *cue.Result) *state.Disc {
	disc := &state.Disc{Device: drive, Status: state.Ready}
	if res == nil {
		n, err := readTrackCount(drive)
		if err != nil {
			log.Printf("warning: failed to read the tracks of %s: %v", drive, err)
			return disc
		}
		disc.Tracks = make([]string, n)
		return disc
	}
	disc.ID = res.ID
	setDiscInfo(disc, res.Info)
	return disc
}

func setDiscInfo(disc *state.Disc, info *types.DiscInfo) {
	if info == nil {
		return
	}
	disc.Album = info.Title
	disc.Artist = info.Artist
	disc.Date = info.ReleaseDate
	disc.Genre = info.Genre
	disc.Tracks = info.Tracks
	disc.Cover = info.CoverArtPath
}
