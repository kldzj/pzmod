package interactive

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

// cmdCheckMods runs a check for potential issues with mods in the server configuration.
func cmdCheckMods(cmd *cobra.Command, config *ini.ServerConfig) {
	if checkForProblems(config) {
		fmt.Println(util.Info, "No warnings or errors below means no issues were detected")
	}
}

// checkForProblems verifies if there are any issues with the mods and maps in the server configuration.
func checkForProblems(config *ini.ServerConfig) bool {
	itemList := getFixedArray(config, util.CfgKeyItems)
	modList := getFixedArray(config, util.CfgKeyMods)
	mapList := getMapList(config)

	// Create a map of all workshop IDs to ensure uniqueness.
	workshopIDs := make(map[string]bool)

	// Add workshop IDs from item list.
	for _, id := range itemList {
		workshopIDs[id] = true
	}

	// Extract workshop IDs from mod list.
	for _, modWithWorkshop := range modList {
		parts := strings.Split(modWithWorkshop, "\\")
		var workshopID string
		if len(parts) == 2 {
			workshopID = parts[0]
		} else {
			workshopID = modWithWorkshop
		}
		workshopIDs[workshopID] = true
	}

	// Validate each workshop ID using parallel processing.
	var wg sync.WaitGroup
	var mu sync.Mutex
	valid := true
	for id := range workshopIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, missing, err := steam.FetchWorkshopItems([]string{id})
			if err != nil {
				mu.Lock()
				fmt.Println(util.Error, err)
				valid = false
				mu.Unlock()
				return
			}

			// Check if the ID is invalid.
			if len(*missing) > 0 {
				mu.Lock()
				fmt.Println(util.Warning, "Invalid Steam Workshop ID:", id)
				valid = false
				mu.Unlock()
				return
			}
		}(id)
	}
	wg.Wait()
	if !valid {
		return false
	}

	// Fetch information about the workshop items.
	items, missingItems, err := steam.FetchWorkshopItems(itemList)
	if err != nil {
		fmt.Println(util.Error, err)
		return false
	}

	// Check if any items are missing.
	if len(*missingItems) > 0 {
		fmt.Println(util.Warning, "Could not fetch the following items:")
		for _, item := range *missingItems {
			fmt.Println(util.Warning, item)
			return false
		}
	}

	// Extract mod IDs and map names from fetched workshop items.
	modIDs := []string{}
	mapNames := []string{}
	for _, item := range *items {
		parsed := item.Parse()
		modIDs = append(modIDs, parsed.Mods...)
		mapNames = append(mapNames, parsed.Maps...)
	}

	// Verify that all mods in the mod list exist in the fetched workshop items.
	for _, modWithWorkshop := range modList {
		parts := strings.Split(modWithWorkshop, "\\")
		var modID string
		if len(parts) == 2 {
			modID = parts[1]
		} else {
			modID = modWithWorkshop
		}

		found := false
		for _, id := range modIDs {
			if id == modID {
				found = true
				break
			}
		}

		// Warn if a mod ID is unknown (not in the fetched workshop items).
		if !found {
			fmt.Println(util.Warning, "Unknown mod ID:", modWithWorkshop, "(not in workshop items)")
			return false
		}
	}

	// Check for unused mod IDs.
	for _, mod := range modIDs {
		found := false
		for _, m := range modList {
			parts := strings.Split(m, "\\")
			if len(parts) == 2 && parts[1] == mod {
				found = true
				break
			} else if m == mod {
				found = true
				break
			}
		}
		if !found {
			fmt.Println(util.Warning, "Unused mod ID:", mod)
			return false
		}
	}

	// Verify that all maps in the map list are found in the workshop items.
	for _, mapName := range mapList {
		found := false
		for _, item := range *items {
			parsed := item.Parse()
			if util.Contains(parsed.Maps, mapName) {
				found = true
				break
			}
		}
		if !found {
			fmt.Println(util.Warning, "Map", mapName, "is not found in any Workshop Item")
			return false
		}
	}

	// Check if any mods are mistakenly used as maps.
	for _, modWithWorkshop := range modList {
		parts := strings.Split(modWithWorkshop, "\\")
		var modID string
		if len(parts) == 2 {
			modID = parts[1]
		} else {
			modID = modWithWorkshop
		}

		found := false
		for _, mapName := range mapList {
			if modID == mapName {
				found = true
				break
			}
		}
		if !found && util.Contains(mapNames, modID) {
			fmt.Println(util.Warning, "Mod ID used as map but not listed in Map=:", modID)
			return false
		}
	}

	// Check dependencies of workshop items.
	for _, item := range *items {
		childItems, missingChildren, err := steam.FetchWorkshopItems(item.GetChildIDs())
		if err != nil {
			fmt.Println(util.Error, err)
			return false
		}

		// Warn about missing dependencies.
		if len(*missingChildren) > 0 {
			fmt.Println(util.Warning, "Could not fetch the following dependencies for", util.Quote(item.Title)+":")
			for _, child := range *missingChildren {
				fmt.Println(util.Warning, child)
				return false
			}
		}

		// Ensure all dependencies are present in the item list.
		for _, child := range item.Children {
			childItem := steam.FindItemByID(childItems, child.PublishedFileID)
			title := "an unknown mod"
			if childItem != nil {
				title = util.Quote(childItem.Title)
			}

			if !util.Contains(itemList, child.PublishedFileID) {
				fmt.Println(
					util.Error, "Missing dependency:", title, util.Paren(child.PublishedFileID)+",",
					"required by", util.Quote(item.Title), util.Paren(item.PublishedFileID),
				)
				return false
			}
		}
	}
	return true
}
