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

func addMods(config *ini.ServerConfig) bool {
	idPrompt := &survey.Input{
		Message: "Mod Workshop ID:",
	}

	var id string
	err := survey.AskOne(idPrompt, &id)
	if err == terminal.InterruptErr || id == "" {
		return false
	}

	if err != nil {
		fmt.Println(util.Error, err)
		return true
	}

	cont, _ := addMod(id, config)
	if !cont {
		return false
	}

	return Continue("adding mods")
}

func addMod(id string, config *ini.ServerConfig) (bool, error) {
	items, missing, err := steam.FetchWorkshopItems([]string{id})
	if err != nil {
		return false, err
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch", (*missing)[0])
	}

	if len(*items) == 0 {
		return false, fmt.Errorf("no items found")
	}

	item := (*items)[0]
	if item.FileType != steam.FileTypeMod {
		return false, fmt.Errorf("workshop item is not a mod")
	}

	parsed := item.Parse()
	if len(parsed.Mods) == 0 {
		return false, fmt.Errorf("parsed item has no mods")
	}

	modList := getFixedArray(config, util.CfgKeyMods)
	itemList := getFixedArray(config, util.CfgKeyItems)
	mapList := getMapList(config)

	modsPrompt := &survey.MultiSelect{
		Message: "Select mods to add:",
		Options: parsed.Mods,
		Default: getEnabledMods(parsed.Mods, modList),
	}

	var mods []string
	err = survey.AskOne(modsPrompt, &mods)
	if err != nil {
		return false, err
	}

	if len(mods) == 0 {
		return false, fmt.Errorf("no mods selected")
	}

	options := []string{addStart, addEnd}
	options = append(options, modList...)
	afterPrompt := &survey.Select{
		Message: "Add after:",
		Options: options,
		Default: addEnd,
	}

	var addAfter string
	err = survey.AskOne(afterPrompt, &addAfter)
	if err != nil {
		return false, err
	}

	if addAfter == addEnd {
		modList = append(modList, mods...)
	} else if addAfter == addStart {
		modList = append(mods, modList...)
	} else {
		index := util.IndexOf(modList, addAfter)
		if index == -1 {
			return false, fmt.Errorf("could not find mod %s", addAfter)
		}

		modList = append(modList[:index+1], append(mods, modList[index+1:]...)...)
	}

	if len(parsed.Maps) > 0 {
		mapsPrompt := &survey.MultiSelect{
			Message: "Select maps to add:",
			Options: parsed.Maps,
			Default: getEnabledMods(parsed.Maps, mapList),
		}

		var maps []string
		err = survey.AskOne(mapsPrompt, &maps)
		if err != nil {
			return false, err
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
			if err != nil {
				return false, err
			}

			if addAfter == addEnd {
				mapList = append(mapList, maps...)
			} else if addAfter == addStart {
				mapList = append(maps, mapList...)
			} else {
				index := util.IndexOf(mapList, addAfter)
				if index == -1 {
					return false, fmt.Errorf("could not find map %s", addAfter)
				}

				mapList = append(mapList[:index+1], append(maps, mapList[index+1:]...)...)
			}

			config.Set(util.CfgKeyMap, strings.Join(util.Dedupe(mapList), ";"))
		}
	}

	itemList = append(itemList, id)

	config.Set(util.CfgKeyItems, strings.Join(util.Dedupe(itemList), ";"))
	config.Set(util.CfgKeyMods, strings.Join(util.Dedupe(modList), ";"))

	checkDependencies(&item, config)
	fmt.Println(util.OK, "Added", util.Quote(item.Title))
	return true, nil
}

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

func getMapList(config *ini.ServerConfig) []string {
	list := config.GetOrDefault(util.CfgKeyMap, "")
	if list == "" {
		return []string{}
	}

	return strings.Split(list, ";")
}

func checkDependencies(parent *steam.WorkshopItem, config *ini.ServerConfig) {
	items, missing, err := steam.FetchWorkshopItems(parent.GetChildIDs())
	if err != nil {
		fmt.Println(util.Error, err)
		return
	}

	itemList := getFixedArray(config, util.CfgKeyItems)
	for _, id := range *missing {
		if !util.Contains(itemList, id) {
			fmt.Println(util.Warning, "Missing dependency that could not be fetched", util.Paren(id))
		}
	}

	notInstalled := []string{}
	for _, item := range *items {
		if !util.Contains(*missing, item.PublishedFileID) && !util.Contains(itemList, item.PublishedFileID) {
			notInstalled = append(notInstalled, item.PublishedFileID)
		}
	}

	if len(notInstalled) == 0 {
		return
	}

	fmt.Println(util.Warning, "Found", len(notInstalled), "missing dependencies")
	for _, id := range notInstalled {
		item := steam.FindItemByID(items, id)
		if item == nil {
			fmt.Println("  -", id)
		} else {
			fmt.Println("  -", item.Title, util.Paren(item.PublishedFileID))
		}
	}

	fmt.Println()
	if !Confirm("Install missing dependencies", true) {
		return
	}

	for _, id := range notInstalled {
		item := steam.FindItemByID(items, id)
		if item == nil {
			fmt.Println(util.Warning, "Could not find dependency", util.Paren(id))
			continue
		}

		fmt.Println(
			util.Warning, "Missing dependency", util.Quote(item.Title), util.Paren(item.PublishedFileID)+",",
			"required by", util.Quote(parent.Title), util.Paren(parent.PublishedFileID),
		)

		fmt.Println(util.Info, "Press Ctrl+C to skip adding this dependency")
		added, err := addMod(item.PublishedFileID, config)
		if err != nil {
			fmt.Println(util.Error, err)
			if !Continue("adding dependencies") {
				break
			}
		}

		if !added {
			fmt.Println(util.Warning, "Did not add dependency", util.Quote(item.Title))
			continue
		}
	}
}
