package interactive

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

// cmdAddMods continuously prompts the user to add mods until they choose to stop.
func cmdAddMods(cmd *cobra.Command, config *ini.ServerConfig) {
	cont := true
	for cont {
		cont = addMods(config)
	}
}

const (
	addStart = "Add to the start of the list"
	addEnd   = "Add to the end of the list"
)

// addMods prompts the user to enter a mod's Workshop ID and attempts to add it to the server configuration.
func addMods(config *ini.ServerConfig) bool {
	idPrompt := &survey.Input{
		Message: "Mod Workshop ID:",
	}

	var id string
	err := survey.AskOne(idPrompt, &id)

	// If the user cancels the input or enters an empty ID, stop the process.
	if err == terminal.InterruptErr || id == "" {
		return false
	}

	if err != nil {
		fmt.Println(util.Error, err)
		return true
	}

	// Attempt to add the mod to the configuration.
	cont, err := addMod(id, config)
	if err != nil {
		fmt.Println(util.Error, "Failed to add mod:", err)
		return true
	}

	if !cont {
		return false
	}

	// Ask the user if they want to continue adding mods.
	return Continue("adding mods")
}

// getEnabledMods filters and returns the mods that are already enabled in the mod list.
func getEnabledMods(mods []string, modList []string) []string {
	if len(mods) == 1 {
		return mods
	}

	var enabledMods []string
	for _, mod := range mods {
		if isEnabled(mod, modList) {
			enabledMods = append(enabledMods, mod)
		}
	}

	return enabledMods
}

// getMapList retrieves the list of maps from the server configuration.
func getMapList(config *ini.ServerConfig) []string {
	list := config.GetOrDefault(util.CfgKeyMap, "")
	if list == "" {
		return []string{}
	}

	return strings.Split(list, ";")
}

// checkDependencies fetches and prompts the user to add any dependencies required by the given workshop item.
func checkDependencies(parent *steam.WorkshopItem, config *ini.ServerConfig) error {
	items, missing, err := steam.FetchWorkshopItems(parent.GetChildIDs())
	if err != nil {
		return fmt.Errorf("failed to fetch dependencies for %s: %w", parent.Title, err)
	}

	// If dependencies are found, prompt the user to add them.
	if len(*items) > 0 {
		fmt.Println(util.Info, "Found dependencies:")
		for _, item := range *items {
			fmt.Println(util.Info, "- ", item.Title)
			if Confirm(fmt.Sprintf("Add dependency %s?", util.Quote(item.Title)), true) {
				_, err := addMod(fmt.Sprint(item.PublishedFileID), config)
				if err != nil {
					return fmt.Errorf("failed to add dependency %s: %w", item.Title, err)
				}
			}
		}
	}

	// Warn the user about any missing dependencies.
	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Missing dependencies:")
		for _, id := range *missing {
			fmt.Println(util.Warning, "- ", id)
		}
	}
	return nil
}
