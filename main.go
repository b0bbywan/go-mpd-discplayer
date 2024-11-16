package main

import (
	"log"

	"github.com/b0bbywan/go-mpd-discplayer/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
