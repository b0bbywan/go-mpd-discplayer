package main

import (
	"flag"
	"log"

	"github.com/b0bbywan/go-mpd-discplayer/cmd"
)

func main() {
	// Define flags
	playFlag := flag.Bool(cmd.ActionPlay, false, "Start playback immediately")
	stopFlag := flag.Bool(cmd.ActionStop, false, "Stop playback immediately")
	flag.Parse()

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
