package cmd

import (
	"fmt"
	"io"
)

// CLI action names, the commands typed after the program name.
const (
	ActionDiscPlay   = "disc play"
	ActionDiscStop   = "disc stop"
	ActionDiscEject  = "disc eject"
	ActionMountList  = "mount list"
	ActionMountPlay  = "mount play"
	ActionMountStop  = "mount stop"
	ActionMountEject = "mount eject"
	ActionStreamPlay = "stream play"
	ActionStop       = "stop"
)

// Action is a CLI action. Device is the drive used by disc play and eject,
// Mount the USB storage of the mount actions, URL the stream played by
// stream play.
type Action struct {
	Name   string
	Device string
	Mount  string
	URL    string
}

// ExecuteAction runs a CLI action, writing its output, if any, to out.
func (player *Player) ExecuteAction(out io.Writer, action Action) error {
	switch action.Name {
	case ActionDiscPlay:
		return player.PlayDisc(action.Device)
	case ActionDiscStop:
		return player.StopDisc()
	case ActionDiscEject:
		return player.EjectDisc(action.Device)
	case ActionMountList:
		return player.printMounts(out)
	case ActionMountPlay:
		return player.PlayMount(action.Mount)
	case ActionMountStop:
		return player.StopMount(action.Mount)
	case ActionMountEject:
		return player.EjectMount(action.Mount)
	case ActionStreamPlay:
		return player.PlayURL(action.URL)
	case ActionStop:
		return player.Stop()
	default:
		return fmt.Errorf("unknown action: %s", action.Name)
	}
}

// printMounts writes the USB storages mounted in MPD, one per line: the
// name, then the storage.
func (player *Player) printMounts(out io.Writer) error {
	mounts, err := player.Mounts()
	if err != nil {
		return err
	}
	for _, m := range mounts {
		if _, err := fmt.Fprintf(out, "%s\t%s\n", m.Name, m.Storage); err != nil {
			return err
		}
	}
	return nil
}
