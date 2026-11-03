package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/b0bbywan/go-mpd-discplayer/cmd"
)

// options are the command-line options. An empty action runs the daemon.
type options struct {
	action  string
	device  string
	version bool
}

// actionFailures are the messages logged when a CLI action fails.
var actionFailures = map[string]string{
	cmd.ActionPlay:  "Failed to start playback",
	cmd.ActionStop:  "Failed to stop playback",
	cmd.ActionEject: "Failed to eject disc",
}

var errSeveralActions = errors.New("cannot use --play, --stop and --eject together, choose one")

// parseArgs parses the command-line arguments, without the program name.
func parseArgs(fs *flag.FlagSet, args []string) (options, error) {
	var opts options
	actions := map[string]*bool{
		cmd.ActionPlay:  fs.Bool(cmd.ActionPlay, false, "Play the disc"),
		cmd.ActionStop:  fs.Bool(cmd.ActionStop, false, "Stop playback, whatever is playing"),
		cmd.ActionEject: fs.Bool(cmd.ActionEject, false, "Stop disc playback and eject the disc"),
	}
	fs.StringVar(&opts.device, "device", "/dev/sr0", "Disc `device` used by --play and --eject")
	fs.BoolVar(&opts.version, "version", false, "Print version and exit")
	fs.Usage = func() { usage(fs) }

	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	for name, set := range actions {
		if !*set {
			continue
		}
		if opts.action != "" {
			fs.Usage()
			return opts, errSeveralActions
		}
		opts.action = name
	}
	return opts, nil
}

func usage(fs *flag.FlagSet) {
	out := fs.Output()
	_, _ = fmt.Fprintf(out, "Usage:\n  %s [options]\n\nOptions:\n", cmd.AppName)
	fs.VisitAll(func(f *flag.Flag) {
		arg, help := flag.UnquoteUsage(f)
		name := "--" + f.Name
		if arg != "" {
			name += " <" + arg + ">"
			help += fmt.Sprintf(" (default: %s)", f.DefValue)
		}
		_, _ = fmt.Fprintf(out, "  %-20s %s\n", name, help)
	})
	_, _ = fmt.Fprintf(out, "  %-20s %s\n", "-h, --help", "Display this help message")
}
