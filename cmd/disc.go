package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
)

func (player *Player) newDiscHandler() {
    discHandler := hwcontrol.NewBasicHandler(
        "disc",
        detect.DeviceDisc,
        // processAdd
        func(ctx context.Context, dev detect.Device) error {
            if err := hwcontrol.SetDiscSpeed(dev.Path(), player.GetDiscSpeed()); err != nil {
                log.Printf("[disc] Error setting disc speed on %s: %v", dev.Path(), err)
            }
            if err := player.Client.StartDiscPlayback(dev.Path()); err != nil {
                return fmt.Errorf("[disc] Error starting %s playback: %w", dev.Path(), err)
            }
            return nil
        },
        // processRemove
        func(ctx context.Context, dev detect.Device) error {
            if err := player.Client.StopDiscPlayback(); err != nil {
                return fmt.Errorf("[disc] Error stopping %s playback: %w", dev.Path(), err)
            }
            return nil
        },
    )

    // Stocker le handler dans Player
    player.handlers = append(player.handlers, discHandler)
}
