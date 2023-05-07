package interactive

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/savioxavier/termlink"
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

	fmt.Println(util.Info, "Fetching collection, this may take a while...")
	items, missing, err := steam.FetchWorkshopItems([]string{id})
	if err != nil {
		fmt.Println(util.Error, err)
		return
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch", (*missing)[0])
		return
	}

	collection := (*items)[0]
	if collection.FileType != steam.FileTypeCollection {
		fmt.Println(util.Error, "Invalid collection")
		return
	}

	children := collection.GetChildIDs()
	items, missing, err = steam.FetchWorkshopItems(children)
	if err != nil {
		fmt.Println(util.Error, err)
		return
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch the following items:")
		for _, id := range *missing {
			fmt.Println(util.Warning, " -", id)
		}
	}

	if len(*items) == 0 {
		fmt.Println(util.Warning, "No items found")
		return
	}

	fmt.Println(util.Info, "If you want to skip specific mods, simply select no mod ids when prompted")
	fmt.Println(util.Info, "Press Ctrl+C to stop adding mods, note that this will save the mods you have already added")
	fmt.Println()

	addedCount := 0
	for _, item := range *items {
		link := item.GetWorkshopUrl()
		if termlink.SupportsHyperlinks() {
			link = termlink.Link("(workshop page)", link)
		} else {
			link = util.Paren(link)
		}

		fmt.Println(util.Info, "Adding", util.Quote(item.Title), link)
		cont, added := addMod(item.PublishedFileID, config)
		if added {
			addedCount++
		}

		fmt.Println()
		if !cont {
			break
		}
	}

	fmt.Println(util.OK, "Added", addedCount, "mods")
}
