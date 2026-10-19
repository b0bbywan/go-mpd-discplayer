package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/b0bbywan/go-mpd-discplayer/cmd"
)

func main() {
	opts, err := parseArgs(flag.CommandLine, os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		log.Fatal(err)
	}

	if opts.version {
		fmt.Printf("%s %s\n", cmd.AppName, cmd.AppVersion)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	player, err := cmd.NewPlayer(ctx, cancel)
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	if opts.action != "" {
		if err := player.ExecuteAction(opts.device, opts.action); err != nil {
			log.Fatalf("%s: %v", actionFailures[opts.action], err)
		}
		return
	}

	go signalMonitor(ctx, cancel)

	// Default behavior
	player.Start()

	<-ctx.Done()
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
