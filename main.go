package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
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

	// Initialize context and WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	player, err := cmd.NewPlayer(ctx, cancel)
	defer player.Cancel()
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}

	var wg sync.WaitGroup
	defer cleanUp(&wg, player)

	// Signal handling goroutine to cleanly stop the program
	wg.Add(1)
	go signalMonitor(&wg, cancel)

	// Handle flags
	if *playFlag {
		if err := cmd.ExecuteAction(player, *deviceFlag, cmd.ActionPlay); err != nil {
			log.Fatalf("Failed to start playback: %v", err)
		}
		return
	}
	if *stopFlag {
		if err := cmd.ExecuteAction(player, *deviceFlag, cmd.ActionStop); err != nil {
			log.Fatalf("Failed to stop playback: %v", err)
		}
		return
	}

	// Default behavior
	if err := cmd.Run(&wg, player); err != nil {
		log.Fatalf("error: %v", err)
	}
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

func signalMonitor(wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()
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

func cleanUp(wg *sync.WaitGroup, player *cmd.Player) {
	<-player.Ctx().Done()

	// Cleanup after receiving the termination signal
	player.Close()

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("All tasks completed. Exiting...")
}
