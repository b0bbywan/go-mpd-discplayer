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
	ejectFlag := flag.Bool(cmd.ActionEject, false, "Stop disc playback and eject the disc")
	deviceFlag := flag.String("device", "/dev/sr0", "Disc Device")
	versionFlag := flag.Bool("version", false, "Print version")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s %s\n", cmd.AppName, cmd.AppVersion)
		return
	}

	actions := 0
	for _, set := range []bool{*playFlag, *stopFlag, *ejectFlag} {
		if set {
			actions++
		}
	}
	if actions > 1 {
		flag.Usage()
		log.Fatalf("Cannot use --play, --stop and --eject together. Choose one.")
	}

	ctx, cancel := context.WithCancel(context.Background())

	player, err := cmd.NewPlayer(ctx, cancel)
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
	if *ejectFlag {
		if err := player.ExecuteAction(*deviceFlag, cmd.ActionEject); err != nil {
			log.Fatalf("Failed to eject disc: %v", err)
		}
		return
	}

	go signalMonitor(ctx, cancel)

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
	fmt.Println("  --eject  Stop disc playback and eject the disc")
	fmt.Println("  --device <device>   Set disc device (only with --play or --eject)")
	fmt.Println("  -h, --help   Display this help message")
}

func signalMonitor(ctx context.Context, cancel context.CancelFunc) {
	// Handle OS signals to gracefully stop the program
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case <-sigChan:
		log.Println("Received termination signal. Exiting...")
		cancel()
	case <-ctx.Done():
		log.Println("Received context signal. Exiting...")
	}
}
