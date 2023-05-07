package interactive

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/kldzj/pzmod/config"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

var (
	itCmdModify    = "Modify server config"
	itCmdSave      = "Save server config"
	itCmdSaveTo    = "Copy server config to"
	itCmdSetApiKey = "Set API Key"
	itCmdExit      = "Exit"
)

var cmdMap = map[string]func(*cobra.Command, *ini.ServerConfig){
	itCmdModify:    cmdModifyMenu,
	itCmdSave:      cmdSaveConfig,
	itCmdSaveTo:    cmdSaveConfigTo,
	itCmdSetApiKey: cmdSetApiKey,
}

func Execute(cmd *cobra.Command, args []string) {
	config, err := config.LoadConfig(cmd)
	if err != nil {
		fmt.Println(util.Error, err)
		return
	}

	cont := true
	for cont {
		cont = mainMenu(cmd, config)
		if cont {
			fmt.Println()
		}
	}
}

func mainMenu(cmd *cobra.Command, config *ini.ServerConfig) bool {
	var mainMenu = &survey.Select{
		Message: "What would you like to do?",
		Options: []string{
			itCmdModify,
			itCmdSave,
			itCmdSaveTo,
			itCmdSetApiKey,
			itCmdExit,
		},
	}

	var mainMenuResult string
	err := survey.AskOne(mainMenu, &mainMenuResult)
	if err != nil {
		if err == terminal.InterruptErr {
			return cmdExit(cmd, config)
		}

		fmt.Println(util.Error, err)
		return true
	}

	if mainMenuResult == itCmdExit {
		return cmdExit(cmd, config)
	}

	fn := cmdMap[mainMenuResult]
	if fn != nil {
		fn(cmd, config)
	} else {
		fmt.Printf("%s Unknown command: %s\n", util.Error, mainMenuResult)
	}

	return true
}

func cmdSetApiKey(cmd *cobra.Command, config *ini.ServerConfig) {
	var apiKey string
	prompt := &survey.Password{
		Message: "API Key:",
		Help:    "Get your API key from https://steamcommunity.com/dev/apikey",
	}

	err := survey.AskOne(prompt, &apiKey)
	if err != nil {
		fmt.Println(util.Error, err)
		return
	}

	if len(apiKey) != 32 {
		fmt.Println(util.Warning, "Invalid API key.")
		return
	}

	err = util.StoreCredentials(apiKey)
	if err != nil {
		fmt.Println(util.Warning, "Failed to store API key.")
	}

	steam.SetApiKey(apiKey)
}

func cmdExit(cmd *cobra.Command, config *ini.ServerConfig) bool {
	hasChanged := config.HasUnsavedChanges()
	if hasChanged {
		fmt.Println(util.Warning, "You have unsaved changes.")
	}

	var confirmExit = &survey.Confirm{
		Message: "Are you sure you want to exit?",
		Default: !hasChanged,
	}

	var confirmExitResult bool
	survey.AskOne(confirmExit, &confirmExitResult)

	return !confirmExitResult
}
