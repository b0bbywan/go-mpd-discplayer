package mpdplayer

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/fhs/gompd/v2/mpd"
)

// ReconnectingMPDClient wraps gompd's MPD client and adds reconnection logic.
type ReconnectingMPDClient struct {
	mpcConfig *MPDConn
	client    *mpd.Client
	mu        sync.Mutex
	ctx       context.Context
}

// NewReconnectingMPDClient creates a new instance of ReconnectingMPDClient.
func NewReconnectingMPDClient(ctx context.Context, mpcConfig *MPDConn) *ReconnectingMPDClient {
	return &ReconnectingMPDClient{
		mpcConfig: mpcConfig,
		ctx:       ctx,
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

	for {
		if err := loadFunc(rc.client); err != nil {
			if isConnError(err) {
				log.Printf("Connection error detected: %v. Reconnecting...", err)
				if reconnectErr := rc.Reconnect(); reconnectErr != nil {
					return fmt.Errorf("reconnection failed: %w", reconnectErr)
				}
				// After reconnection, retry loadFunc
				continue
			}
			// For non-connection errors, return immediately
			return fmt.Errorf("function execution error: %w", err)
		}
		// If loadFunc succeeds, break the loop
		break
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
		// Calculate wait time with exponential backoff, capped by reconnectWait
		waitTime := reconnectingWaitTime(retries, rc.mpcConfig.ReconnectWait, start)

		// Wait for the exponential backoff period, checking context for cancellation
		select {
		case <-rc.ctx.Done():
			log.Println("Reconnection attempt canceled by context.")
			return rc.ctx.Err()
		case <-time.After(waitTime): // Sleep for the calculated retry interval
		}
	}
	return fmt.Errorf("failed to connect to MPD server %s://%s after %s: %w", rc.mpcConfig.Type, rc.mpcConfig.Address, rc.mpcConfig.ReconnectWait, err)
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
