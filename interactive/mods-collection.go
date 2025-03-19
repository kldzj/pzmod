package interactive

import (
	"errors"
	"fmt"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kldzj/pzmod/ini"
	"github.com/kldzj/pzmod/steam"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

// cmdAddModsFromCollection prompts the user for a Steam Workshop Collection ID
// and attempts to add all mods from that collection to the server configuration.
func cmdAddModsFromCollection(cmd *cobra.Command, config *ini.ServerConfig) {
	var id string
	prompt := &survey.Input{
		Message: "Collection Workshop ID:",
	}

	// Ask the user to input a collection ID
	err := survey.AskOne(prompt, &id)
	if err != nil {
		fmt.Println(util.Error, err)
		return
	}

	if id == "" {
		fmt.Println(util.Error, "value cannot be empty!")
		return
	}

	// Add mods from the specified collection
	added, errors := addModsFromCollection(id, config)

	// Display results
	if added > 0 {
		fmt.Println(util.OK, "Successfully added", added, "mods")
	} else {
		fmt.Println(util.Error, "Failed to add mods")
	}

	// Display errors if any occurred
	if len(errors) > 0 {
		fmt.Println(util.Warning, "Errors occurred:")
		for _, err := range errors {
			fmt.Println(util.Warning, " -", err)
		}
	}
}

// addModsFromCollection retrieves mods from a Steam Workshop collection and adds them to the server configuration.
func addModsFromCollection(id string, config *ini.ServerConfig) (int, []error) {
	if id == "" {
		return 0, []error{errors.New("collection ID cannot be empty")}
	}

	fmt.Println(util.Info, "Fetching collection, this may take a while...")

	// Fetch the collection details from the Steam Workshop
	items, missing, err := steam.FetchWorkshopItems([]string{id})
	if err != nil {
		return 0, []error{err}
	}

	// Warn if the collection could not be fetched
	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch", (*missing)[0])
		return 0, nil
	}

	collection := (*items)[0]
	if collection.FileType != steam.FileTypeCollection {
		return 0, []error{fmt.Errorf("invalid collection")}
	}

	children := collection.GetChildIDs()
	items, missing, err = steam.FetchWorkshopItems(children)
	if err != nil {
		return 0, []error{err}
	}

	if len(*missing) > 0 {
		fmt.Println(util.Warning, "Could not fetch the following items:")
		for _, id := range *missing {
			fmt.Println(util.Warning, " -", id)
		}
	}

	if len(*items) == 0 {
		fmt.Println(util.Warning, "No items found")
		return 0, nil
	}

	addedCount := 0
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)
	errorChan := make(chan error, len(*items))

	processItem := func(item steam.WorkshopItem) {
		defer wg.Done()
		if item.FileType == steam.FileTypeMod {
			added := AddModWithoutPrompt(item.PublishedFileID, config)
			if !added {
				errorChan <- fmt.Errorf("failed to add mod %s", item.PublishedFileID)
				return
			}
			mu.Lock()
			addedCount++
			mu.Unlock()
		} else if item.FileType == steam.FileTypeCollection {
			subAdded, subErrors := addModsFromCollection(item.PublishedFileID, config)
			mu.Lock()
			addedCount += subAdded
			mu.Unlock()
			if len(subErrors) > 0 {
				for _, err := range subErrors {
					errorChan <- err
				}
			}
		}
	}

	for _, item := range *items {
		wg.Add(1)
		go processItem(item)
	}

	wg.Wait()
	close(errorChan)

	for err := range errorChan {
		mu.Lock()
		errors = append(errors, err)
		mu.Unlock()
	}

	return addedCount, errors
}
