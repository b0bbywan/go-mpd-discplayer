package mounts

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/blockdev"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

type mpdFinder struct {
	client           *mpdplayer.ReconnectingMPDClient
	mpdLibraryFolder string
}

func newMpdFinder(client *mpdplayer.ReconnectingMPDClient, mpdLibraryFolder string) (*mpdFinder, error) {
	// Mounts of devices still plugged in are kept (MPD restores them from its
	// state file), the others are removed. Rely on udev rather than MPD
	// neighbors, which may not list the devices yet at boot.
	if present, err := presentIdentifiers(); err != nil {
		log.Printf("Skipping stale mounts cleanup: %v", err)
	} else if err := client.ClearStaleMounts(present); err != nil {
		log.Printf("warning: %v", err)
	}
	return &mpdFinder{
		client:           client,
		mpdLibraryFolder: mpdLibraryFolder,
	}, nil
}

func (m *mpdFinder) validate(device *udev.Device, mountpoint, target string) (string, error) {
	return m.mount(device, mountpoint, target)
}

func (m *mpdFinder) clear(device *udev.Device, mountpoint string) (string, error) {
	return m.unmount(device)
}

func (m *mpdFinder) mount(device *udev.Device, mountpoint, target string) (string, error) {
	identifiers := neighborIdentifiers(device)
	label := deviceLabel(device)
	if err := m.mountWithRetry(identifiers, label); err != nil {
		return "", fmt.Errorf("failed to mount %s -> %s: %w", device.Devnode(), label, err)
	}
	return filepath.Join(m.mpdLibraryFolder, label), nil
}

func (m *mpdFinder) mountWithRetry(identifiers []string, label string) error {
	ticker := time.NewTicker(RetryInterval)
	defer ticker.Stop()
	timeoutChan := time.After(2 * RetryTimeout)

	for {
		err := m.client.Mount(identifiers, label)
		if err == nil {
			return nil
		}
		select {
		case <-ticker.C:
			log.Printf("Polling for neighbor matching %v...", identifiers)
		case <-timeoutChan:
			return err
		}
	}
}

// presentIdentifiers returns the identifiers of all block devices currently known to udev.
func presentIdentifiers() ([]string, error) {
	devices, err := blockdev.List()
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, dev := range devices {
		ids = append(ids, neighborIdentifiers(dev)...)
	}
	return ids, nil
}

func neighborIdentifiers(device *udev.Device) []string {
	var ids []string
	if uuid := device.PropertyValue("ID_FS_UUID"); uuid != "" {
		ids = append(ids, uuid)
	}
	if serial := device.PropertyValue("ID_SERIAL"); serial != "" {
		ids = append(ids, serial)
	}
	return ids
}

func (m *mpdFinder) unmount(device *udev.Device) (string, error) {
	label := deviceLabel(device)
	if err := m.client.Unmount(label); err != nil {
		return "", fmt.Errorf("failed to unmount %s: %w", device.Devnode(), err)
	}
	return filepath.Join(m.mpdLibraryFolder, label), nil
}

func deviceLabel(device *udev.Device) string {
	if label := device.PropertyValue("ID_FS_LABEL"); label != "" {
		return label
	}
	return device.PropertyValue("ID_FS_UUID")
}
