package mpdplayer

import (
	"fmt"
	"time"
)

type MPDConn struct {
	Type          string // "unix" or "tcp"
	Address       string // socket path or TCP address
	ReconnectWait time.Duration
	CueFolder     string // folder of the disc CUE sheets in the MPD library, "" when out of it
}

func NewMPDConnection(connectionType, address string, reconnectWait time.Duration, cueFolder string) (*MPDConn, error) {
	conn := &MPDConn{
		Type:          connectionType,
		Address:       address,
		ReconnectWait: reconnectWait,
		CueFolder:     cueFolder,
	}

	if err := validateMPDConnection(conn); err != nil {
		return nil, fmt.Errorf("failed to create valid MPD config")
	}
	return conn, nil
}

// validateMPDConnection checks the validity of the MPD connection settings
func validateMPDConnection(conn *MPDConn) error {
	if conn.Type != "unix" && conn.Type != "tcp" {
		return fmt.Errorf("invalid MPDConnection.Type: %s, must be 'unix' or 'tcp'", conn.Type)
	}
	if conn.Address == "" {
		return fmt.Errorf("MPDConnection.Address cannot be empty")
	}
	return nil
}
