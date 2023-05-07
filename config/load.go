package config

import (
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func LoadConfig(cmd *cobra.Command) (*ini.ServerConfig, error) {
	apiKey, err := util.LoadCredentials()
	if err != nil {
		cmd.Println(util.Warning, "Steam API key not found")
	} else {
		steam.SetApiKey(apiKey)
	}

	configPath := cmd.Flag("file").Value.String()
	return ini.LoadNewServerConfig(configPath)
}

func UnsafeLoadConfig(cmd *cobra.Command) *ini.ServerConfig {
	config, err := LoadConfig(cmd)
	cobra.CheckErr(err)
	return config
}
