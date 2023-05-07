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

func cmdUpdateMods(cmd *cobra.Command, config *ini.ServerConfig) {
	cont := true
	for cont {
		cont = updateMod(config)
		fmt.Println()
	}
}

func updateMod(config *ini.ServerConfig) bool {
	modList := getFixedArray(config, util.CfgKeyMods)
	workshopList := getFixedArray(config, util.CfgKeyItems)

	fmt.Println(util.Info, "Fetching workshop items...")
	items, missing, err := steam.FetchWorkshopItems(workshopList)
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
	options = append(options, workshopList...)
	itemPrompt := &survey.Select{
		Message: "Select mod to update:",
		Options: options,
		Description: func(value string, index int) string {
			if value == itCmdExit {
				return "Stop updating mods"
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

	parsed := item.Parse()
	enabled := getEnabledMods(parsed.Mods, modList)
	modsPrompt := &survey.MultiSelect{
		Message: "Select mods to add:",
		Options: parsed.Mods,
		Default: enabled,
	}

	var mods []string
	err = survey.AskOne(modsPrompt, &mods)
	if err == terminal.InterruptErr {
		return false
	}

	if err != nil {
		fmt.Println(util.Error, err)
		return true
	}

	if len(mods) == 0 {
		fmt.Println(util.Warning, "No mods selected")
		return true
	}

	options = []string{addStart, addEnd}
	options = append(options, modList...)
	afterPrompt := &survey.Select{
		Message: "Add after:",
		Options: options,
		Default: addEnd,
	}

	var addAfter string
	err = survey.AskOne(afterPrompt, &addAfter)
	if err == terminal.InterruptErr {
		return false
	}

	if err != nil {
		fmt.Println(util.Error, err)
		return true
	}

	mods = util.Filter(mods, func(mod string) bool {
		return !util.Contains(enabled, mod)
	})

	if addAfter == addStart {
		modList = append(mods, modList...)
	} else if addAfter == addEnd {
		modList = append(modList, mods...)
	} else {
		index := util.IndexOf(modList, addAfter)
		modList = append(modList[:index+1], append(mods, modList[index+1:]...)...)
	}

	if len(parsed.Maps) > 0 {
		mapList := getMapList(config)
		mapsPrompt := &survey.MultiSelect{
			Message: "Select maps to add:",
			Options: parsed.Maps,
			Default: getEnabledMods(parsed.Maps, mapList),
		}

		var maps []string
		err = survey.AskOne(mapsPrompt, &maps)
		if err == terminal.InterruptErr {
			return false
		}

		if err != nil {
			fmt.Println(util.Error, err)
			return true
		}

		if len(maps) == 0 {
			fmt.Println(util.Warning, "No maps selected")
		} else {
			options = []string{addStart, addEnd}
			options = append(options, mapList...)

			afterPrompt = &survey.Select{
				Message: "Add after:",
				Options: options,
				Default: addStart,
			}

			var addAfter string
			err = survey.AskOne(afterPrompt, &addAfter)
			if err == terminal.InterruptErr {
				return false
			}

			if err != nil {
				fmt.Println(util.Error, err)
				return true
			}

			if addAfter == addEnd {
				mapList = append(mapList, maps...)
			} else if addAfter == addStart {
				mapList = append(maps, mapList...)
			} else {
				index := util.IndexOf(mapList, addAfter)
				if index == -1 {
					fmt.Println(util.Warning, "Could not find map", addAfter)
					return true
				}

				mapList = append(mapList[:index+1], append(maps, mapList[index+1:]...)...)
			}

			config.Set(util.CfgKeyMap, strings.Join(util.Dedupe(mapList), ";"))
		}
	}

	config.Set(util.CfgKeyMods, strings.Join(util.Dedupe(modList), ";"))
	fmt.Println(util.OK, "Updated mod", util.Quote(item.Title))
	return Continue("updating mods")
}

func mapTitlesToIDs(items *[]steam.WorkshopItem) map[string]string {
	m := make(map[string]string)
	for _, item := range *items {
		m[item.PublishedFileID] = item.Title
	}

	return m
}
