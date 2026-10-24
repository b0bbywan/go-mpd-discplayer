package main

import (
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/b0bbywan/go-mpd-discplayer/cmd"
)

const defaultDevice = "/dev/sr0"

// options are the command-line options. An empty action runs the daemon.
type options struct {
	action  cmd.Action
	version bool
}

// actionFailures are the messages logged when a CLI action fails.
var actionFailures = map[string]string{
	cmd.ActionDiscPlay:   "Failed to start disc playback",
	cmd.ActionDiscStop:   "Failed to stop disc playback",
	cmd.ActionDiscEject:  "Failed to eject disc",
	cmd.ActionMountList:  "Failed to list mounts",
	cmd.ActionMountPlay:  "Failed to start USB playback",
	cmd.ActionMountStop:  "Failed to stop USB playback",
	cmd.ActionMountEject: "Failed to unmount USB storage",
	cmd.ActionStreamPlay: "Failed to play stream",
	cmd.ActionStop:       "Failed to stop playback",
}

// legacyActions maps the deprecated action flags to their command.
var legacyActions = map[string]string{
	"play":  cmd.ActionDiscPlay,
	"stop":  cmd.ActionStop,
	"eject": cmd.ActionDiscEject,
}

var (
	errSeveralActions = errors.New("cannot use --play, --stop and --eject together, choose one")
	errMixedActions   = errors.New("cannot use --play, --stop or --eject with a command")
)

// parseArgs parses the command-line arguments, without the program name.
func parseArgs(fs *flag.FlagSet, args []string) (options, error) {
	var opts options
	legacy := map[string]*bool{}
	for name, action := range legacyActions {
		legacy[name] = fs.Bool(name, false, fmt.Sprintf("Deprecated, same as %q", action))
	}
	device := fs.String("device", defaultDevice, "Disc device used by disc play and disc eject")
	fs.BoolVar(&opts.version, "version", false, "Print version and exit")
	fs.Usage = func() { usage(fs) }

	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	for name, set := range legacy {
		if !*set {
			continue
		}
		if opts.action.Name != "" {
			return opts, usageError(fs, errSeveralActions)
		}
		opts.action = cmd.Action{Name: legacyActions[name]}
		_, _ = fmt.Fprintf(fs.Output(), "warning: --%s is deprecated, use \"%s %s\"\n",
			name, cmd.AppName, legacyActions[name])
	}
	if opts.action.Name != "" {
		if fs.NArg() > 0 {
			return opts, usageError(fs, errMixedActions)
		}
		if opts.action.Name == cmd.ActionDiscPlay || opts.action.Name == cmd.ActionDiscEject {
			opts.action.Device = *device
		}
		return opts, nil
	}

	if fs.NArg() == 0 {
		return opts, nil
	}
	var err error
	opts.action, err = parseCommand(fs, fs.Args(), *device)
	return opts, err
}

// parseCommand parses a command and its arguments: disc play|stop|eject,
// mount list|play|stop|eject, stream play <url> or stop.
func parseCommand(fs *flag.FlagSet, args []string, device string) (cmd.Action, error) {
	switch {
	case args[0] == "stop" && len(args) == 1:
		return cmd.Action{Name: cmd.ActionStop}, nil
	case args[0] == "mount" && len(args) == 2 && args[1] == "list":
		return cmd.Action{Name: cmd.ActionMountList}, nil
	case args[0] == "mount" && len(args) == 3 && slices.Contains([]string{"play", "stop", "eject"}, args[1]):
		return cmd.Action{Name: "mount " + args[1], Mount: args[2]}, nil
	case args[0] == "stream" && len(args) == 3 && args[1] == "play":
		return cmd.Action{Name: cmd.ActionStreamPlay, URL: args[2]}, nil
	case args[0] == "disc" && len(args) >= 2:
		return parseDisc(fs, args[1], args[2:], device)
	}
	return cmd.Action{}, usageError(fs, fmt.Errorf("invalid command: %s", strings.Join(args, " ")))
}

// parseDisc parses a disc command, play and eject accepting --device.
func parseDisc(fs *flag.FlagSet, name string, args []string, device string) (cmd.Action, error) {
	action := cmd.Action{Name: "disc " + name}
	sub := flag.NewFlagSet(action.Name, fs.ErrorHandling())
	sub.SetOutput(fs.Output())
	sub.Usage = fs.Usage
	switch action.Name {
	case cmd.ActionDiscPlay, cmd.ActionDiscEject:
		action.Device = device
		sub.StringVar(&action.Device, "device", device, "Disc device")
	case cmd.ActionDiscStop:
	default:
		return action, usageError(fs, fmt.Errorf("invalid command: %s", action.Name))
	}

	if err := sub.Parse(args); err != nil {
		return action, err
	}
	if sub.NArg() > 0 {
		return action, usageError(fs, fmt.Errorf("unexpected argument: %s", sub.Arg(0)))
	}
	return action, nil
}

// usageError prints the usage, then returns err.
func usageError(fs *flag.FlagSet, err error) error {
	fs.Usage()
	return err
}

func usage(fs *flag.FlagSet) {
	_, _ = fmt.Fprintf(fs.Output(), `Usage:
  %[1]s               Run the daemon
  %[1]s <command>     Run a single action and exit
  %[1]s --version     Print version and exit

Commands:
  disc play [--device <device>]    Play the disc
  disc stop                        Stop playback if the disc is playing, the queue is left untouched
  disc eject [--device <device>]   Stop disc playback and eject the disc
  mount list                       List the USB storages mounted in MPD: name, then storage
  mount play <name>                Play a USB storage
  mount stop <name>                Stop playback if the USB storage is playing, the queue is left untouched
  mount eject <name>               Stop the USB storage playback and unmount it from MPD
  stream play <url>                Play a URL, a webradio for instance
  stop                             Stop playback, whatever is playing

Options:
  --device <device>   Disc device used by disc play and disc eject (default: %[2]s)
  --version           Print version and exit
  -h, --help          Display this help message

Deprecated options:
  --play    Same as "disc play"
  --stop    Same as "stop"
  --eject   Same as "disc eject"
`, cmd.AppName, defaultDevice)
}
