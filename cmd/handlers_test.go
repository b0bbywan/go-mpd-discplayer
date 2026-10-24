package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
)

var errFake = errors.New("fake error")

// fakeClient records the MPD operations, as "name arg". Operations listed in
// fail return errFake.
type fakeClient struct {
	calls []string
	fail  map[string]bool
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
func (c *fakeClient) UpdateDB(uri string) error           { return c.record("UpdateDB", uri) }
func (c *fakeClient) Disconnect()                         { _ = c.record("Disconnect", "") }

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
		return func(p *Player) error { return p.ExecuteAction(a) }
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
		{"ExecuteAction disc play", run(Action{Name: ActionDiscPlay, Device: "/dev/sr1"}), []string{"StartDiscPlayback /dev/sr1"}, false},
		{"ExecuteAction disc stop", run(Action{Name: ActionDiscStop}), []string{"StopIfPlaying cdda://"}, false},
		{"ExecuteAction disc eject", run(Action{Name: ActionDiscEject, Device: notADrive}), []string{"StopAndRemove cdda://"}, true},
		{"ExecuteAction stream play", run(Action{Name: ActionStreamPlay, URL: "http://radio/stream"}), []string{"StartPlayback http://radio/stream"}, false},
		{"ExecuteAction stop", run(Action{Name: ActionStop}), []string{"Stop"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{}
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
