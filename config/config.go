package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/b0bbywan/go-disc-cuer/config"
	"github.com/b0bbywan/go-mpd-discplayer/hwcontrol/mounts"
	"github.com/b0bbywan/go-mpd-discplayer/mpdplayer"
	"github.com/b0bbywan/go-mpd-discplayer/notifications"
)

const (
	AppName    = "mpd-discplayer"
	AppVersion = "0.4"
)

type PlayerConfig struct {
	MPDConnection      *mpdplayer.MPDConn
	NotificationConfig *notifications.NotificationConfig
	MountConfig        *mounts.MountConfig
	MPDLibraryFolder   string
	TargetDevice       string
	DiscSpeed          int
	CuerCacheLocation  string
}

func NewPlayerConfig() (*PlayerConfig, error) {
	viper.SetDefault("MPDConnection.Type", "tcp")
	viper.SetDefault("MPDConnection.Address", "127.0.0.1:6600")
	viper.SetDefault("MPDConnection.ReconnectWait", 30)
	viper.SetDefault("MPDLibraryFolder", "/var/lib/mpd/music")
	viper.SetDefault("MPDCueSubfolder", ".disc-cuer")
	viper.SetDefault("MPDUSBSubfolder", ".")
	viper.SetDefault("DiscSpeed", 12)
	viper.SetDefault("SoundsLocation", filepath.Join("/usr/local/share/", AppName))
	viper.SetDefault("AudioBackend", "pulse")
	viper.SetDefault("PulseServer", "")
	viper.SetDefault("MountConfig", "mpd")

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

	cuerCacheLocation := filepath.Join(
		viper.GetString("MPDLibraryFolder"),
		viper.GetString("MPDCueSubfolder"),
	)
	cuerConfig, err := config.NewConfig(AppName, AppVersion, cuerCacheLocation)
	if err != nil {
		log.Printf("Failed to create Cuer Config: %v", err)
	}

	mpdConnection, err := mpdplayer.NewMPDConnection(
		viper.GetString("MPDConnection.Type"),
		viper.GetString("MPDConnection.Address"),
		time.Duration(viper.GetInt("MPDConnection.ReconnectWait")*int(time.Second)),
		cuerConfig,
		viper.GetInt("DiscSpeed"),
	)
	if err != nil {
		return nil, fmt.Errorf("Error validating MPD Connection: %w", err)
	}

	mountConfig := mounts.NewMountConfig(
		viper.GetString("MPDLibraryFolder"),
		viper.GetString("MPDUSBSubfolder"),
		viper.GetString("MountConfig"),
	)

	notificationConfig := notifications.NewNotificationConfig(
		viper.GetString("AudioBackend"),
		viper.GetString("PulseServer"),
		viper.GetString("SoundsLocation"),
	)

	return &PlayerConfig{
		NotificationConfig: notificationConfig,
		MPDConnection:      mpdConnection,
		MountConfig:        mountConfig,
		MPDLibraryFolder:   viper.GetString("MPDLibraryFolder"),
		CuerCacheLocation:  cuerCacheLocation,
	}, nil
}
