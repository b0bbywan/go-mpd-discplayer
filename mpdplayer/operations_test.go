package mpdplayer

import (
	"reflect"
	"strings"
	"testing"
)

func assertCommands(t *testing.T, f *fakeMPD, want ...string) {
	t.Helper()
	if got := f.sent(); !reflect.DeepEqual(got, want) {
		t.Errorf("commands = %q, want %q", got, want)
	}
}

func assertQueue(t *testing.T, f *fakeMPD, want ...string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.queue) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(f.queue, want) {
		t.Errorf("queue = %q, want %q", f.queue, want)
	}
}

func TestStartPlayback(t *testing.T) {
	f := newFakeMPD()
	f.queue = []string{"old/track.flac"}
	client := f.start(t)

	if err := client.StartPlayback("http://radio.example/stream"); err != nil {
		t.Fatal(err)
	}
	assertCommands(t, f, "clear", "add http://radio.example/stream", "play")
	assertQueue(t, f, "http://radio.example/stream")
}

func TestStartUSBPlayback(t *testing.T) {
	f := newFakeMPD()
	client := f.start(t)

	if err := client.StartUSBPlayback("MYUSB"); err != nil {
		t.Fatal(err)
	}
	assertCommands(t, f, "clear", "update MYUSB", "status", "add MYUSB", "play")
}

func TestStopAndRemove(t *testing.T) {
	queue := []string{"cdda:///1", "usb/a.flac", "cdda:///2", "cdda:///3", "usb/b.flac", "cdda:///4"}
	tests := []struct {
		name      string
		prefix    string
		current   int
		wantStop  bool
		wantQueue []string
	}{
		{"prefix playing", CDDAPathPrefix, 2, true, []string{"usb/a.flac", "usb/b.flac"}},
		{"other playing", CDDAPathPrefix, 1, false, []string{"usb/a.flac", "usb/b.flac"}},
		{"nothing playing", CDDAPathPrefix, -1, true, []string{"usb/a.flac", "usb/b.flac"}},
		{"usb prefix", "usb", 0, false, []string{"cdda:///1", "cdda:///2", "cdda:///3", "cdda:///4"}},
		{"prefix absent", "nas", 0, false, queue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeMPD()
			f.queue = append([]string(nil), queue...)
			f.current = tt.current
			client := f.start(t)

			if err := client.StopAndRemove(tt.prefix); err != nil {
				t.Fatal(err)
			}
			stopped := false
			for _, c := range f.sent() {
				stopped = stopped || c == "stop"
			}
			if stopped != tt.wantStop {
				t.Errorf("stop sent = %v, want %v", stopped, tt.wantStop)
			}
			assertQueue(t, f, tt.wantQueue...)
		})
	}
}

func TestMount(t *testing.T) {
	const neighbor = "udisks://Kingston-1234-by-uuid-AAAA-BBBB"
	tests := []struct {
		name       string
		mounts     []fakeMount
		neighbors  []string
		ids        []string
		wantErr    string
		wantMounts []fakeMount
	}{
		{
			name:       "neighbor found by uuid",
			neighbors:  []string{"udisks://other", neighbor},
			ids:        []string{"AAAA-BBBB", "Kingston_1234"},
			wantMounts: []fakeMount{{"MYUSB", neighbor}},
		},
		{
			name:       "neighbor found by serial",
			neighbors:  []string{neighbor},
			ids:        []string{"CCCC", "Kingston-1234"},
			wantMounts: []fakeMount{{"MYUSB", neighbor}},
		},
		{
			name:       "already mounted",
			mounts:     []fakeMount{{"MYUSB", neighbor}},
			ids:        []string{"AAAA-BBBB"},
			wantMounts: []fakeMount{{"MYUSB", neighbor}},
		},
		{
			name:      "no neighbor",
			neighbors: []string{"udisks://other"},
			ids:       []string{"AAAA-BBBB"},
			wantErr:   "no neighbor found",
		},
		{
			name:       "label used by another storage",
			mounts:     []fakeMount{{"MYUSB", "udisks://other"}},
			neighbors:  []string{neighbor},
			ids:        []string{"AAAA-BBBB"},
			wantErr:    "Mount point busy",
			wantMounts: []fakeMount{{"MYUSB", "udisks://other"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeMPD()
			f.mounts = tt.mounts
			f.neighbors = tt.neighbors
			client := f.start(t)

			err := client.Mount(tt.ids, "MYUSB")
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
			if !reflect.DeepEqual(f.mounts, tt.wantMounts) {
				t.Errorf("mounts = %v, want %v", f.mounts, tt.wantMounts)
			}
		})
	}
}

func TestMountAlreadyMountedSkipsNeighbors(t *testing.T) {
	f := newFakeMPD()
	f.mounts = []fakeMount{{"MYUSB", "udisks://by-uuid-AAAA"}}
	client := f.start(t)

	if err := client.Mount([]string{"AAAA"}, "MYUSB"); err != nil {
		t.Fatal(err)
	}
	// At boot the udisks neighbor may not be listed yet
	assertCommands(t, f, "listmounts")
}

func TestUnmount(t *testing.T) {
	f := newFakeMPD()
	f.mounts = []fakeMount{{"MYUSB", "udisks://x"}}
	client := f.start(t)

	if err := client.Unmount("MYUSB"); err != nil {
		t.Fatal(err)
	}
	if len(f.mounts) != 0 {
		t.Errorf("mounts = %v, want none", f.mounts)
	}
	if err := client.Unmount("MYUSB"); err == nil {
		t.Error("unmounting an unknown label should fail")
	}
}

func TestClearStaleMounts(t *testing.T) {
	f := newFakeMPD()
	f.mounts = []fakeMount{
		{"", "/var/lib/mpd/music"},
		{"PLUGGED", "udisks://Kingston-by-uuid-AAAA"},
		{"GONE", "udisks://Sandisk-by-uuid-BBBB"},
		{"NAS", "nfs://nas/music"},
	}
	client := f.start(t)

	if err := client.ClearStaleMounts([]string{"AAAA", "Samsung_SSD"}); err != nil {
		t.Fatal(err)
	}
	want := []fakeMount{
		{"", "/var/lib/mpd/music"},
		{"PLUGGED", "udisks://Kingston-by-uuid-AAAA"},
		{"NAS", "nfs://nas/music"},
	}
	if !reflect.DeepEqual(f.mounts, want) {
		t.Errorf("mounts = %v, want %v", f.mounts, want)
	}
}

func TestClearStaleMountsNothingPlugged(t *testing.T) {
	f := newFakeMPD()
	f.mounts = []fakeMount{{"GONE", "udisks://by-uuid-BBBB"}}
	client := f.start(t)

	if err := client.ClearStaleMounts(nil); err != nil {
		t.Fatal(err)
	}
	if len(f.mounts) != 0 {
		t.Errorf("mounts = %v, want none", f.mounts)
	}
}

func TestUpdateDB(t *testing.T) {
	f := newFakeMPD()
	client := f.start(t)

	if err := client.UpdateDB("MYUSB"); err != nil {
		t.Fatal(err)
	}
	assertCommands(t, f, "update MYUSB")
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s    string
		subs []string
		want bool
	}{
		{"udisks://by-uuid-AAAA", []string{"BBBB", "AAAA"}, true},
		{"udisks://by-uuid-AAAA", []string{"BBBB"}, false},
		{"udisks://by-uuid-AAAA", nil, false},
		{"udisks://by-uuid-AAAA", []string{""}, false},
	}
	for _, tt := range tests {
		if got := containsAny(tt.s, tt.subs); got != tt.want {
			t.Errorf("containsAny(%q, %q) = %v, want %v", tt.s, tt.subs, got, tt.want)
		}
	}
}

func TestStop(t *testing.T) {
	f := newFakeMPD()
	f.queue = []string{"cdda:///1", "usb/a.flac"}
	f.current = 1
	client := f.start(t)

	if err := client.Stop(); err != nil {
		t.Fatal(err)
	}
	assertCommands(t, f, "stop")
	assertQueue(t, f, "cdda:///1", "usb/a.flac")
}
