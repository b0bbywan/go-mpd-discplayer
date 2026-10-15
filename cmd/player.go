package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jochenvg/go-udev"
	"github.com/spf13/viper"

	"github.com/b0bbywan/go-disc-cuer/config"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/detect"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/mounts"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

const (
	AppName          = "mpd-discplayer"
	defaultMpdFolder = "/var/lib/mpd/music"
)

var AppVersion = "dev"

// playerClient is the part of the MPD client the player drives.
type playerClient interface {
	StartDiscPlayback(device string) error
	StartUSBPlayback(label string) error
	StartPlayback(uri string) error
	Stop() error
	StopIfPlaying(prefix string) error
	StopAndRemove(prefix string) error
	UpdateDB(uri string) error
	Disconnect()
}

// deviceMounter exposes USB devices to MPD.
type deviceMounter interface {
	Mount(device *udev.Device) (string, error)
	Unmount(device *udev.Device) (string, error)
}

type Player struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        *sync.WaitGroup
	discSpeed int
	Client    playerClient
	Notifier  *notifications.Notifier
	Mounter   deviceMounter
	scheduler *scheduler
	handlers  []Handler
}

func NewPlayer(ctx context.Context, cancel context.CancelFunc) (*Player, error) {
	viper.SetDefault("MPDConnection.Type", "tcp")
	viper.SetDefault("MPDConnection.Address", "127.0.0.1:6600")
	viper.SetDefault("MPDConnection.ReconnectWait", 30)
	viper.SetDefault("MPDLibraryFolder", defaultMpdFolder)
	viper.SetDefault("MPDCueSubfolder", ".disc-cuer")
	viper.SetDefault("MPDUSBSubfolder", ".udisks")
	viper.SetDefault("DiscSpeed", 12)
	viper.SetDefault("SoundsLocation", filepath.Join("/usr/local/share/", AppName))
	viper.SetDefault("AudioBackend", "pulse")
	viper.SetDefault("PulseServer", "")
	viper.SetDefault("MountConfig", "mpd")
	viper.SetDefault("Schedule", make(map[string]string))

	// Load from configuration file, environment variables, and CLI flags
	viper.SetConfigName("config")                       // name of config file (without extension)
	viper.SetConfigType("yaml")                         // config file format
	viper.AddConfigPath(filepath.Join("/etc", AppName)) // Global configuration path
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", AppName)) // User config path
	}

	// Environment variable support
	viper.SetEnvPrefix(strings.ReplaceAll(AppName, "-", "_")) // environment variables start with MPD_PLAYER
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		// File not found is acceptable, only raise errors for other issues
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}
	var wg sync.WaitGroup

	mpdConnection, err := mpdplayer.NewMPDConnection(
		viper.GetString("MPDConnection.Type"),
		viper.GetString("MPDConnection.Address"),
		time.Duration(viper.GetInt("MPDConnection.ReconnectWait")*int(time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("error validating MPD connection: %w", err)
	}
	mpdClient := mpdplayer.NewReconnectingMPDClient(ctx, mpdConnection)
	if err = setMpdFolder(mpdClient); err != nil {
		log.Printf("warning: %v", err)
	}

	cuerCacheLocation := filepath.Join(
		viper.GetString("MPDLibraryFolder"),
		viper.GetString("MPDCueSubfolder"),
	)
	cuerConfig, err := config.NewConfig(AppName, AppVersion, cuerCacheLocation)
	if err != nil {
		log.Printf("Failed to create Cuer Config: %v", err)
	}
	mpdClient.SetCuerConfig(cuerConfig)

	mountConfig := mounts.NewMountConfig(
		viper.GetString("MPDLibraryFolder"),
		viper.GetString("MPDUSBSubfolder"),
		viper.GetString("MountConfig"),
	)
	mounter, err := mounts.NewMountManager(mountConfig, mpdClient)
	if err != nil {
		return nil, fmt.Errorf("USB Playback disabled: Failed to create mount manager: %w", err)
	}

	notificationConfig := notifications.NewNotificationConfig(
		viper.GetString("AudioBackend"),
		viper.GetString("PulseServer"),
		viper.GetString("SoundsLocation"),
	)
	notifier := notifications.NewNotifier(notificationConfig)

	schedules := newSchedulerUris(mpdClient, notifier, viper.GetStringMapString("Schedule"))
	scheduler := newScheduler(schedules)

	return &Player{
		ctx:       ctx,
		cancel:    cancel,
		wg:        &wg,
		discSpeed: viper.GetInt("DiscSpeed"),
		Client:    mpdClient,
		Notifier:  notifier,
		Mounter:   mounter,
		scheduler: scheduler,
	}, nil
}

func (p *Player) Start() {
	p.StartScheduler()

	p.newDiscHandler()
	p.newUSBHandler()

	detector := detect.NewUdevDetector()
	events := make(chan detect.DeviceEvent)

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer close(events)
		if err := detector.Run(p.ctx, events); err != nil {
			log.Printf("Failed to run udev detector: %s", err)
			p.cancel()
		}
	}()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.run(events)
	}()
}

func (p *Player) run(events <-chan detect.DeviceEvent) {
	for {
		select {
		case <-p.ctx.Done():
			log.Println("ending dispatch")
			return
		case ev, ok := <-events:
			if !ok {
				log.Println("event channel closed, exiting")
				return
			}
			p.dispatch(ev)
		}
	}
}

func (p *Player) dispatch(ev detect.DeviceEvent) {
	var err error
	for _, h := range p.handlers {
		if !h.Handles(ev.Device.Kind()) {
			continue
		}
		switch ev.Type {
		case detect.DeviceAdded:
			p.NotifyEvent(notifications.EventAdd)
			err = h.OnAdd(p.ctx, ev.Device)
		case detect.DeviceRemoved:
			p.NotifyEvent(notifications.EventRemove)
			err = h.OnRemove(p.ctx, ev.Device)
		case detect.DevicePresent:
			err = h.OnPresent(p.ctx, ev.Device)
		}
		if err != nil {
			p.NotifyEvent(notifications.EventError)
			log.Printf("[dispatcher] Error during callback execution %s %s: %s", ev.Type, ev.Device.Kind(), err)
		}
	}
}

func (p *Player) Close() {
	p.cancel()
	if p.Client != nil {
		p.Client.Disconnect()
	}
	if p.Notifier != nil {
		p.Notifier.Close()
	}
	if p.scheduler != nil {
		p.scheduler.Close()
	}
	p.wg.Wait()
}

// PlayDisc replaces the queue with the disc in drive and starts playback.
// An empty drive stands for the drive holding an audio disc.
func (p *Player) PlayDisc(drive string) error {
	if drive == "" {
		var err error
		if drive, err = detect.FindAudioDisc(); err != nil {
			return err
		}
	}
	if err := hwcontrol.SetDiscSpeed(drive, p.discSpeed); err != nil {
		log.Printf("warning: failed to set disc speed on %s: %v", drive, err)
	}
	if err := p.Client.StartDiscPlayback(drive); err != nil {
		return fmt.Errorf("error adding tracks: %w", err)
	}
	return nil
}

// PlayURL replaces the queue with url, a webradio for instance, and starts playback.
func (p *Player) PlayURL(url string) error {
	if err := p.Client.StartPlayback(url); err != nil {
		return fmt.Errorf("error playing %s: %w", url, err)
	}
	return nil
}

// Stop stops playback, whatever is playing. The queue is left untouched.
func (p *Player) Stop() error {
	if err := p.Client.Stop(); err != nil {
		return fmt.Errorf("error stopping playback: %w", err)
	}
	return nil
}

// StopDisc stops playback when a disc track is playing. The queue is left untouched.
func (p *Player) StopDisc() error {
	if err := p.Client.StopIfPlaying(mpdplayer.CDDAPathPrefix); err != nil {
		return fmt.Errorf("error stopping disc playback: %w", err)
	}
	return nil
}

// EjectDisc stops disc playback and ejects the disc in drive. An empty
// drive stands for the drive holding a disc.
func (p *Player) EjectDisc(drive string) error {
	if drive == "" {
		var err error
		if drive, err = detect.FindDisc(); err != nil {
			return err
		}
	}
	// Stop first so MPD is not left reading a disc that is going away
	if err := p.Client.StopAndRemove(mpdplayer.CDDAPathPrefix); err != nil {
		log.Printf("warning: failed to stop disc playback before eject: %v", err)
	}
	if err := hwcontrol.EjectDisc(drive); err != nil {
		return fmt.Errorf("error ejecting %s: %w", drive, err)
	}
	return nil
}

func (p *Player) NotifyEvent(name string) {
	if p.Notifier != nil {
		p.Notifier.PlayEvent(name)
	}
}

func setMpdFolder(mpdClient *mpdplayer.ReconnectingMPDClient) error {
	if viper.GetString("MPDLibraryFolder") == defaultMpdFolder {
		musicDir, err := mpdClient.GetConfig()
		if err != nil {
			return fmt.Errorf("warning: Couldn't get music_directory from mpd: %w", err)
		}
		viper.Set("MPDLibraryFolder", musicDir)
		log.Printf("Overriding MPDLibraryFolder with music directory from MPD: %s", musicDir)
	}
	return nil
}
