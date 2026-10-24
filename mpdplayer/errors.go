package mpdplayer

import (
	"errors"
	"strings"
)

// ErrUnknownMount is returned for a name that is not a USB storage mounted in MPD.
var ErrUnknownMount = errors.New("no USB storage mounted in MPD with this name")

func isConnError(err error) bool {
	if err == nil {
		return false
	}
	// Example: Check for common connection error messages
	return strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "EOF")
}
