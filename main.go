package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/b0bbywan/go-mpd-discplayer/cmd"
)

func main() {
	// Define flags
	flag.Usage = usage
	playFlag := flag.Bool(cmd.ActionPlay, false, "Start playback immediately")
	stopFlag := flag.Bool(cmd.ActionStop, false, "Stop playback immediately")
	deviceFlag := flag.String("device", "/dev/sr0", "Disc Device")
	flag.Parse()

	if *playFlag && *stopFlag {
		flag.Usage()
		log.Fatalf("Cannot use --play and --stop together. Choose one.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go signalMonitor(ctx, cancel)

	player, err := cmd.NewPlayer()
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	// Handle flags
	if *playFlag {
		if err := player.ExecuteAction(*deviceFlag, cmd.ActionPlay); err != nil {
			log.Fatalf("Failed to start playback: %v", err)
		}
		return
	}
	if *stopFlag {
		if err := player.ExecuteAction(*deviceFlag, cmd.ActionStop); err != nil {
			log.Fatalf("Failed to stop playback: %v", err)
		}
		return
	}

	// Default behavior
	player.Start()

	<-ctx.Done()
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  mpd-discplayer [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --play   Start playback immediately")
	fmt.Println("  --stop   Stop playback immediately")
	fmt.Println("  --device <device>   Set device to play from (only with --play)")
	fmt.Println("  -h, --help   Display this help message")
}

func signalMonitor(ctx context.Context, cancel context.CancelFunc) {
	// Handle OS signals to gracefully stop the program
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)
	for range sigChan {
		log.Println("Received termination signal. Exiting...")
		cancel()
		return
	}
}
