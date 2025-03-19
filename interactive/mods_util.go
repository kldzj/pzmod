package interactive

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
)

// addMod fetches a workshop mod by its ID and adds it to the server configuration.
// It allows the user to manually enter Mod IDs if parsing fails.
func addMod(id string, config *ini.ServerConfig) (bool, error) {
	items, missing, err := steam.FetchWorkshopItems([]string{id})
	if err != nil {
		return false, err
	}

	// Warn if the mod could not be fetched
	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch", (*missing)[0])
	}

	// If no valid items were fetched, return an error
	if len(*items) == 0 {
		return false, fmt.Errorf("no items found")
	}

	item := (*items)[0]

	// Ensure the workshop item is a mod
	if item.FileType != steam.FileTypeMod {
		return false, fmt.Errorf("workshop item is not a mod")
	}

	fmt.Println(util.Info, "Adding", util.Quote(item.Title))
	fmt.Println(util.Info, "Mod size:", util.HumanizeBytes(uint64(item.FileSize)), "", item.GetWorkshopLink())

	// Parse the item to extract mod IDs
	parsed := item.Parse()
	if len(parsed.Mods) == 0 {
		fmt.Println(util.Warning, "Could not parse Mod ID(s) from item:", util.Quote(item.Title))

		// Allow manual entry of Mod IDs if parsing fails
		if Confirm("Would you like to enter the Mod ID manually?", true) {
			for {
				var mod string
				err := survey.AskOne(&survey.Input{
					Message: "Mod name:",
					Help:    "Enter a single Mod ID, or leave blank to finish.",
				}, &mod)

				if err != nil {
					return false, err
				}

				if mod == "" {
					if len(parsed.Mods) == 0 {
						return false, fmt.Errorf("need at least one Mod ID to continue")
					}
					break
				}

				parsed.Mods = append(parsed.Mods, mod)
				fmt.Println(util.Info, "Manually added Mod ID:", mod)

				if !Confirm("Add another Mod ID?", true) {
					break
				}
			}
		} else {
			return false, fmt.Errorf("parsed item has no mods")
		}
	}

	// Retrieve existing mod, item, and map lists from the config
	modList := getFixedArray(config, util.CfgKeyMods)
	itemList := getFixedArray(config, util.CfgKeyItems)
	mapList := getMapList(config)

	// Ask the user to select mods to add if multiple options are available
	var mods []string
	if len(parsed.Mods) == 1 {
		mods = parsed.Mods
	} else {
		modsPrompt := &survey.MultiSelect{
			Message: "Select mods to add:",
			Options: parsed.Mods,
			Default: getEnabledMods(parsed.Mods, modList),
		}

		err = survey.AskOne(modsPrompt, &mods)
		if err != nil {
			return false, err
		}
	}

	// Ensure at least one mod is selected
	if len(mods) == 0 {
		return false, fmt.Errorf("no mods selected")
	}

	// Ask the user where to insert the mod in the list
	options := []string{addStart, addEnd}
	options = append(options, modList...)
	afterPrompt := &survey.Select{
		Message: "Add mod after:",
		Help:    "Press Ctrl+C to cancel adding this mod.",
		Options: options,
		Default: addEnd,
	}

	var addAfter string
	err = survey.AskOne(afterPrompt, &addAfter)
	if err != nil {
		return false, err
	}

	// Insert the mod at the selected position
	if addAfter == addEnd {
		for i := range mods {
			mods[i] = id + "\\" + mods[i]
		}
		modList = append(modList, mods...)
	} else if addAfter == addStart {
		for i := range mods {
			mods[i] = id + "\\" + mods[i]
		}
		modList = append(mods, modList...)
	} else {
		index := util.IndexOf(modList, addAfter)
		if index == -1 {
			return false, fmt.Errorf("could not find mod %s", addAfter)
		}
		for i := range mods {
			mods[i] = id + "\\" + mods[i]
		}
		modList = append(modList[:index+1], append(mods, modList[index+1:]...)...)
	}

	// Prompt the user to add associated maps, if any
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
			// Ask where to insert each map in the list
			for _, m := range maps {
				options = []string{addStart, addEnd}
				options = append(options, mapList...)
				afterPrompt = &survey.Select{
					Message: fmt.Sprintf("Add %s after:", util.Quote(m)),
					Options: options,
					Default: addStart,
				}

				var addAfter string
				err = survey.AskOne(afterPrompt, &addAfter)
				if err != nil {
					return false, err
				}

				if addAfter == addEnd {
					mapList = append(mapList, m)
				} else if addAfter == addStart {
					mapList = append([]string{m}, mapList...)
				} else {
					index := util.IndexOf(mapList, addAfter)
					if index == -1 {
						return false, fmt.Errorf("could not find map %s", addAfter)
					}
					mapList = append(mapList[:index+1], append([]string{m}, mapList[index+1:]...)...)
				}
			}
		}
	}

	// Add the mod ID to the list of workshop items
	itemList = append(itemList, id)

	// Update the server configuration with the new mod, item, and map lists
	config.Set(util.CfgKeyItems, strings.Join(util.Dedupe(itemList), ";"))
	config.Set(util.CfgKeyMods, strings.Join(util.Dedupe(modList), ";"))
	config.Set(util.CfgKeyMap, strings.Join(util.Dedupe(mapList), ";"))

	fmt.Println(util.OK, "Added", util.Quote(item.Title))
	checkDependencies(&item, config)

	return true, nil
}

// AddModWithoutPrompt adds a mod to the server configuration without user interaction.
// It automatically fetches and adds the mod, along with its dependencies.
func AddModWithoutPrompt(id string, config *ini.ServerConfig) bool {
	items, missing, err := steam.FetchWorkshopItems([]string{id})
	if err != nil {
		fmt.Println(util.Error, err)
		return false
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch", (*missing)[0])
		return false
	}

	item := (*items)[0]
	if item.FileType != steam.FileTypeMod {
		fmt.Println(util.Error, "Item is not a mod")
		return false
	}

	modList := getFixedArray(config, util.CfgKeyMods)
	itemList := getFixedArray(config, util.CfgKeyItems)
	mapList := getMapList(config)

	parsed := item.Parse()
	if len(parsed.Mods) == 0 {
		fmt.Println(util.Warning, "Could not parse Mod ID(s) from item:", util.Quote(item.Title))
		return false
	}

	// Add mods to the list
	mods := parsed.Mods
	for i := range mods {
		mods[i] = id + "\\" + mods[i]
	}
	modList = append(modList, mods...)

	// Add associated maps if available
	if len(parsed.Maps) > 0 {
		mapList = append(mapList, parsed.Maps...)
	}

	itemList = append(itemList, id)

	// Fetch and add dependencies
	childItems, _, _ := steam.FetchWorkshopItems(item.GetChildIDs())
	for _, child := range item.Children {
		childItem := steam.FindItemByID(childItems, child.PublishedFileID)
		if childItem != nil && !util.Contains(itemList, child.PublishedFileID) {
			itemList = append(itemList, child.PublishedFileID)
		}
	}

	config.Set(util.CfgKeyItems, strings.Join(util.Dedupe(itemList), ";"))
	config.Set(util.CfgKeyMods, strings.Join(util.Dedupe(modList), ";"))
	config.Set(util.CfgKeyMap, strings.Join(util.Dedupe(mapList), ";"))

	fmt.Println(util.OK, "Added", util.Quote(item.Title))

	return true
}
