package interactive

import (
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
		cmd.Println(util.Error, err)
		return
	}

	cmd.Println(util.Info, "Fetching collection, this may take a while...")
	items, missing, err := steam.FetchWorkshopItems([]string{id})
	if err != nil {
		cmd.Println(util.Error, err)
		return
	}

	if len(*missing) > 0 {
		cmd.Println(util.Warning, "Could not fetch", (*missing)[0])
		return
	}

	collection := (*items)[0]
	if collection.FileType != steam.FileTypeCollection {
		cmd.Println(util.Error, "Invalid collection")
		return
	}

	children := collection.GetChildIDs()
	items, missing, err = steam.FetchWorkshopItems(children)
	if err != nil {
		cmd.Println(util.Error, err)
		return
	}

	if len(*missing) > 0 {
		cmd.Println(util.Warning, "Could not fetch the following items:")
		for _, id := range *missing {
			cmd.Println(util.Warning, " -", id)
		}
	}

	if len(*items) == 0 {
		cmd.Println(util.Warning, "No items found")
		return
	}

	cmd.Println(util.Info, "Press Ctrl+C to skip an item")
	cmd.Println()

	addedCount := 0
	for _, item := range *items {
		link := item.GetWorkshopUrl()
		if termlink.SupportsHyperlinks() {
			link = termlink.Link("(workshop page)", link)
		} else {
			link = util.Paren(link)
		}

		cmd.Println(util.Info, "Adding", util.Quote(item.Title), link)
		added, err := addMod(item.PublishedFileID, config)
		if err != nil {
			cmd.Println(util.Warning, err)
			if !Continue("adding mods") {
				break
			}
		}

		if added {
			addedCount++
		}

		cmd.Println()
	}

	cmd.Println(util.OK, "Added", addedCount, "mods")
}
