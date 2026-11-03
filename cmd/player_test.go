package cmd

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
)

type fakeDevice struct {
	kind detect.DeviceKind
}

func (d fakeDevice) Path() string                  { return "/dev/fake" }
func (d fakeDevice) Kind() detect.DeviceKind       { return d.kind }
func (d fakeDevice) DetectEvent() detect.EventType { return detect.InvalidEvent }
func (d fakeDevice) Udev() *udev.Device            { return nil }

// recordingHandler records the callbacks it gets, as "kind:callback".
func recordingHandler(kind detect.DeviceKind, calls *[]string, err error) *BasicHandler {
	record := func(name string) func(context.Context, detect.Device) error {
		return func(_ context.Context, dev detect.Device) error {
			*calls = append(*calls, string(dev.Kind())+":"+name)
			return err
		}
	}
	return NewBasicHandler(kind, record("add"), record("remove"), record("present"))
}

func TestBasicHandler(t *testing.T) {
	h := NewBasicHandler(detect.DeviceUSB, nil, nil, nil)
	if !h.Handles(detect.DeviceUSB) || h.Handles(detect.DeviceDisc) {
		t.Error("Handles() should only accept its own kind")
	}
	dev := fakeDevice{detect.DeviceUSB}
	ctx := context.Background()
	if h.OnAdd(ctx, dev) != nil || h.OnRemove(ctx, dev) != nil || h.OnPresent(ctx, dev) != nil {
		t.Error("callbacks left nil should do nothing")
	}
}

func TestDispatch(t *testing.T) {
	tests := []struct {
		event detect.EventType
		kind  detect.DeviceKind
		want  []string
	}{
		{detect.DeviceAdded, detect.DeviceDisc, []string{"disc:add"}},
		{detect.DeviceRemoved, detect.DeviceDisc, []string{"disc:remove"}},
		{detect.DeviceAdded, detect.DeviceUSB, []string{"usb:add"}},
		{detect.DeviceRemoved, detect.DeviceUSB, []string{"usb:remove"}},
		{detect.DevicePresent, detect.DeviceUSB, []string{"usb:present"}},
		{detect.InvalidEvent, detect.DeviceUSB, nil},
	}
	for _, tt := range tests {
		t.Run(string(tt.event)+"_"+string(tt.kind), func(t *testing.T) {
			var calls []string
			p := &Player{
				ctx: context.Background(),
				handlers: []Handler{
					recordingHandler(detect.DeviceDisc, &calls, nil),
					recordingHandler(detect.DeviceUSB, &calls, nil),
				},
			}
			p.dispatch(detect.DeviceEvent{Type: tt.event, Device: fakeDevice{tt.kind}})
			if !reflect.DeepEqual(calls, tt.want) {
				t.Errorf("calls = %v, want %v", calls, tt.want)
			}
		})
	}
}

func TestDispatchHandlerError(t *testing.T) {
	var calls []string
	p := &Player{
		ctx:      context.Background(),
		handlers: []Handler{recordingHandler(detect.DeviceUSB, &calls, errors.New("boom"))},
	}
	// The error is logged, the dispatcher keeps going
	p.dispatch(detect.DeviceEvent{Type: detect.DeviceAdded, Device: fakeDevice{detect.DeviceUSB}})
	p.dispatch(detect.DeviceEvent{Type: detect.DeviceRemoved, Device: fakeDevice{detect.DeviceUSB}})
	if want := []string{"usb:add", "usb:remove"}; !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func runUntilDone(t *testing.T, p *Player, events <-chan detect.DeviceEvent) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		p.run(events)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not return")
	}
}

func TestRunDispatchesUntilEventsClosed(t *testing.T) {
	var calls []string
	p := &Player{
		ctx:      context.Background(),
		handlers: []Handler{recordingHandler(detect.DeviceUSB, &calls, nil)},
	}
	events := make(chan detect.DeviceEvent, 2)
	events <- detect.DeviceEvent{Type: detect.DevicePresent, Device: fakeDevice{detect.DeviceUSB}}
	events <- detect.DeviceEvent{Type: detect.DeviceRemoved, Device: fakeDevice{detect.DeviceUSB}}
	close(events)

	runUntilDone(t, p, events)
	if want := []string{"usb:present", "usb:remove"}; !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runUntilDone(t, &Player{ctx: ctx}, make(chan detect.DeviceEvent))
}

func TestCloseWithoutServices(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Player{ctx: ctx, cancel: cancel, wg: &sync.WaitGroup{}}
	p.Close()
	if ctx.Err() == nil {
		t.Error("Close() should cancel the context")
	}
}

func TestExecuteActionUnknown(t *testing.T) {
	if err := (&Player{}).ExecuteAction(io.Discard, Action{Name: "rewind"}); err == nil {
		t.Error("an unknown action should fail")
	}
}

func TestScheduler(t *testing.T) {
	if newScheduler(nil) != nil {
		t.Error("no schedule should give no scheduler")
	}
	var nilScheduler *scheduler
	nilScheduler.Close() // must not panic

	p := &Player{}
	p.StartScheduler() // must not panic without a scheduler
	p.StopScheduler()

	called := make(chan struct{}, 1)
	s := newScheduler([]*ScheduleUri{
		{schedule: "not a cron", uri: "bad", callback: func() {}},
		{schedule: "@every 1s", uri: "good", callback: func() {
			select {
			case called <- struct{}{}:
			default:
			}
		}},
	})
	if s == nil {
		t.Fatal("an invalid cron should not disable the valid ones")
	}
	defer s.Close()
	if n := len(s.c.Entries()); n != 1 {
		t.Errorf("%d cron entries, want 1", n)
	}
	s.c.Start()
	select {
	case <-called:
	case <-time.After(3 * time.Second):
		t.Error("the valid schedule never ran")
	}
}
