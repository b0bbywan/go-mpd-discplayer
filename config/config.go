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
)

const (
	AppName    = "mpd-discplayer"
	AppVersion = "0.4"
)

type MPDConn struct {
	Type          string // "unix" or "tcp"
	Address       string // socket path or TCP address
	ReconnectWait time.Duration
}

var (
	MPDConnection    MPDConn
	MPDLibraryFolder string
	TargetDevice     string
	DiscSpeed        int
	SoundsLocation   string
	CuerConfig       *config.Config
)

func init() {
	viper.SetDefault("MPDConnection.Type", "tcp")
	viper.SetDefault("MPDConnection.Address", "127.0.0.1:6600")
	viper.SetDefault("MPDConnection.ReconnectWait", 30)
	viper.SetDefault("MPDLibraryFolder", "/var/lib/mpd/music")
	viper.SetDefault("DiscSpeed", 12)
	viper.SetDefault("SoundsLocation", filepath.Join("/usr/local/share/", AppName))

	// Load from configuration file, environment variables, and CLI flags
	viper.SetConfigName("config")                              // name of config file (without extension)
	viper.SetConfigType("yaml")                                // config file format
	viper.AddConfigPath(filepath.Join("/etc", config.AppName)) // Global configuration path
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
			fmt.Fprintf(os.Stderr, "Error reading config file: %w", err)
			os.Exit(1)
		}
	}

	DiscSpeed = viper.GetInt("DiscSpeed")
	SoundsLocation = viper.GetString("SoundsLocation")
	// Populate the MPDConnection struct
	MPDConnection = MPDConn{
		Type:          viper.GetString("MPDConnection.Type"),
		Address:       viper.GetString("MPDConnection.Address"),
		ReconnectWait: time.Duration(viper.GetInt("MPDConnection.ReconnectWait") * int(time.Second)),
	}
	if err = validateMPDConnection(MPDConnection); err != nil {
		log.Fatalf("Error validating MPD Connection: %w", err)
	}
	MPDLibraryFolder = viper.GetString("MPDLibraryFolder")
	CuerConfig, err = config.NewConfig(AppName, AppVersion, filepath.Join(MPDLibraryFolder, ".disc-cuer"))
	if err != nil {
		log.Fatalf("Failed to create disc-cuer config: %v", err)
	}
}

// validateMPDConnection checks the validity of the MPD connection settings
func validateMPDConnection(conn MPDConn) error {
	if conn.Type != "unix" && conn.Type != "tcp" {
		return fmt.Errorf("invalid MPDConnection.Type: %s, must be 'unix' or 'tcp'", conn.Type)
	}
	if conn.Address == "" {
		return fmt.Errorf("MPDConnection.Address cannot be empty")
	}
	return nil
}
