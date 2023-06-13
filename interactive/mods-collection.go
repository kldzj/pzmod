package interactive

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func cmdAddModsFromCollection(cmd *cobra.Command, config *ini.ServerConfig) {
	var id string
	prompt := &survey.Input{
		Message: "Collection Workshop ID:",
	}

	err := survey.AskOne(prompt, &id)
	if err != nil {
		fmt.Println(util.Error, err)
		return
	}

	added := addModsFromCollection(id, config)
	fmt.Println(util.OK, "Added", added, "mods")
}

func addModsFromCollection(id string, config *ini.ServerConfig) int {
	fmt.Println(util.Info, "Fetching collection, this may take a while...")
	items, missing, err := steam.FetchWorkshopItems([]string{id})
	if err != nil {
		fmt.Println(util.Error, err)
		return 0
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch", (*missing)[0])
		return 0
	}

	collection := (*items)[0]
	if collection.FileType != steam.FileTypeCollection {
		fmt.Println(util.Error, "Invalid collection")
		return 0
	}

	children := collection.GetChildIDs()
	items, missing, err = steam.FetchWorkshopItems(children)
	if err != nil {
		fmt.Println(util.Error, err)
		return 0
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch the following items:")
		for _, id := range *missing {
			fmt.Println(util.Warning, " -", id)
		}
	}

	if len(*items) == 0 {
		fmt.Println(util.Warning, "No items found")
		return 0
	}

	fmt.Println(util.Info, "Press Ctrl+C to skip an item")
	fmt.Println()

	addedCount := 0
	for _, item := range *items {
		link := item.GetWorkshopLink()
		if item.FileType == steam.FileTypeMod {
			added, err := addMod(item.PublishedFileID, config)
			if err != nil {
				fmt.Println(util.Warning, err)
				if !Continue("adding mods") {
					break
				}
			}

			if added {
				addedCount++
			}
		} else if item.FileType == steam.FileTypeCollection {
			fmt.Println(util.Warning, util.Quote(item.Title), link, "is a collection containing", len(item.GetChildIDs()), "items")
			if !Confirm("Do you want to step through the collection and add each item?", true) {
				continue
			}

			addedCount += addModsFromCollection(item.PublishedFileID, config)
			fmt.Println(util.Info, "Done with", util.Quote(item.Title))
		} else {
			fmt.Println(util.Warning, util.Quote(item.Title), link, "is not a mod")
		}

		fmt.Println()
	}

	return addedCount
}
