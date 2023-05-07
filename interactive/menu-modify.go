package interactive

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

var (
	itModCmdInfo      = "Update server info"
	itModCmdMods      = "List installed mods"
	itModCmdAddMod    = "Install mod(s)"
	itModCmdAddColl   = "Install mod(s) from collection"
	itModCmdUpdateMod = "Update mod(s)"
	itModCmdRemoveMod = "Remove mod(s)"
	itModCmdCheckMods = "Check for problems"
	itModCmdExit      = "Finish modifying"
)

var modCmdMap = map[string]func(*cobra.Command, *ini.ServerConfig){
	itModCmdInfo:      cmdUpdateServerInfo,
	itModCmdMods:      cmdListMods,
	itModCmdAddMod:    cmdAddMods,
	itModCmdAddColl:   cmdAddModsFromCollection,
	itModCmdUpdateMod: cmdUpdateMods,
	itModCmdRemoveMod: cmdRemoveMods,
	itModCmdCheckMods: cmdCheckMods,
}

func cmdModifyMenu(cmd *cobra.Command, config *ini.ServerConfig) {
	cont := true
	for cont {
		cont = modifyMenu(cmd, config)
		if cont {
			cmd.Println()
		}
	}
}

func modifyMenu(cmd *cobra.Command, config *ini.ServerConfig) bool {
	var modifyMenu = &survey.Select{
		Message: "What would you like to do?",
		Options: []string{
			itModCmdMods,
			itModCmdAddMod,
			itModCmdAddColl,
			itModCmdUpdateMod,
			itModCmdRemoveMod,
			itModCmdCheckMods,
			itModCmdInfo,
			itModCmdExit,
		},
	}

	var modifyMenuResult string
	err := survey.AskOne(modifyMenu, &modifyMenuResult)
	if err != nil {
		if err == terminal.InterruptErr {
			return false
		}

		cmd.Println(util.Error, err)
		return true
	}

	if modifyMenuResult == itModCmdExit {
		return false
	}

	fn := modCmdMap[modifyMenuResult]
	if fn != nil {
		fn(cmd, config)
	} else {
		cmd.Printf("%s Unknown command: %s\n", util.Error, modifyMenuResult)
	}

	return true
}
