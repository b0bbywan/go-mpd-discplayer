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
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/mounts"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

const (
	AppName          = "mpd-discplayer"
	AppVersion       = "0.7"
	defaultMpdFolder = "/var/lib/mpd/music"
)

type Player struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        *sync.WaitGroup
	discSpeed int
	Client    *mpdplayer.ReconnectingMPDClient
	Notifier  *notifications.Notifier
	Mounter   *mounts.MountManager
	handlers  []*hwcontrol.EventHandler
	scheduler *scheduler
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
			return nil, fmt.Errorf("Error reading config file: %w", err)
		}
	}
	var wg sync.WaitGroup

	mpdConnection, err := mpdplayer.NewMPDConnection(
		viper.GetString("MPDConnection.Type"),
		viper.GetString("MPDConnection.Address"),
		time.Duration(viper.GetInt("MPDConnection.ReconnectWait")*int(time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("Error validating MPD Connection: %w", err)
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

	// Create event handlers (subscribers) passing the context
	p.newDiscHandlers()
	p.newUSBHandlers()

	for _, handler := range p.handlers {
		handler.StartSubscriber(p.wg, p.ctx)
	}
	p.wg.Add(1)
	go p.run()
}

func (p *Player) run() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			log.Println("Stopping from cmd.")
			return
		default:
			if err := hwcontrol.StartMonitor(p.ctx, p.handlers); err != nil {
				log.Printf("Error starting monitor: %w\n", err)
				time.Sleep(time.Second) // Retry after some delay
				continue
			}
		}
	}
}

func (p *Player) SetHandlerProcessor(
	handler *hwcontrol.EventHandler,
	callback func(device *udev.Device) error,
	logMessage, eventName string,
) {
	handler.SetProcessor(
		p.wg,
		logMessage,
		callback,
		p.Notifier,
		eventName,
	)
	p.AddHandler(handler)
}

func (p *Player) AddHandler(handlers ...*hwcontrol.EventHandler) {
	p.handlers = append(p.handlers, handlers...)
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

func (p *Player) GetDiscSpeed() int {
	return p.discSpeed
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
