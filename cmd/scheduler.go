package cmd

import (
	"log"

	"github.com/robfig/cron/v3"

	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

type scheduler struct {
	c        *cron.Cron
	schedule []*ScheduleUri
}

type ScheduleUri struct {
	schedule string
	uri      string
	callback func()
	jobId    cron.EntryID
}

func newScheduler(schedulers []*ScheduleUri) *scheduler {
	if len(schedulers) == 0 {
		return nil
	}

	c := cron.New()
	for _, v := range schedulers {
		jobId, err := c.AddFunc(v.schedule, v.callback)
		if err != nil {
			log.Printf("Failed to add %s cron, check syntax: %v", v.schedule, err)
		}
		v.jobId = jobId
		log.Printf("Added schedule: cron='%s' uri='%s'", v.schedule, v.uri)
	}
	s := &scheduler{
		c:        c,
		schedule: schedulers,
	}
	return s
}

func (s *scheduler) Close() {
	if s != nil {
		s.c.Stop()
		s.c = nil
		s.schedule = nil
	}
}

func (p *Player) StopScheduler() {
	if p.scheduler != nil {
		p.scheduler.c.Stop()
	}
}

func (p *Player) StartScheduler() {
	if p.scheduler != nil {
		p.scheduler.c.Start()
	}
}

func newSchedulerUris(
	mpdClient *mpdplayer.ReconnectingMPDClient,
	notifier *notifications.Notifier,
	schedules map[string]string,
) []*ScheduleUri {
	var schedulers []*ScheduleUri
	for k, v := range schedules {
		schedulers = append(schedulers, newSchedulerUri(mpdClient, notifier, k, v))
	}
	return schedulers
}

func newSchedulerUri(
	mpdClient *mpdplayer.ReconnectingMPDClient,
	notifier *notifications.Notifier,
	schedule, uri string,
) *ScheduleUri {
	callback := func() {
		if notifier != nil {
			notifier.PlayEvent(notifications.EventAdd)
		}
		if err := mpdClient.StartPlayback(uri); err != nil {
			if notifier != nil {
				notifier.PlayError()
			}
			log.Printf("Failed to play %s: %v", uri, err)
		}
	}
	return &ScheduleUri{
		schedule: schedule,
		uri:      uri,
		callback: callback,
	}
}
