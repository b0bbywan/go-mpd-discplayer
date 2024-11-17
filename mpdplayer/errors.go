package mpdplayer

import (
	"strings"
)

func isConnError(err error) bool {
	if err == nil {
		return false
	}
	// Example: Check for common connection error messages
	return strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "EOF")
}
