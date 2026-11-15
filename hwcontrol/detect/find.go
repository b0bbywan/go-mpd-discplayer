package detect

import (
	"errors"
	"fmt"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/blockdev"
)

var (
	ErrNoDisc       = errors.New("no disc in the drives")
	ErrSeveralDiscs = errors.New("discs in several drives")
	// Eject only: no disc is detected, so the drive cannot be picked by its disc
	ErrNoDrive       = errors.New("no disc drive")
	ErrSeveralDrives = errors.New("several empty disc drives")
)

// FindAudioDisc returns the drive holding an audio disc, as the daemon would
// play it on insertion.
func FindAudioDisc() (string, error) {
	return findDrive("an audio disc", func(device *udev.Device) bool {
		return discPreChecker(device) && hasAudioTracks(device)
	})
}

// FindEjectDrive returns the drive to eject: the drive holding a disc of any
// kind or, when none holds one, the only drive, so an empty tray opens too.
func FindEjectDrive() (string, error) {
	devices, err := blockdev.List()
	if err != nil {
		return "", fmt.Errorf("failed to list drives: %w", err)
	}

	var drives, loaded []string
	for _, device := range devices {
		if device.PropertyValue("ID_CDROM") != "1" {
			continue
		}
		drives = append(drives, device.Devnode())
		if device.PropertyValue("ID_CDROM_MEDIA") == "1" {
			loaded = append(loaded, device.Devnode())
		}
	}
	return pickEjectDrive(drives, loaded)
}

// pickEjectDrive prefers the drive holding a disc, then the only drive.
func pickEjectDrive(drives, loaded []string) (string, error) {
	if len(loaded) > 0 {
		return pickDrive("a disc", loaded)
	}
	switch len(drives) {
	case 0:
		return "", ErrNoDrive
	case 1:
		return drives[0], nil
	default:
		return "", fmt.Errorf("%w %v", ErrSeveralDrives, drives)
	}
}

func findDrive(what string, holds func(*udev.Device) bool) (string, error) {
	devices, err := blockdev.List()
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
