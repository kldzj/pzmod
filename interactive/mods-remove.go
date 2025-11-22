package interactive

import (
	"fmt"
	"strings"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

// Options for bulk mod and map removal
const (
	removeAllOption         = "Remove all mods"
	removeAllKeepMapsOption = "Remove all mods (keep maps)"
	removeAllMapsOption     = "Remove all maps"
)

// cmdRemoveMods continuously prompts the user to remove mods or maps
// until the user decides to stop or there are no more mods/maps left.
func cmdRemoveMods(cmd *cobra.Command, config *ini.ServerConfig) {
	for {
		exit := removeMod(config)
		if exit {
			break
		}

		// Check if any mods or maps are left after the removal
		hasMods := config.GetOrDefault(util.CfgKeyMods, "") != ""
		hasMaps := config.GetOrDefault(util.CfgKeyMap, "") != ""

		if !hasMods && !hasMaps {
			fmt.Println(util.Info, "No more mods or maps to remove.")
			break
		}
	}
}

// removeMod handles interactive mod or map removal.
// Returns true if the user chooses to exit or no mods/maps remain.
func removeMod(config *ini.ServerConfig) bool {
	modList := getFixedArray(config, util.CfgKeyMods)
	itemList := getFixedArray(config, util.CfgKeyItems)
	mapList := getMapList(config)

	// If there are no mods or maps, print a message and exit.
	if len(itemList) == 0 && len(mapList) == 0 {
		fmt.Println(util.Info, "No mods or maps to remove.")
		return true
	}

	// Populate options for user selection
	options := []string{itCmdExit}

	if len(itemList) > 0 {
		options = append(options, removeAllOption, removeAllKeepMapsOption)
	}

	if len(mapList) > 0 {
		options = append(options, removeAllMapsOption)
	}

	// Fetching mod details from the Steam Workshop
	if len(itemList) > 0 {
		fmt.Println(util.Info, "Fetching workshop items...")

		items, missing, err := steam.FetchWorkshopItems(itemList)
		if err != nil {
			return true
		}

		// Display a warning for missing items
		if len(*missing) > 0 {
			fmt.Println(util.Warning, "Could not fetch the following items:")
			for _, id := range *missing {
				fmt.Println(util.Warning, " -", id)
			}
		}

		// If there are valid workshop items, allow user selection
		if len(*items) > 0 {
			titles := mapTitlesToIDs(items)
			options = append(options, itemList...)

			// Prompt the user to select a mod or map for removal
			itemPrompt := &survey.Select{
				Message: "Select mod or map to remove:",
				Options: options,
				Description: func(value string, index int) string {
					switch value {
					case itCmdExit:
						return "Stop removing mods or maps"
					case removeAllOption:
						return "Remove all mods and maps"
					case removeAllKeepMapsOption:
						return "Remove all mods but keep maps"
					case removeAllMapsOption:
						return "Remove all maps"
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
				return true
			}

			if id == "" || id == itCmdExit {
				return true
			}

			// Handle bulk removal options
			if id == removeAllOption {
				if Confirm("Are you sure you want to remove all mods and maps?", false) {
					removeAllMods(config, false)
					return true
				}
				return false
			}

			if id == removeAllKeepMapsOption {
				if Confirm("Are you sure you want to remove all mods but keep maps?", false) {
					removeAllMods(config, true)
					return true
				}
				return false
			}

			if id == removeAllMapsOption {
				if Confirm("Are you sure you want to remove all maps?", false) {
					removeAllMaps(config)
					return true
				}
				return false
			}

			// Find and remove the selected mod
			item := steam.FindItemByID(items, id)
			if item == nil {
				fmt.Println(util.Error, "Could not find item with id", id)
				return false
			}

			if !Confirm("Are you sure you want to remove "+util.Quote(item.Title)+"?", true) {
				return false
			}

			fmt.Println(util.Info, "Removing", util.Quote(item.Title), "...")

			parsed := item.Parse()

			// Remove the mod, item, and any related maps
			modList = util.Filter(modList, func(s string) bool {
				return !strings.HasPrefix(s, id+"\\")
			})

			itemList = util.Filter(itemList, func(s string) bool {
				return s != id
			})

			mapList = util.Filter(mapList, func(s string) bool {
				return !util.Contains(parsed.Maps, s)
			})

			config.Set(util.CfgKeyMods, strings.Join(modList, ";"))
			config.Set(util.CfgKeyItems, strings.Join(itemList, ";"))
			config.Set(util.CfgKeyMap, strings.Join(mapList, ";"))

			fmt.Println(util.OK, "Removed", util.Quote(item.Title))

			if len(itemList) == 0 && len(mapList) > 0 {
				return true
			}

			return false
		}
	}

	// Handle map-only removal
	if len(mapList) > 0 {
		removeMaps := Confirm("Are you sure you want to remove all maps?", false)
		if removeMaps {
			removeAllMaps(config)
		}
		return true
	}

	return false
}

// removeAllMods removes all mods, optionally keeping maps if specified.
func removeAllMods(config *ini.ServerConfig, keepMaps bool) {
	if !keepMaps {
		config.Set(util.CfgKeyMods, "")
		config.Set(util.CfgKeyItems, "")
		config.Set(util.CfgKeyMap, "")
		fmt.Println(util.OK, "All mods and maps have been removed successfully.")
		return
	}

	mapList := getMapList(config)
	itemList := getFixedArray(config, util.CfgKeyItems)
	modList := getFixedArray(config, util.CfgKeyMods)
	neededItemIDs := make(map[string]bool)

	fmt.Println(util.Info, "Fetching workshop items for dependency check...")

	items, _, err := steam.FetchWorkshopItems(itemList)
	if err != nil {
		fmt.Println(util.Error, "Failed to fetch workshop items:", err)
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Identify mods required for maps
	for _, mapName := range mapList {
		wg.Add(1)
		go func(mapName string) {
			defer wg.Done()
			for _, item := range *items {
				parsed := item.Parse()
				if util.Contains(parsed.Maps, mapName) {
					mu.Lock()
					neededItemIDs[item.PublishedFileID] = true
					mu.Unlock()
				}
			}
		}(mapName)
	}

	wg.Wait()

	// Keep only required mods for maps
	itemList = util.Filter(itemList, func(id string) bool {
		return neededItemIDs[id]
	})

	modList = util.Filter(modList, func(mod string) bool {
		for id := range neededItemIDs {
			if strings.HasPrefix(mod, id+"\\") {
				return true
			}
		}
		return false
	})

	config.Set(util.CfgKeyMods, strings.Join(modList, ";"))
	config.Set(util.CfgKeyItems, strings.Join(itemList, ";"))

	fmt.Println(util.OK, "All mods have been removed, but map-related mods were kept.")
}

// removeAllMaps removes all maps and associated mods.
func removeAllMaps(config *ini.ServerConfig) {

	mapList := getMapList(config)

	itemList := getFixedArray(config, util.CfgKeyItems)

	modList := getFixedArray(config, util.CfgKeyMods)

	mapRelatedItems := make(map[string]bool)

	fmt.Println(util.Info, "Fetching workshop items for map dependency check...")

	items, _, err := steam.FetchWorkshopItems(itemList)

	if err != nil {

		fmt.Println(util.Error, "Failed to fetch workshop items:", err)

		return

	}

	var wg sync.WaitGroup

	var mu sync.Mutex

	for _, mapName := range mapList {

		wg.Add(1)

		go func(mapName string) {

			defer wg.Done()

			for _, item := range *items {

				parsed := item.Parse()

				if util.Contains(parsed.Maps, mapName) {

					mu.Lock()

					mapRelatedItems[item.PublishedFileID] = true

					mu.Unlock()

				}

			}

		}(mapName)

	}

	wg.Wait()

	itemList = util.Filter(itemList, func(id string) bool {

		return !mapRelatedItems[id]

	})

	modList = util.Filter(modList, func(mod string) bool {

		for id := range mapRelatedItems {

			if strings.HasPrefix(mod, id+"\\") {

				return false

			}

		}

		return true

	})

	config.Set(util.CfgKeyMap, "")

	config.Set(util.CfgKeyMods, strings.Join(modList, ";"))

	config.Set(util.CfgKeyItems, strings.Join(itemList, ";"))

	fmt.Println(util.OK, "All maps and map-related mods have been removed successfully.")

}

// mapTitlesToIDs creates a mapping of mod IDs to their titles.
func mapTitlesToIDs(items *[]steam.WorkshopItem) map[string]string {

	m := make(map[string]string)

	for _, item := range *items {

		m[item.PublishedFileID] = item.Title

	}

	return m

}
