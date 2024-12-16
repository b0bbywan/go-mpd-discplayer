package mounts

import (
	"fmt"
	"path/filepath"

	"github.com/jochenvg/go-udev"

	"github.com/b0bbywan/go-mpd-discplayer/config"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

type mpdFinder struct {
	client *mpdplayer.ReconnectingMPDClient
}

func newMpdFinder(client *mpdplayer.ReconnectingMPDClient) (*mpdFinder, error) {
	if err := client.ClearMounts(); err != nil {
		return nil, fmt.Errorf("Failed to clear mounts: %w", err)
	}
	return &mpdFinder{
		client: client,
	}, nil
}

func (m *mpdFinder) validate(device *udev.Device, mountpoint, target string) (string, error) {
	return m.mount(device, mountpoint, target)
}

func (m *mpdFinder) clear(device *udev.Device, mountpoint string) (string, error) {
	return m.unmount(device)
}

func (m *mpdFinder) mount(device *udev.Device, mountpoint, target string) (string, error) {
	udiskId := device.PropertyValue("ID_FS_UUID")
	label := device.PropertyValue("ID_FS_LABEL")
	if err := m.client.Mount(udiskId, label); err != nil {
		return "", fmt.Errorf("Failed to mount %s -> %s: %w", device.Devnode(), label, err)
	}
	return filepath.Join(config.MPDLibraryFolder, label), nil
}

func (m *mpdFinder) unmount(device *udev.Device) (string, error) {
	label := device.PropertyValue("ID_FS_LABEL")
	if err := m.client.Unmount(label); err != nil {
		return "", fmt.Errorf("Failed to unmount %s: %w", device.Devnode(), err)
	}
	return filepath.Join(config.MPDLibraryFolder, label), nil
}
