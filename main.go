package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/b0bbywan/go-mpd-discplayer/cmd"
)

func main() {
	// Define flags
	flag.Usage = usage
	playFlag := flag.Bool(cmd.ActionPlay, false, "Start playback immediately")
	stopFlag := flag.Bool(cmd.ActionStop, false, "Stop playback immediately")
	flag.Parse()

	if *playFlag && *stopFlag {
		flag.Usage()
		log.Fatalf("Cannot use --play and --stop together. Choose one.")
	}

	// Handle flags
	if *playFlag {
		if err := cmd.ExecuteAction(cmd.ActionPlay); err != nil {
			log.Fatalf("Failed to start playback: %w", err)
		}
		return
	}
	if *stopFlag {
		if err := cmd.ExecuteAction(cmd.ActionStop); err != nil {
			log.Fatalf("Failed to stop playback: %w", err)
		}
		return
	}

	// Default behavior
	if err := cmd.Run(); err != nil {
		log.Fatalf("error: %w", err)
	}
}

func usage() {
    fmt.Println("Usage:")
    fmt.Println("  mpd-discplayer [options]")
    fmt.Println("")
    fmt.Println("Options:")
    fmt.Println("  --play   Start playback immediately")
    fmt.Println("  --stop   Stop playback immediately")
    fmt.Println("  -h, --help   Display this help message")
}
