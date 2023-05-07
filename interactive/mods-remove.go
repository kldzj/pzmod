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

func cmdRemoveMods(cmd *cobra.Command, config *ini.ServerConfig) {
	cont := true
	for cont {
		cont = removeMod(config)
		fmt.Println()
	}
}

func removeMod(config *ini.ServerConfig) bool {
	modList := getFixedArray(config, util.CfgKeyMods)
	itemList := getFixedArray(config, util.CfgKeyItems)

	if len(itemList) == 0 {
		fmt.Println(util.Warning, "No mods found")
		return false
	}

	fmt.Println(util.Info, "Fetching workshop items...")
	items, missing, err := steam.FetchWorkshopItems(itemList)
	if err != nil {
		fmt.Println(util.Error, err)
		return true
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch the following items:")
		for _, id := range *missing {
			fmt.Println(util.Warning, " -", id)
		}
	}

	if len(*items) == 0 {
		fmt.Println(util.Warning, "No items found")
		return true
	}

	titles := mapTitlesToIDs(items)
	options := []string{itCmdExit}
	options = append(options, itemList...)
	itemPrompt := &survey.Select{
		Message: "Select mod to remove:",
		Options: options,
		Description: func(value string, index int) string {
			if value == itCmdExit {
				return "Stop removing mods"
			}

			if title, ok := titles[value]; ok {
				return title
			}

			return value
		},
	}

	var id string
	err = survey.AskOne(itemPrompt, &id)
	if err == terminal.InterruptErr || id == "" {
		return false
	}

	if err != nil {
		fmt.Println(util.Error, err)
		return true
	}

	if id == itCmdExit {
		return false
	}

	item := steam.FindItemByID(items, id)
	if item == nil {
		fmt.Println(util.Error, "Could not find item")
		return true
	}

	if !Confirm("Are you sure you want to remove "+util.Quote(item.Title)+"?", true) {
		fmt.Println(util.Warning, "Aborted")
		return true
	}

	fmt.Println(util.Info, "Removing", util.Quote(item.Title), "...")
	parsed := item.Parse()
	modList = util.Filter(modList, func(s string) bool {
		return !util.Contains(parsed.Mods, s)
	})

	itemList = util.Filter(itemList, func(s string) bool {
		return s != id
	})

	mapList := getMapList(config)
	mapList = util.Filter(mapList, func(s string) bool {
		return !util.Contains(parsed.Maps, s)
	})

	config.Set(util.CfgKeyMods, strings.Join(modList, ";"))
	config.Set(util.CfgKeyItems, strings.Join(itemList, ";"))
	config.Set(util.CfgKeyMap, strings.Join(mapList, ";"))

	return Continue("removing mods")
}
