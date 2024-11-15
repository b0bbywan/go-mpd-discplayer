package disc

import (
	"go.uploadedlobster.com/discid"
)

func GetTrackCount() (int, error) {
	disc, err := discid.Read("")
	if err != nil {
		return 0, err
	}
	defer disc.Close()

	return disc.LastTrackNumber(), nil
}
