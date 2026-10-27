package mpdplayer

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/fhs/gompd/v2/mpd"
)

func TestExecuteMarksMPDErrors(t *testing.T) {
	f := newFakeMPD()
	client := f.start(t)

	// MPD answers ACK to a command it does not know
	err := client.execute(func(c *mpd.Client) error { return c.Command("nosuchcommand").OK() })
	if !errors.Is(err, ErrMPD) {
		t.Errorf("command failed by MPD: error = %v, want ErrMPD", err)
	}

	// An error of ours, reading the drive for instance, is not MPD's
	err = client.execute(func(*mpd.Client) error { return errors.New("no drive") })
	if err == nil || errors.Is(err, ErrMPD) {
		t.Errorf("error outside MPD: error = %v, want an error other than ErrMPD", err)
	}
}

func TestExecuteMarksUnreachableMPD(t *testing.T) {
	// A port nobody listens on
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	conn, err := NewMPDConnection("tcp", addr, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	client := NewReconnectingMPDClient(context.Background(), conn)
	t.Cleanup(client.Disconnect)

	if err := client.Stop(); !errors.Is(err, ErrMPD) {
		t.Errorf("MPD unreachable: error = %v, want ErrMPD", err)
	}
}
