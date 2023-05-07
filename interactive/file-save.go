package interactive

import (
	"fmt"
	"os"
	"path"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func cmdSaveConfig(cmd *cobra.Command, config *ini.ServerConfig) {
	err := config.Save()
	if err != nil {
		fmt.Println(util.Error, err)
		return
	} else {
		fmt.Println(util.OK, "Saved to", config.Path)
	}
}

func cmdSaveConfigTo(cmd *cobra.Command, config *ini.ServerConfig) {
	var configPath string
	prompt := &survey.Input{
		Message: "Path:",
		Default: config.Path,
	}

	survey.AskOne(prompt, &configPath)
	if configPath == "" || configPath == config.Path {
		fmt.Println(util.Warning, "Path not changed")
		return
	}

	if !path.IsAbs(configPath) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println(util.Error, err)
			return
		}

		configPath = path.Join(cwd, configPath)
	}

	if util.IsDir(configPath) {
		fmt.Println(util.Error, "Path is a directory")
		return
	}

	if util.FileExists(configPath) {
		if !ConfirmOverwrite(configPath) {
			fmt.Println(util.Warning, "Not saved")
			return
		}

		fmt.Println(util.Warning, "Overwriting", configPath)
	}

	err := config.SaveTo(configPath)
	if err != nil {
		fmt.Println(util.Error, err)
		return
	} else {
		fmt.Println(util.OK, "Saved to", configPath)
	}
}
