package interactive

import (
	"fmt"

	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func cmdCheckMods(cmd *cobra.Command, config *ini.ServerConfig) {
	fmt.Println(util.Info, "No warnings or errors below means no issues were detected")
	checkForProblems(config)
}

func checkForProblems(config *ini.ServerConfig) {
	itemList := getFixedArray(config, "WorkshopItems")
	modList := getFixedArray(config, "Mods")

	items, missing, err := steam.FetchWorkshopItems(itemList)
	if err != nil {
		fmt.Println(util.Error, err)
		return
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch the following items:")
		for _, item := range *missing {
			fmt.Println(util.Warning, item)
		}
	}

	modIDs := []string{}
	mapNames := []string{}
	for _, item := range *items {
		parsed := item.Parse()
		modIDs = append(modIDs, parsed.Mods...)
		mapNames = append(mapNames, parsed.Maps...)
	}

	for _, mod := range modIDs {
		if !util.Contains(modList, mod) {
			fmt.Println(util.Warning, "Unused mod ID:", mod)
		}
	}

	mapList := getMapList(config)
	for _, mapName := range mapNames {
		if !util.Contains(mapList, mapName) {
			fmt.Println(util.Warning, "Unused map name:", mapName)
		}
	}

	for _, mod := range modList {
		if !util.Contains(modIDs, mod) {
			fmt.Println(util.Warning, "Unknown mod ID:", mod, "(not in workshop items)")
		}
	}

	for _, item := range *items {
		childItems, missingChildren, err := steam.FetchWorkshopItems(item.GetChildIDs())
		if err != nil {
			fmt.Println(util.Error, err)
			return
		}

		if len(*missingChildren) > 0 {
			fmt.Println(util.Warning, "Could not fetch the following dependencies for", util.Quote(item.Title)+":")
			for _, child := range *missingChildren {
				fmt.Println(util.Warning, child)
			}
		}

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
			}
		}
	}
}
