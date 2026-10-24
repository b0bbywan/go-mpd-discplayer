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
	discPlay := func(device string) options {
		return options{action: cmd.Action{Name: cmd.ActionDiscPlay, Device: device}}
	}
	tests := []struct {
		args []string
		want options
	}{
		{nil, options{}},
		{[]string{"--version"}, options{version: true}},
		{[]string{"disc", "play"}, discPlay("/dev/sr0")},
		{[]string{"disc", "play", "--device", "/dev/sr1"}, discPlay("/dev/sr1")},
		{[]string{"--device", "/dev/sr1", "disc", "play"}, discPlay("/dev/sr1")},
		{[]string{"disc", "stop"}, options{action: cmd.Action{Name: cmd.ActionDiscStop}}},
		{[]string{"disc", "eject", "-device=/dev/sr1"}, options{action: cmd.Action{Name: cmd.ActionDiscEject, Device: "/dev/sr1"}}},
		{[]string{"mount", "list"}, options{action: cmd.Action{Name: cmd.ActionMountList}}},
		{[]string{"mount", "play", "MYUSB"}, options{action: cmd.Action{Name: cmd.ActionMountPlay, Mount: "MYUSB"}}},
		{[]string{"mount", "stop", "MY USB"}, options{action: cmd.Action{Name: cmd.ActionMountStop, Mount: "MY USB"}}},
		{[]string{"mount", "eject", "MYUSB"}, options{action: cmd.Action{Name: cmd.ActionMountEject, Mount: "MYUSB"}}},
		{[]string{"stream", "play", "https://radio/live"}, options{action: cmd.Action{Name: cmd.ActionStreamPlay, URL: "https://radio/live"}}},
		{[]string{"stop"}, options{action: cmd.Action{Name: cmd.ActionStop}}},
	}
	for _, tt := range tests {
		got, out, err := parse(tt.args...)
		if err != nil || got != tt.want {
			t.Errorf("parseArgs(%q) = %+v, %v, want %+v", tt.args, got, err, tt.want)
		}
		if out != "" {
			t.Errorf("parseArgs(%q) printed %q", tt.args, out)
		}
	}
}

func TestParseArgsLegacy(t *testing.T) {
	tests := []struct {
		args []string
		want cmd.Action
	}{
		{[]string{"--play"}, cmd.Action{Name: cmd.ActionDiscPlay, Device: "/dev/sr0"}},
		{[]string{"--stop"}, cmd.Action{Name: cmd.ActionStop}},
		{[]string{"--eject", "--device", "/dev/sr1"}, cmd.Action{Name: cmd.ActionDiscEject, Device: "/dev/sr1"}},
		{[]string{"-play", "-device=/dev/sr1"}, cmd.Action{Name: cmd.ActionDiscPlay, Device: "/dev/sr1"}},
	}
	for _, tt := range tests {
		got, out, err := parse(tt.args...)
		if err != nil || got.action != tt.want {
			t.Errorf("parseArgs(%q) = %+v, %v, want %+v", tt.args, got.action, err, tt.want)
		}
		if !strings.Contains(out, "deprecated") || !strings.Contains(out, tt.want.Name) {
			t.Errorf("parseArgs(%q) did not warn about the deprecation: %q", tt.args, out)
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
		{[]string{"--play", "disc", "play"}, errMixedActions},
		{[]string{"-h"}, flag.ErrHelp},
		{[]string{"disc", "play", "-h"}, flag.ErrHelp},
		{[]string{"--rewind"}, nil},
		{[]string{"rewind"}, nil},
		{[]string{"disc"}, nil},
		{[]string{"disc", "rewind"}, nil},
		{[]string{"disc", "stop", "--device", "/dev/sr1"}, nil},
		{[]string{"disc", "play", "extra"}, nil},
		{[]string{"mount"}, nil},
		{[]string{"mount", "list", "MYUSB"}, nil},
		{[]string{"mount", "play"}, nil},
		{[]string{"mount", "rewind", "MYUSB"}, nil},
		{[]string{"mount", "play", "MYUSB", "extra"}, nil},
		{[]string{"stream", "play"}, nil},
		{[]string{"stream", "play", "a", "b"}, nil},
		{[]string{"stop", "now"}, nil},
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
		"disc play [--device <device>]", "disc stop", "disc eject [--device <device>]",
		"mount list", "mount play <name>", "mount stop <name>", "mount eject <name>",
		"stream play <url>", "stop ", "--version ", "(default: /dev/sr0)", "-h, --help",
		"--play ", "--stop ", "--eject ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("usage does not contain %q:\n%s", want, out)
		}
	}
}

func TestActionFailures(t *testing.T) {
	for _, action := range []string{
		cmd.ActionDiscPlay, cmd.ActionDiscStop, cmd.ActionDiscEject,
		cmd.ActionMountList, cmd.ActionMountPlay, cmd.ActionMountStop, cmd.ActionMountEject,
		cmd.ActionStreamPlay, cmd.ActionStop,
	} {
		if actionFailures[action] == "" {
			t.Errorf("no failure message for %s", action)
		}
	}
}
