package mpdplayer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fhs/gompd/v2/mpd"
)

// ErrMPD is returned when MPD cannot be reached or fails a command.
var ErrMPD = errors.New("mpd error")

func isConnError(err error) bool {
	if err == nil {
		return false
	}
	// Example: Check for common connection error messages
	return strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "EOF")
}

// markMPDError wraps err with ErrMPD when MPD failed it. Other errors, from
// the drive for instance, are returned as they are.
func markMPDError(err error) error {
	var mpdErr mpd.Error
	if errors.As(err, &mpdErr) {
		return fmt.Errorf("%w: %w", ErrMPD, err)
	}
	return err
}
