package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

func TestStoreZeroValue(t *testing.T) {
	var s Store
	if st := s.Load(); st.Disc != nil || st.Mounts != nil {
		t.Errorf("Load() on an empty store = %+v, want the zero State", st)
	}
}

func TestStoreUpdate(t *testing.T) {
	var s Store
	disc := &Disc{Device: "/dev/sr0", Status: Resolving}
	s.Update(func(st State) State {
		st.Disc = disc
		return st
	})
	mounts := []mpdplayer.Mount{{Name: "MUSIC", Storage: "udisks://x"}}
	s.Update(func(st State) State {
		st.Mounts = mounts
		return st
	})

	want := State{Disc: disc, Mounts: mounts}
	if got := s.Load(); !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestStoreUpdatesAreSerialized(t *testing.T) {
	var s Store
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			s.Update(func(st State) State {
				st.Mounts = append(st.Mounts[:len(st.Mounts):len(st.Mounts)], mpdplayer.Mount{})
				return st
			})
		})
	}
	wg.Wait()
	if n := len(s.Load().Mounts); n != 100 {
		t.Errorf("%d mounts after 100 updates, want 100", n)
	}
}

func TestDiscFile(t *testing.T) {
	f := DiscFile(filepath.Join(t.TempDir(), "app", "disc.json"))

	if _, err := f.Load(); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load() of a missing file error = %v, want fs.ErrNotExist", err)
	}

	want := SavedDisc{ID: "9c0bf40c", Device: "/dev/sr0"}
	if err := f.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := f.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}

	if err := f.Remove(); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := f.Load(); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load() after Remove() error = %v, want fs.ErrNotExist", err)
	}
	if err := f.Remove(); err != nil {
		t.Errorf("Remove() of a missing file error = %v, want nil", err)
	}
}

func TestDiscFileCorrupted(t *testing.T) {
	f := DiscFile(filepath.Join(t.TempDir(), "disc.json"))
	if err := os.WriteFile(string(f), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Load(); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load() of a corrupted file error = %v, want a decode error", err)
	}
}

func TestRuntimeDiscFile(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	f, err := RuntimeDiscFile("mpd-discplayer")
	if err != nil {
		t.Fatalf("RuntimeDiscFile() error = %v", err)
	}
	if want := DiscFile("/run/user/1000/mpd-discplayer/disc.json"); f != want {
		t.Errorf("RuntimeDiscFile() = %q, want %q", f, want)
	}

	t.Setenv("XDG_RUNTIME_DIR", "")
	if _, err := RuntimeDiscFile("mpd-discplayer"); err == nil {
		t.Error("RuntimeDiscFile() without XDG_RUNTIME_DIR expected error, got nil")
	}
}
