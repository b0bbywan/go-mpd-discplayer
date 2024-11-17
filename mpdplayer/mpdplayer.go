package mpdplayer

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fhs/gompd/v2/mpd"
)

// ReconnectingMPDClient wraps gompd's MPD client and adds reconnection logic.
type ReconnectingMPDClient struct {
	connectionType	string
	address			string
	reconnectWait	time.Duration
	client			*mpd.Client
	mu				sync.Mutex
}

// NewReconnectingMPDClient creates a new instance of ReconnectingMPDClient.
func NewReconnectingMPDClient(connectionType, address string, reconnectWait time.Duration) *ReconnectingMPDClient {
	return &ReconnectingMPDClient{
		connectionType: connectionType,
		address:       address,
		reconnectWait: reconnectWait,
	}
}

// Connect establishes the initial connection to the MPD server.
func (rc *ReconnectingMPDClient) Connect() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.connectWithoutLock()
}

// Execute runs the provided loadFunc, reconnecting if necessary.
func (rc *ReconnectingMPDClient) execute(loadFunc func(*mpd.Client) error) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Ensure connection is valid
	if rc.client == nil {
		if err := rc.connectWithoutLock(); err != nil {
			return fmt.Errorf("reconnection failed: %w", err)
		}
	}

	// Execute the provided function
	if err := loadFunc(rc.client); err != nil {
		// Handle connection issues and retry
		if isConnError(err) {
			log.Printf("Connection error detected: %v. Reconnecting...", err)
			rc.client.Close()
			rc.client = nil

			if err := rc.connectWithoutLock(); err != nil {
				return fmt.Errorf("reconnection failed: %w", err)
			}

			// Retry the function
			if err := loadFunc(rc.client); err != nil {
				return fmt.Errorf("function execution failed after reconnection: %w", err)
			}
		} else {
			return fmt.Errorf("function execution error: %w", err)
		}
	}

	return nil
}

// Disconnect safely closes the MPD connection.
func (rc *ReconnectingMPDClient) Disconnect() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.disconnectWithoutLock()
}

func (rc *ReconnectingMPDClient) disconnectWithoutLock() {
	if rc.client != nil {
		rc.client.Close()
		rc.client = nil
	}
}

func (rc *ReconnectingMPDClient) Reconnect() error {
	rc.disconnectWithoutLock()
	return rc.Connect()
}

func (rc *ReconnectingMPDClient) connectWithoutLock() error {
	if rc.client != nil {
		rc.client.Close()
	}

	client, err := mpd.Dial(rc.connectionType, rc.address)
	if err != nil {
		return fmt.Errorf("failed to connect to MPD server: %w", err)
	}

	rc.client = client
	return nil
}
