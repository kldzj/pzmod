package interactive

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func cmdRemoveMods(cmd *cobra.Command, config *ini.ServerConfig) {
	cont := true
	for cont {
		err := removeMod(config)
		if err != nil {
			fmt.Println(util.Error, err)
		}

		cont = Continue("removing mods")
	}
}

func removeMod(config *ini.ServerConfig) error {
	modList := getFixedArray(config, util.CfgKeyMods)
	itemList := getFixedArray(config, util.CfgKeyItems)

	if len(itemList) == 0 {
		return fmt.Errorf("no items found")
	}

	fmt.Println(util.Info, "Fetching workshop items...")
	items, missing, err := steam.FetchWorkshopItems(itemList)
	if err != nil {
		return err
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch the following items:")
		for _, id := range *missing {
			fmt.Println(util.Warning, " -", id)
		}
	}

	if len(*items) == 0 {
		return fmt.Errorf("no items found")
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
	if err != nil {
		return err
	}

	if id == "" || id == itCmdExit {
		return nil
	}

	item := steam.FindItemByID(items, id)
	if item == nil {
		return fmt.Errorf("could not find item with id %s", id)
	}

	if !Confirm("Are you sure you want to remove "+util.Quote(item.Title)+"?", true) {
		return fmt.Errorf("removal aborted")
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

	return nil
}

func mapTitlesToIDs(items *[]steam.WorkshopItem) map[string]string {
	m := make(map[string]string)
	for _, item := range *items {
		m[item.PublishedFileID] = item.Title
	}

	return m
}
