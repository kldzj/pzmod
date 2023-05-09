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

func cmdUpdateMods(cmd *cobra.Command, config *ini.ServerConfig) {
	cont := true
	for cont {
		err := updateMod(config)
		if err != nil {
			fmt.Println(util.Error, err)
		}

		cont = Continue("updating mods")
	}
}

func updateMod(config *ini.ServerConfig) error {
	modList := getFixedArray(config, util.CfgKeyMods)
	workshopList := getFixedArray(config, util.CfgKeyItems)

	fmt.Println(util.Info, "Fetching workshop items...")
	items, missing, err := steam.FetchWorkshopItems(workshopList)
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
	if err := survey.AskOne(itemPrompt, &id); err != nil {
		return err
	}

	if id == "" || id == itCmdExit {
		return nil
	}

	item := steam.FindItemByID(items, id)
	if item == nil {
		return fmt.Errorf("could not find item")
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
	if err != nil {
		return err
	}

	if len(mods) == 0 {
		return fmt.Errorf("no mods selected")
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
	if err != nil {
		return err
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
		if err != nil {
			return err
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
			if err := survey.AskOne(afterPrompt, &addAfter); err != nil {
				return err
			}

			if addAfter == addEnd {
				mapList = append(mapList, maps...)
			} else if addAfter == addStart {
				mapList = append(maps, mapList...)
			} else {
				index := util.IndexOf(mapList, addAfter)
				if index == -1 {
					return fmt.Errorf("could not find map %s", addAfter)
				}

				mapList = append(mapList[:index+1], append(maps, mapList[index+1:]...)...)
			}

			config.Set(util.CfgKeyMap, strings.Join(util.Dedupe(mapList), ";"))
		}
	}

	config.Set(util.CfgKeyMods, strings.Join(util.Dedupe(modList), ";"))
	fmt.Println(util.OK, "Updated mod", util.Quote(item.Title))
	return nil
}

func mapTitlesToIDs(items *[]steam.WorkshopItem) map[string]string {
	m := make(map[string]string)
	for _, item := range *items {
		m[item.PublishedFileID] = item.Title
	}

	return m
}
