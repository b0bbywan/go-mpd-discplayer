package mpdplayer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeMount struct {
	mount, storage string
}

// fakeMPD is a minimal in-memory MPD server speaking the text protocol. It
// keeps the state the operations under test rely on and records every command.
type fakeMPD struct {
	mu        sync.Mutex
	queue     []string
	current   int // index in queue, -1 when nothing is playing
	mounts    []fakeMount
	neighbors []string
	commands  []string
}

func newFakeMPD() *fakeMPD {
	return &fakeMPD{current: -1}
}

// start serves f on a local TCP port and returns a client connected to it.
func (f *fakeMPD) start(t *testing.T) *ReconnectingMPDClient {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(conn)
		}
	}()

	conn, err := NewMPDConnection("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client := NewReconnectingMPDClient(context.Background(), conn)
	t.Cleanup(client.Disconnect)
	return client
}

func (f *fakeMPD) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	if _, err := io.WriteString(conn, "OK MPD 0.24.0\n"); err != nil {
		return
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "close" {
			return
		}
		if _, err := io.WriteString(conn, f.handle(line)); err != nil {
			return
		}
	}
}

func (f *fakeMPD) handle(line string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands = append(f.commands, line)

	name, args := parseCommand(line)
	switch name {
	case "clear":
		f.queue, f.current = nil, -1
	case "add":
		f.queue = append(f.queue, args[0])
	case "play":
		if len(f.queue) > 0 {
			f.current = 0
		}
	case "stop":
		f.current = -1
	case "currentsong":
		if f.current < 0 {
			return "OK\n"
		}
		return fmt.Sprintf("file: %s\nPos: %d\nOK\n", f.queue[f.current], f.current)
	case "playlistinfo":
		var b strings.Builder
		for i, file := range f.queue {
			fmt.Fprintf(&b, "file: %s\nPos: %d\n", file, i)
		}
		return b.String() + "OK\n"
	case "delete":
		start, end, err := parseRange(args[0])
		if err != nil || start > end || end > len(f.queue) {
			return "ACK [2@0] {delete} Bad song index\n"
		}
		f.queue = append(f.queue[:start:start], f.queue[end:]...)
	case "status":
		return fmt.Sprintf("playlistlength: %d\nOK\n", len(f.queue))
	case "update":
		return "updating_db: 1\nOK\n"
	case "listmounts":
		var b strings.Builder
		for _, m := range f.mounts {
			fmt.Fprintf(&b, "mount: %s\nstorage: %s\n", m.mount, m.storage)
		}
		return b.String() + "OK\n"
	case "listneighbors":
		var b strings.Builder
		for _, n := range f.neighbors {
			fmt.Fprintf(&b, "neighbor: %s\nname: %s\n", n, n)
		}
		return b.String() + "OK\n"
	case "mount":
		for _, m := range f.mounts {
			if m.mount == args[0] {
				return "ACK [56@0] {mount} Mount point busy\n"
			}
		}
		f.mounts = append(f.mounts, fakeMount{args[0], args[1]})
	case "unmount":
		for i, m := range f.mounts {
			if m.mount == args[0] {
				f.mounts = append(f.mounts[:i:i], f.mounts[i+1:]...)
				return "OK\n"
			}
		}
		return "ACK [50@0] {unmount} Not a mount point\n"
	default:
		return fmt.Sprintf("ACK [5@0] {} unknown command %q\n", name)
	}
	return "OK\n"
}

// sent returns the commands received so far, without the quotes around arguments.
func (f *fakeMPD) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.commands))
	for i, c := range f.commands {
		name, args := parseCommand(c)
		out[i] = strings.Join(append([]string{name}, args...), " ")
	}
	return out
}

// parseCommand splits an MPD command line into its name and unquoted arguments.
func parseCommand(line string) (string, []string) {
	var tokens []string
	var cur strings.Builder
	inQuotes, escaped, hasToken := false, false, false
	for _, c := range line {
		switch {
		case escaped:
			cur.WriteRune(c)
			escaped = false
		case c == '\\' && inQuotes:
			escaped = true
		case c == '"':
			inQuotes = !inQuotes
			hasToken = true
		case c == ' ' && !inQuotes:
			if hasToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				hasToken = false
			}
		default:
			cur.WriteRune(c)
			hasToken = true
		}
	}
	if hasToken {
		tokens = append(tokens, cur.String())
	}
	if len(tokens) == 0 {
		return "", nil
	}
	return tokens[0], tokens[1:]
}

// parseRange parses "N" or "START:END" into a half-open range.
func parseRange(s string) (int, int, error) {
	if before, after, ok := strings.Cut(s, ":"); ok {
		start, err := strconv.Atoi(before)
		if err != nil {
			return 0, 0, err
		}
		end, err := strconv.Atoi(after)
		return start, end, err
	}
	n, err := strconv.Atoi(s)
	return n, n + 1, err
}
