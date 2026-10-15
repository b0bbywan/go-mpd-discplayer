package detect

import (
	"errors"
	"fmt"

	"github.com/jochenvg/go-udev"
)

var (
	ErrNoDisc       = errors.New("no disc in the drives")
	ErrSeveralDiscs = errors.New("discs in several drives")
)

// FindAudioDisc returns the drive holding an audio disc, as the daemon would
// play it on insertion.
func FindAudioDisc() (string, error) {
	return findDrive("an audio disc", func(device *udev.Device) bool {
		return discPreChecker(device) && hasAudioTracks(device)
	})
}

// FindDisc returns the drive holding a disc of any kind.
func FindDisc() (string, error) {
	return findDrive("a disc", func(device *udev.Device) bool {
		return device.PropertyValue("ID_CDROM_MEDIA") == "1"
	})
}

func findDrive(what string, holds func(*udev.Device) bool) (string, error) {
	u := udev.Udev{}
	enumerate := u.NewEnumerate()
	if err := enumerate.AddMatchSubsystem("block"); err != nil {
		return "", fmt.Errorf("failed to add filter: %w", err)
	}
	if err := enumerate.AddMatchProperty("ID_CDROM", "1"); err != nil {
		return "", fmt.Errorf("failed to add filter: %w", err)
	}
	devices, err := enumerate.Devices()
	if err != nil {
		return "", fmt.Errorf("failed to list drives: %w", err)
	}

	var drives []string
	for _, device := range devices {
		if holds(device) {
			drives = append(drives, device.Devnode())
		}
	}
	return pickDrive(what, drives)
}

// pickDrive returns the only drive, playing several discs is not supported.
func pickDrive(what string, drives []string) (string, error) {
	switch len(drives) {
	case 0:
		return "", fmt.Errorf("looking for %s: %w", what, ErrNoDisc)
	case 1:
		return drives[0], nil
	default:
		return "", fmt.Errorf("looking for %s: %w %v", what, ErrSeveralDiscs, drives)
	}
}
