package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-disc-cuer/cue"
	"github.com/b0bbywan/go-disc-cuer/types"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

var errFake = errors.New("fake error")

// fakeClient records the MPD operations, as "name arg". Operations listed in
// fail return errFake.
type fakeClient struct {
	calls    []string
	fail     map[string]bool
	mounts   []mpdplayer.Mount
	playback mpdplayer.Playback
	queued   mpdplayer.QueuedDisc
}

func (c *fakeClient) record(name, arg string) error {
	call := name
	if arg != "" {
		call += " " + arg
	}
	c.calls = append(c.calls, call)
	if c.fail[name] {
		return errFake
	}
	return nil
}

func (c *fakeClient) StartDiscPlayback(device, cuePath string) error {
	return c.record("StartDiscPlayback", strings.TrimSpace(device+" "+cuePath))
}
func (c *fakeClient) StartUSBPlayback(label string) error { return c.record("StartUSBPlayback", label) }
func (c *fakeClient) StartPlayback(uri string) error      { return c.record("StartPlayback", uri) }
func (c *fakeClient) Stop() error                         { return c.record("Stop", "") }
func (c *fakeClient) StopIfPlaying(prefixes ...string) error {
	return c.record("StopIfPlaying", strings.Join(prefixes, " "))
}
func (c *fakeClient) StopAndRemove(prefixes ...string) error {
	return c.record("StopAndRemove", strings.Join(prefixes, " "))
}
func (c *fakeClient) DiscPrefixes() []string     { return []string{"cdda://", ".disc-cuer/"} }
func (c *fakeClient) Unmount(label string) error { return c.record("Unmount", label) }
func (c *fakeClient) UpdateDB(uri string) error  { return c.record("UpdateDB", uri) }
func (c *fakeClient) Disconnect()                { _ = c.record("Disconnect", "") }

func (c *fakeClient) USBMounts() ([]mpdplayer.Mount, error) {
	return c.mounts, c.record("USBMounts", "")
}

func (c *fakeClient) QueuedDisc() (mpdplayer.QueuedDisc, error) {
	return c.queued, c.record("QueuedDisc", "")
}

func (c *fakeClient) Status() (mpdplayer.Status, error) {
	return mpdplayer.Status{Playback: c.playback, Mounts: c.mounts}, c.record("Status", "")
}

// fakeCuer resolves every disc to res, or fails with err, and reads info as
// the cached metadata, or fails with cachedErr. It records its calls.
type fakeCuer struct {
	res       *cue.Result
	err       error
	info      *types.DiscInfo
	cachedErr error
	onResolve func() // run when Resolve is called
	cacheDir  string // cache directory, one subdirectory per disc id
	calls     []string
}

func (c *fakeCuer) CuePath(id string) string {
	return filepath.Join(c.cacheDir, id, "playlist.cue")
}

func (c *fakeCuer) CoverPath(id string) string {
	return filepath.Join(c.cacheDir, id, "cover.jpg")
}

func (c *fakeCuer) Resolve(opts cue.Options) (*cue.Result, error) {
	c.calls = append(c.calls, "Resolve "+opts.Device)
	if c.onResolve != nil {
		c.onResolve()
	}
	return c.res, c.err
}

func (c *fakeCuer) Cached(id string) (*types.DiscInfo, error) {
	c.calls = append(c.calls, "Cached "+id)
	return c.info, c.cachedErr
}

type fakeMounter struct {
	relPath string
	err     error
	calls   []string
}

func (m *fakeMounter) Mount(*udev.Device) (string, error) {
	m.calls = append(m.calls, "Mount")
	return m.relPath, m.err
}

func (m *fakeMounter) Unmount(*udev.Device) (string, error) {
	m.calls = append(m.calls, "Unmount")
	return m.relPath, m.err
}

func newTestPlayer(client *fakeClient, mounter *fakeMounter) *Player {
	p := &Player{ctx: context.Background(), Client: client, Mounter: mounter}
	p.newDiscHandler()
	p.newUSBHandler()
	return p
}

// callHandler runs the callback of the handler of kind for event.
func callHandler(t *testing.T, p *Player, kind detect.DeviceKind, event detect.EventType) error {
	t.Helper()
	dev := fakeDevice{kind}
	for _, h := range p.handlers {
		if !h.Handles(kind) {
			continue
		}
		switch event {
		case detect.DeviceAdded:
			return h.OnAdd(p.ctx, dev)
		case detect.DeviceRemoved:
			return h.OnRemove(p.ctx, dev)
		case detect.DevicePresent:
			return h.OnPresent(p.ctx, dev)
		}
	}
	t.Fatalf("no %s handler", kind)
	return nil
}

func TestDeviceHandlers(t *testing.T) {
	tests := []struct {
		kind        detect.DeviceKind
		event       detect.EventType
		mountErr    error
		fail        string
		wantMounter []string
		wantClient  []string
		wantErr     bool
	}{
		// Disc: the speed cannot be set on the fake device, which is only logged
		{kind: detect.DeviceDisc, event: detect.DeviceAdded, wantClient: []string{"StartDiscPlayback /dev/fake"}},
		{kind: detect.DeviceDisc, event: detect.DeviceAdded, fail: "StartDiscPlayback", wantClient: []string{"StartDiscPlayback /dev/fake"}, wantErr: true},
		{kind: detect.DeviceDisc, event: detect.DeviceRemoved, wantClient: []string{"StopAndRemove cdda:// .disc-cuer/"}},
		// Without disc file nor cuer, the disc is kept once MPD tells it is not playing
		{kind: detect.DeviceDisc, event: detect.DevicePresent, wantClient: []string{"Status"}},

		{kind: detect.DeviceUSB, event: detect.DeviceAdded, wantMounter: []string{"Mount"}, wantClient: []string{"USBMounts", "StartUSBPlayback KEY"}},
		{kind: detect.DeviceUSB, event: detect.DeviceAdded, mountErr: errFake, wantMounter: []string{"Mount"}, wantErr: true},
		{kind: detect.DeviceUSB, event: detect.DeviceAdded, fail: "StartUSBPlayback", wantMounter: []string{"Mount"}, wantClient: []string{"USBMounts", "StartUSBPlayback KEY"}, wantErr: true},
		{kind: detect.DeviceUSB, event: detect.DeviceRemoved, wantMounter: []string{"Unmount"}, wantClient: []string{"USBMounts", "StopAndRemove KEY"}},
		{kind: detect.DeviceUSB, event: detect.DeviceRemoved, mountErr: errFake, wantMounter: []string{"Unmount"}, wantClient: []string{"USBMounts"}, wantErr: true},
		{kind: detect.DeviceUSB, event: detect.DevicePresent, wantMounter: []string{"Mount"}, wantClient: []string{"USBMounts", "UpdateDB KEY"}},
		{kind: detect.DeviceUSB, event: detect.DevicePresent, mountErr: errFake, wantMounter: []string{"Mount"}, wantErr: true},
		{kind: detect.DeviceUSB, event: detect.DevicePresent, fail: "UpdateDB", wantMounter: []string{"Mount"}, wantClient: []string{"USBMounts", "UpdateDB KEY"}, wantErr: true},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%s_%s_mountErr=%v_fail=%s", tt.kind, tt.event, tt.mountErr != nil, tt.fail)
		t.Run(name, func(t *testing.T) {
			client := &fakeClient{fail: map[string]bool{tt.fail: true}}
			mounter := &fakeMounter{relPath: "KEY", err: tt.mountErr}
			p := newTestPlayer(client, mounter)

			err := callHandler(t, p, tt.kind, tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(mounter.calls, tt.wantMounter) {
				t.Errorf("mounter calls = %v, want %v", mounter.calls, tt.wantMounter)
			}
			if !reflect.DeepEqual(client.calls, tt.wantClient) {
				t.Errorf("client calls = %v, want %v", client.calls, tt.wantClient)
			}
		})
	}
}

func TestRefreshMounts(t *testing.T) {
	usb := []mpdplayer.Mount{{Name: "MYUSB", Storage: "udisks://x"}}
	client := &fakeClient{mounts: usb}
	p := newTestPlayer(client, &fakeMounter{relPath: "MYUSB"})

	if err := callHandler(t, p, detect.DeviceUSB, detect.DeviceAdded); err != nil {
		t.Fatal(err)
	}
	if got := p.State().Mounts; !reflect.DeepEqual(got, usb) {
		t.Errorf("state mounts = %v, want %v", got, usb)
	}

	// MPD unreachable: the state keeps the previous list
	client.mounts = nil
	client.fail = map[string]bool{"USBMounts": true}
	p.refreshMounts()
	if got := p.State().Mounts; !reflect.DeepEqual(got, usb) {
		t.Errorf("state mounts after a failed refresh = %v, want %v", got, usb)
	}

	client.fail = nil
	p.refreshMounts()
	if got := p.State().Mounts; got != nil {
		t.Errorf("state mounts after unmount = %v, want none", got)
	}
}

func TestPlayerActions(t *testing.T) {
	notADrive := filepath.Join(t.TempDir(), "sr0")
	run := func(a Action) func(p *Player) error {
		return func(p *Player) error { return p.ExecuteAction(io.Discard, a) }
	}
	tests := []struct {
		name    string
		action  func(p *Player) error
		want    []string
		wantErr bool
	}{
		{"PlayDisc", func(p *Player) error { return p.PlayDisc("/dev/sr1") }, []string{"StartDiscPlayback /dev/sr1"}, false},
		{"PlayURL", func(p *Player) error { return p.PlayURL("http://radio/stream") }, []string{"StartPlayback http://radio/stream"}, false},
		{"Stop", func(p *Player) error { return p.Stop() }, []string{"Stop"}, false},
		{"StopDisc", func(p *Player) error { return p.StopDisc() }, []string{"StopIfPlaying cdda:// .disc-cuer/"}, false},
		// Playback is stopped before ejecting, even when the eject fails
		{"EjectDisc", func(p *Player) error { return p.EjectDisc(notADrive) }, []string{"StopAndRemove cdda:// .disc-cuer/"}, true},
		{"PlayMount", func(p *Player) error { return p.PlayMount("MYUSB") }, []string{"USBMounts", "StartUSBPlayback MYUSB"}, false},
		{"StopMount", func(p *Player) error { return p.StopMount("MYUSB") }, []string{"USBMounts", "StopIfPlaying MYUSB/"}, false},
		{"EjectMount", func(p *Player) error { return p.EjectMount("MYUSB") }, []string{"USBMounts", "StopAndRemove MYUSB/", "Unmount MYUSB", "USBMounts"}, false},
		// An unknown mount is rejected before touching the player
		{"PlayMount unknown", func(p *Player) error { return p.PlayMount("NAS") }, []string{"USBMounts"}, true},
		{"EjectMount unknown", func(p *Player) error { return p.EjectMount("MYUSB2") }, []string{"USBMounts"}, true},
		{"ExecuteAction disc play", run(Action{Name: ActionDiscPlay, Device: "/dev/sr1"}), []string{"StartDiscPlayback /dev/sr1"}, false},
		{"ExecuteAction disc stop", run(Action{Name: ActionDiscStop}), []string{"StopIfPlaying cdda:// .disc-cuer/"}, false},
		{"ExecuteAction disc eject", run(Action{Name: ActionDiscEject, Device: notADrive}), []string{"StopAndRemove cdda:// .disc-cuer/"}, true},
		{"ExecuteAction mount list", run(Action{Name: ActionMountList}), []string{"USBMounts"}, false},
		{"ExecuteAction mount play", run(Action{Name: ActionMountPlay, Mount: "MYUSB"}), []string{"USBMounts", "StartUSBPlayback MYUSB"}, false},
		{"ExecuteAction mount stop", run(Action{Name: ActionMountStop, Mount: "MYUSB"}), []string{"USBMounts", "StopIfPlaying MYUSB/"}, false},
		{"ExecuteAction mount eject", run(Action{Name: ActionMountEject, Mount: "MYUSB"}), []string{"USBMounts", "StopAndRemove MYUSB/", "Unmount MYUSB", "USBMounts"}, false},
		{"ExecuteAction stream play", run(Action{Name: ActionStreamPlay, URL: "http://radio/stream"}), []string{"StartPlayback http://radio/stream"}, false},
		{"ExecuteAction stop", run(Action{Name: ActionStop}), []string{"Stop"}, false},
		{"ExecuteAction status", run(Action{Name: ActionStatus}), []string{"Status"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{mounts: []mpdplayer.Mount{{Name: "MYUSB", Storage: "udisks://x"}}}
			err := tt.action(newTestPlayer(client, &fakeMounter{}))
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(client.calls, tt.want) {
				t.Errorf("client calls = %v, want %v", client.calls, tt.want)
			}
		})
	}
}

func TestPlayDiscResolvesCue(t *testing.T) {
	tests := []struct {
		name string
		cuer cueResolver
		want []string
	}{
		{"resolved", &fakeCuer{res: &cue.Result{CuePath: "/cache/x/playlist.cue"}}, []string{"StartDiscPlayback /dev/sr1 /cache/x/playlist.cue"}},
		// Without a CUE sheet the tracks are played without metadata
		{"resolve error", &fakeCuer{err: errFake}, []string{"StartDiscPlayback /dev/sr1"}},
		{"no cuer", nil, []string{"StartDiscPlayback /dev/sr1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{}
			p := newTestPlayer(client, &fakeMounter{})
			p.Cuer = tt.cuer
			if err := p.PlayDisc("/dev/sr1"); err != nil {
				t.Fatalf("PlayDisc() error = %v", err)
			}
			if !reflect.DeepEqual(client.calls, tt.want) {
				t.Errorf("client calls = %v, want %v", client.calls, tt.want)
			}
		})
	}
}

func TestUnknownMountError(t *testing.T) {
	p := newTestPlayer(&fakeClient{}, &fakeMounter{})
	if err := p.StopMount("MYUSB"); !errors.Is(err, mpdplayer.ErrUnknownMount) {
		t.Errorf("StopMount() = %v, want ErrUnknownMount", err)
	}
}

func TestPrintMounts(t *testing.T) {
	client := &fakeClient{mounts: []mpdplayer.Mount{
		{Name: "MYUSB", Storage: "udisks://a"},
		{Name: "OTHER", Storage: "udisks://b"},
	}}
	var out strings.Builder
	if err := newTestPlayer(client, &fakeMounter{}).ExecuteAction(&out, Action{Name: ActionMountList}); err != nil {
		t.Fatal(err)
	}
	if want := "MYUSB\tudisks://a\nOTHER\tudisks://b\n"; out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

func ptr(f float64) *float64 { return &f }

func TestPrintStatus(t *testing.T) {
	usb := []mpdplayer.Mount{{Name: "MUSIC", Storage: "udisks://a"}}
	tests := []struct {
		name     string
		playback mpdplayer.Playback
		mounts   []mpdplayer.Mount
		want     string
	}{
		{"disc", mpdplayer.Playback{State: "play", Source: mpdplayer.SourceDisc,
			Title: "Bruxelles", Artist: "Benabar", Album: "Reprise des negociations",
			Position: 4, Length: 12, Elapsed: ptr(102.3), Duration: ptr(224)}, usb,
			"state: play\nsource: disc\ntitle: Bruxelles\nartist: Benabar\nalbum: Reprise des negociations\n" +
				"position: 4\nlength: 12\nelapsed: 102.3\nduration: 224\nmounts: MUSIC\tudisks://a\n"},
		{"mount", mpdplayer.Playback{State: "pause", Source: mpdplayer.SourceMount, Mount: "MUSIC",
			Position: 1, Length: 1, Elapsed: ptr(0)}, usb,
			"state: pause\nsource: mount\nmount: MUSIC\nposition: 1\nlength: 1\nelapsed: 0\nmounts: MUSIC\tudisks://a\n"},
		{"stream", mpdplayer.Playback{State: "play", Source: mpdplayer.SourceStream, URL: "https://radio/live",
			Title: "Live", Name: "Radio", Position: 1, Length: 1, Elapsed: ptr(12)}, nil,
			"state: play\nsource: stream\ntitle: Live\nname: Radio\nurl: https://radio/live\nposition: 1\nlength: 1\nelapsed: 12\n"},
		{"stopped", mpdplayer.Playback{State: "stop"}, nil, "state: stop\nlength: 0\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{playback: tt.playback, mounts: tt.mounts}
			var out strings.Builder
			if err := newTestPlayer(client, &fakeMounter{}).ExecuteAction(&out, Action{Name: ActionStatus}); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Errorf("output =\n%s\nwant\n%s", out.String(), tt.want)
			}
		})
	}
}
