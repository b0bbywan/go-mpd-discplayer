// Package retry polls an operation until it succeeds or a timeout expires.
package retry

import "time"

// Until calls fn right away, then every interval, until it returns nil or
// timeout expires. It returns the last error of fn on timeout.
func Until(timeout, interval time.Duration, fn func() error) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	deadline := time.After(timeout)
	for {
		err := fn()
		if err == nil {
			return nil
		}
		select {
		case <-ticker.C:
		case <-deadline:
			return err
		}
	}
}
