package cmd

import (
	"github.com/spf13/viper"

	"github.com/b0bbywan/go-mpd-discplayer/api"
)

// APIConfig returns the web API settings, nil when the API is disabled.
// The configuration must be loaded first, by NewPlayer.
func APIConfig() *api.Config {
	viper.SetDefault("API.Enabled", false)
	viper.SetDefault("API.Bind", "127.0.0.1")
	viper.SetDefault("API.Port", 8019)
	viper.SetDefault("API.Socket", "")

	if !viper.GetBool("API.Enabled") {
		return nil
	}
	return &api.Config{
		Bind:   viper.GetString("API.Bind"),
		Port:   viper.GetInt("API.Port"),
		Socket: viper.GetString("API.Socket"),
	}
}
