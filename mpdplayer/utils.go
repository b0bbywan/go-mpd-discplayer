package mpdplayer

import (
	"github.com/b0bbywan/go-disc-cuer/utils"
)

func getTrackCount(device string) (int, error) {
	return utils.GetTrackCount(device)
}
