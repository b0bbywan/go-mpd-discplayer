// Package blockdev lists the block devices known to udev.
package blockdev

import (
	"fmt"

	"github.com/jochenvg/go-udev"
)

// List returns the block devices udev has finished processing.
func List() ([]*udev.Device, error) {
	u := udev.Udev{}
	e := u.NewEnumerate()
	if err := e.AddMatchSubsystem("block"); err != nil {
		return nil, fmt.Errorf("failed to add filter: %w", err)
	}
	if err := e.AddMatchIsInitialized(); err != nil {
		return nil, fmt.Errorf("failed to add filter: %w", err)
	}
	devices, err := e.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate block devices: %w", err)
	}
	return devices, nil
}
