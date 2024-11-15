package main

import (
	"log"
	"os"

	"github.com/b0bbywan/go-mpd-discplayer/cmd"
)

var action string

// init handles argument validation and sets the action variable
func init() {
	if len(os.Args) < 2 {
		log.Fatalf("error: No ACTION specified")
	}
	action = os.Args[1]
}

func main() {
	if err := cmd.ExecuteAction(action); err != nil {
		log.Fatalf("error: %v", err)
	}
}
