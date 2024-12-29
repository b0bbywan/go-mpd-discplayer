package mpdplayer

import (
	"fmt"
	"time"

	"github.com/b0bbywan/go-disc-cuer/config"
)

type MPDConn struct {
	Type          string // "unix" or "tcp"
	Address       string // socket path or TCP address
	ReconnectWait time.Duration
	CuerConfig    *config.Config
}

func NewMPDConnection(connectionType, address string, reconnectWait time.Duration) (*MPDConn, error) {
	conn := &MPDConn{
		Type:          connectionType,
		Address:       address,
		ReconnectWait: reconnectWait,
	}

	if err := validateMPDConnection(conn); err != nil {
		return nil, fmt.Errorf("Failed to create valid MPD Config")
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

func (rc *ReconnectingMPDClient) SetCuerConfig(cuerConfig *config.Config) {
	rc.mpcConfig.CuerConfig = cuerConfig
}
