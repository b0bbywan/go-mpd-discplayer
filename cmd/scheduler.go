package cmd

import (
	"log"

	"github.com/robfig/cron/v3"

	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
)

type scheduler struct {
	c        *cron.Cron
	schedule map[string]string
}

func newScheduler(client *mpdplayer.ReconnectingMPDClient, schedule map[string]string) *scheduler {
	if len(schedule) == 0 {
		return nil
	}

	c := cron.New()
	for k, v := range schedule {
		if _, err := c.AddFunc(k, func() {
			if err := client.StartPlayback(v); err != nil {
				log.Printf("Failed to play %s", k)
			}
		}); err != nil {
			log.Printf("Failed to add %s cron, check syntax: %w", k, err)
		}
		log.Printf("Added schedule: cron='%s', uri='%s'", k, v)
	}
	s := &scheduler{
		c:        c,
		schedule: schedule,
	}
	s.c.Start()
	return s
}

func (p *Player) StopScheduler() {
	if p.scheduler != nil {
		p.scheduler.c.Stop()
	}
}

func (p *Player) ResumeScheduler() {
	if p.scheduler != nil {
		p.scheduler.c.Start()
	}
}
