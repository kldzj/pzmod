package interactive

import (
	"github.com/fatih/color"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func cmdListMods(cmd *cobra.Command, config *ini.ServerConfig) {
	workshopList := getFixedArray(config, "WorkshopItems")
	if len(workshopList) == 0 {
		cmd.Println(util.Warning, "No workshop items found")
		return
	}

	modList := getFixedArray(config, "Mods")
	mapList := getMapList(config)
	items, missing, err := steam.FetchWorkshopItems(workshopList)
	if err != nil {
		cmd.Println(util.Error, err)
		return
	}

	for _, item := range *missing {
		cmd.Println(util.Warning, "Could not fetch workshop item", item)
	}

	for idx, item := range *items {
		parsed := item.Parse()
		cmd.Println(util.Bold(item.Title), "("+item.PublishedFileID+")")

		if len(parsed.Mods) > 0 {
			cmd.Println(" ", util.Underline("Available mods"))
			for _, mod := range parsed.Mods {
				if isEnabled(mod, modList) {
					cmd.Println("   -", mod)
				} else {
					cmd.Println("   -", color.YellowString(mod))
				}
			}
		} else {
			cmd.Println(util.Warning, "Parsed workshop item has no mods")
		}

		if len(parsed.Maps) > 0 {
			cmd.Println(" ", util.Underline("Available maps"))
			for _, mapName := range parsed.Maps {
				if isEnabled(mapName, mapList) {
					cmd.Println("   -", mapName)
				} else {
					cmd.Println("   -", color.YellowString(mapName))
				}
			}
		}

		if idx < len(*items)-1 {
			cmd.Println()
		}
	}
}
