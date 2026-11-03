package main

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/b0bbywan/go-mpd-discplayer/cmd"
)

func parse(args ...string) (options, string, error) {
	fs := flag.NewFlagSet(cmd.AppName, flag.ContinueOnError)
	var out bytes.Buffer
	fs.SetOutput(&out)
	opts, err := parseArgs(fs, args)
	return opts, out.String(), err
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args []string
		want options
	}{
		{nil, options{device: "/dev/sr0"}},
		{[]string{"--play"}, options{action: cmd.ActionPlay, device: "/dev/sr0"}},
		{[]string{"--stop"}, options{action: cmd.ActionStop, device: "/dev/sr0"}},
		{[]string{"--eject", "--device", "/dev/sr1"}, options{action: cmd.ActionEject, device: "/dev/sr1"}},
		{[]string{"-play", "-device=/dev/sr1"}, options{action: cmd.ActionPlay, device: "/dev/sr1"}},
		{[]string{"--version"}, options{device: "/dev/sr0", version: true}},
	}
	for _, tt := range tests {
		got, _, err := parse(tt.args...)
		if err != nil || got != tt.want {
			t.Errorf("parseArgs(%q) = %+v, %v, want %+v", tt.args, got, err, tt.want)
		}
	}
}

func TestParseArgsErrors(t *testing.T) {
	tests := []struct {
		args    []string
		wantErr error
	}{
		{[]string{"--play", "--stop"}, errSeveralActions},
		{[]string{"--stop", "--eject"}, errSeveralActions},
		{[]string{"-h"}, flag.ErrHelp},
		{[]string{"--rewind"}, nil},
	}
	for _, tt := range tests {
		_, out, err := parse(tt.args...)
		if err == nil || (tt.wantErr != nil && !errors.Is(err, tt.wantErr)) {
			t.Errorf("parseArgs(%q) error = %v, want %v", tt.args, err, tt.wantErr)
		}
		if !strings.Contains(out, "Usage:") {
			t.Errorf("parseArgs(%q) did not print the usage", tt.args)
		}
	}
}

func TestUsage(t *testing.T) {
	_, out, _ := parse("-h")
	for _, want := range []string{
		"--play ", "--stop ", "--eject ", "--version ",
		"--device <device>", "(default: /dev/sr0)", "-h, --help",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("usage does not contain %q:\n%s", want, out)
		}
	}
}

func TestActionFailures(t *testing.T) {
	for _, action := range []string{cmd.ActionPlay, cmd.ActionStop, cmd.ActionEject} {
		if actionFailures[action] == "" {
			t.Errorf("no failure message for %s", action)
		}
	}
}
