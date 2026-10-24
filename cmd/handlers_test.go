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

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

var errFake = errors.New("fake error")

// fakeClient records the MPD operations, as "name arg". Operations listed in
// fail return errFake.
type fakeClient struct {
	calls  []string
	fail   map[string]bool
	mounts []mpdplayer.Mount
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

func (c *fakeClient) StartDiscPlayback(device string) error {
	return c.record("StartDiscPlayback", device)
}
func (c *fakeClient) StartUSBPlayback(label string) error { return c.record("StartUSBPlayback", label) }
func (c *fakeClient) StartPlayback(uri string) error      { return c.record("StartPlayback", uri) }
func (c *fakeClient) Stop() error                         { return c.record("Stop", "") }
func (c *fakeClient) StopIfPlaying(prefix string) error   { return c.record("StopIfPlaying", prefix) }
func (c *fakeClient) StopAndRemove(prefix string) error   { return c.record("StopAndRemove", prefix) }
func (c *fakeClient) Unmount(label string) error          { return c.record("Unmount", label) }
func (c *fakeClient) UpdateDB(uri string) error           { return c.record("UpdateDB", uri) }
func (c *fakeClient) Disconnect()                         { _ = c.record("Disconnect", "") }

func (c *fakeClient) USBMounts() ([]mpdplayer.Mount, error) {
	return c.mounts, c.record("USBMounts", "")
}

func (c *fakeClient) Status() (mpdplayer.Status, error) {
	return mpdplayer.Status{Mounts: c.mounts}, c.record("Status", "")
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
		{kind: detect.DeviceDisc, event: detect.DeviceRemoved, wantClient: []string{"StopAndRemove cdda://"}},
		{kind: detect.DeviceDisc, event: detect.DevicePresent},

		{kind: detect.DeviceUSB, event: detect.DeviceAdded, wantMounter: []string{"Mount"}, wantClient: []string{"StartUSBPlayback KEY"}},
		{kind: detect.DeviceUSB, event: detect.DeviceAdded, mountErr: errFake, wantMounter: []string{"Mount"}, wantErr: true},
		{kind: detect.DeviceUSB, event: detect.DeviceAdded, fail: "StartUSBPlayback", wantMounter: []string{"Mount"}, wantClient: []string{"StartUSBPlayback KEY"}, wantErr: true},
		{kind: detect.DeviceUSB, event: detect.DeviceRemoved, wantMounter: []string{"Unmount"}, wantClient: []string{"StopAndRemove KEY"}},
		{kind: detect.DeviceUSB, event: detect.DeviceRemoved, mountErr: errFake, wantMounter: []string{"Unmount"}, wantErr: true},
		{kind: detect.DeviceUSB, event: detect.DevicePresent, wantMounter: []string{"Mount"}, wantClient: []string{"UpdateDB KEY"}},
		{kind: detect.DeviceUSB, event: detect.DevicePresent, mountErr: errFake, wantMounter: []string{"Mount"}, wantErr: true},
		{kind: detect.DeviceUSB, event: detect.DevicePresent, fail: "UpdateDB", wantMounter: []string{"Mount"}, wantClient: []string{"UpdateDB KEY"}, wantErr: true},
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
		{"StopDisc", func(p *Player) error { return p.StopDisc() }, []string{"StopIfPlaying cdda://"}, false},
		// Playback is stopped before ejecting, even when the eject fails
		{"EjectDisc", func(p *Player) error { return p.EjectDisc(notADrive) }, []string{"StopAndRemove cdda://"}, true},
		{"PlayMount", func(p *Player) error { return p.PlayMount("MYUSB") }, []string{"USBMounts", "StartUSBPlayback MYUSB"}, false},
		{"StopMount", func(p *Player) error { return p.StopMount("MYUSB") }, []string{"USBMounts", "StopIfPlaying MYUSB/"}, false},
		{"EjectMount", func(p *Player) error { return p.EjectMount("MYUSB") }, []string{"USBMounts", "StopAndRemove MYUSB/", "Unmount MYUSB"}, false},
		// An unknown mount is rejected before touching the player
		{"PlayMount unknown", func(p *Player) error { return p.PlayMount("NAS") }, []string{"USBMounts"}, true},
		{"EjectMount unknown", func(p *Player) error { return p.EjectMount("MYUSB2") }, []string{"USBMounts"}, true},
		{"ExecuteAction disc play", run(Action{Name: ActionDiscPlay, Device: "/dev/sr1"}), []string{"StartDiscPlayback /dev/sr1"}, false},
		{"ExecuteAction disc stop", run(Action{Name: ActionDiscStop}), []string{"StopIfPlaying cdda://"}, false},
		{"ExecuteAction disc eject", run(Action{Name: ActionDiscEject, Device: notADrive}), []string{"StopAndRemove cdda://"}, true},
		{"ExecuteAction mount list", run(Action{Name: ActionMountList}), []string{"USBMounts"}, false},
		{"ExecuteAction mount play", run(Action{Name: ActionMountPlay, Mount: "MYUSB"}), []string{"USBMounts", "StartUSBPlayback MYUSB"}, false},
		{"ExecuteAction mount stop", run(Action{Name: ActionMountStop, Mount: "MYUSB"}), []string{"USBMounts", "StopIfPlaying MYUSB/"}, false},
		{"ExecuteAction mount eject", run(Action{Name: ActionMountEject, Mount: "MYUSB"}), []string{"USBMounts", "StopAndRemove MYUSB/", "Unmount MYUSB"}, false},
		{"ExecuteAction stream play", run(Action{Name: ActionStreamPlay, URL: "http://radio/stream"}), []string{"StartPlayback http://radio/stream"}, false},
		{"ExecuteAction stop", run(Action{Name: ActionStop}), []string{"Stop"}, false},
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
