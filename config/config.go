package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/b0bbywan/go-disc-cuer/config"
)

const (
	AppName = "mpd-discplayer"
)

type MPDConn struct {
	Type    string // "unix" or "tcp"
	Address string // socket path or TCP address
}

var (
	MPDConnection MPDConn
)

func init() {
	viper.SetDefault("MPDConnection.Type", "tcp")
	viper.SetDefault("MPDConnection.Address", "127.0.0.1:6600")
	viper.SetDefault("TargetDevice", "sr0")

	// Load from configuration file, environment variables, and CLI flags
	viper.SetConfigName("config")  // name of config file (without extension)
	viper.SetConfigType("yaml")    // config file format
	viper.AddConfigPath(filepath.Join("/etc", config.AppName))  // Global configuration path
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", config.AppName)) // User config path
	}

	// Environment variable support
	viper.SetEnvPrefix(strings.ReplaceAll(AppName, "-", "_")) // environment variables start with MPD_PLAYER
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		// File not found is acceptable, only raise errors for other issues
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}

	// Populate the MPDConnection struct
	MPDConnection = MPDConn{
		Type:    viper.GetString("MPDConnection.Type"),
		Address: viper.GetString("MPDConnection.Address"),
	}
	if err = validateMPDConnection(MPDConnection); err != nil {
		log.Fatalf("Error validating MPD Connection: %v\n", err)
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
