package cmd

import (
	"fmt"
	"io"
	"strconv"
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
	ActionStatus     = "status"
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
	case ActionStatus:
		return player.printStatus(out)
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

// printStatus writes the playback state, one "key: value" per line with the
// keys of GET /player/status, leaving out what is missing, then one
// "mounts: name<TAB>storage" line per USB storage mounted in MPD.
func (player *Player) printStatus(out io.Writer) error {
	st, err := player.Status()
	if err != nil {
		return err
	}
	pb := st.Playback
	lines := [][2]string{
		{"state", pb.State},
		{"source", pb.Source},
		{"mount", pb.Mount},
		{"title", pb.Title},
		{"artist", pb.Artist},
		{"album", pb.Album},
		{"name", pb.Name},
		{"url", pb.URL},
		{"position", formatPosition(pb.Position)},
		{"length", strconv.Itoa(pb.Length)},
		{"elapsed", formatSeconds(pb.Elapsed)},
		{"duration", formatSeconds(pb.Duration)},
	}
	for _, m := range st.Mounts {
		lines = append(lines, [2]string{"mounts", m.Name + "\t" + m.Storage})
	}
	for _, l := range lines {
		if l[1] == "" {
			continue
		}
		if _, err := fmt.Fprintf(out, "%s: %s\n", l[0], l[1]); err != nil {
			return err
		}
	}
	return nil
}

// formatPosition and formatSeconds format a status number, "" when missing.
func formatPosition(pos int) string {
	if pos == 0 {
		return ""
	}
	return strconv.Itoa(pos)
}

func formatSeconds(s *float64) string {
	if s == nil {
		return ""
	}
	return strconv.FormatFloat(*s, 'f', -1, 64)
}
