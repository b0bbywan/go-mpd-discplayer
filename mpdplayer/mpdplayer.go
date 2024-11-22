package mpdplayer

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/fhs/gompd/v2/mpd"
	"github.com/b0bbywan/go-mpd-discplayer/config"
)

// ReconnectingMPDClient wraps gompd's MPD client and adds reconnection logic.
type ReconnectingMPDClient struct {
	mpcConfig		config.MPDConn
	client			*mpd.Client
	mu				sync.Mutex
}

// NewReconnectingMPDClient creates a new instance of ReconnectingMPDClient.
func NewReconnectingMPDClient(mpcConfig config.MPDConn) *ReconnectingMPDClient {
	return &ReconnectingMPDClient{
		mpcConfig: mpcConfig,
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
			if err := rc.Reconnect(); err != nil {
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

func (rc *ReconnectingMPDClient) Reconnect() error {
	rc.disconnectWithoutLock()
	return rc.connectWithoutLock()
}

func (rc *ReconnectingMPDClient) disconnectWithoutLock() {
	if rc.client != nil {
		rc.client.Close()
		rc.client = nil
	}
}

func (rc *ReconnectingMPDClient) connectWithoutLock() error {
	if rc.client != nil {
		rc.client.Close()
	}

	var err error
	start := time.Now()
	for retries := 0; time.Since(start) < rc.mpcConfig.ReconnectWait; retries++ {
		client, err := mpd.Dial(rc.mpcConfig.Type, rc.mpcConfig.Address)
		if err == nil {
			rc.client = client
			return nil
		}
		waitTime := reconnectingWaitTime(retries, rc.mpcConfig.ReconnectWait, start)
		// Calculate wait time with exponential backoff, capped by reconnectWait
		log.Printf("Retrying connection in %s: %w", waitTime, err)
		time.Sleep(waitTime)
	}
	return fmt.Errorf("failed to connect to MPD server after %s: %w", rc.mpcConfig.ReconnectWait, err)
}

func reconnectingWaitTime(retries int, reconnectWait time.Duration, start time.Time) time.Duration {
	maxWait := float64(reconnectWait.Seconds())
	waitTime := time.Duration(math.Min(math.Pow(2, float64(retries)), maxWait)) * time.Second

	// Ensure cumulative sleep doesn't exceed reconnectWait
	elapsed := time.Since(start)
	if elapsed+waitTime > reconnectWait {
		waitTime = reconnectWait - elapsed
	}

	return waitTime
}
