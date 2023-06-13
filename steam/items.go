package steam

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kldzj/pzmod/util"
	"github.com/savioxavier/termlink"
)

type WorkshopItemChild struct {
	PublishedFileID string `json:"publishedfileid"`
	FileType        int    `json:"file_type"`
}

type WorkshopItem struct {
	Result          uint8               `json:"result"`
	FileType        uint8               `json:"file_type"`
	FileSize        ItemSize            `json:"file_size"`
	PublishedFileID string              `json:"publishedfileid"`
	Creator         string              `json:"creator,omitempty"`
	Description     string              `json:"file_description,omitempty"`
	Title           string              `json:"title,omitempty"`
	Banned          bool                `json:"banned,omitempty"`
	Children        []WorkshopItemChild `json:"children,omitempty"`
}

type WorkshopItemsResponse struct {
	PublishedFileDetails []WorkshopItem `json:"publishedfiledetails"`
}

type WorkshopItemResponse struct {
	Response WorkshopItemsResponse `json:"response"`
}

type ParsedWorkshopItem struct {
	Mods []string
	Maps []string
}

func (w *WorkshopItem) Parse() *ParsedWorkshopItem {
	var mods []string
	var maps []string

	lines := strings.Split(w.Description, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Mod ID: ") {
			mods = append(mods, strings.TrimPrefix(line, "Mod ID: "))
		} else if strings.HasPrefix(line, "ModID: ") {
			mods = append(mods, strings.TrimPrefix(line, "ModID: "))
		} else if strings.HasPrefix(line, "Map Folder: ") {
			maps = append(maps, strings.TrimPrefix(line, "Map Folder: "))
		}
	}

	return &ParsedWorkshopItem{
		Mods: util.Dedupe(mods),
		Maps: util.Dedupe(maps),
	}
}

func (w *WorkshopItem) GetChildIDs() []string {
	ids := make([]string, len(w.Children))
	for i, child := range w.Children {
		ids[i] = child.PublishedFileID
	}

	return ids
}

func (w *WorkshopItem) GetWorkshopUrl() string {
	return fmt.Sprintf("https://steamcommunity.com/sharedfiles/filedetails/?id=%s", w.PublishedFileID)
}

func (w *WorkshopItem) GetWorkshopLink() string {
	link := w.GetWorkshopUrl()
	if termlink.SupportsHyperlinks() {
		link = termlink.Link("(workshop page)", link)
	} else {
		link = util.Paren(link)
	}

	return link
}

func FindItemByID(items *[]WorkshopItem, id string) *WorkshopItem {
	for _, item := range *items {
		if item.PublishedFileID == id {
			return &item
		}
	}

	return nil
}

var cache *WorkshopItemCache = NewWorkshopItemCache(5 * time.Minute)

func FetchWorkshopItems(ids []string) (*[]WorkshopItem, *[]string, error) {
	var toFetch []string
	var items []WorkshopItem
	for _, id := range ids {
		item, ok := cache.Get(id)
		if ok {
			items = append(items, *item)
			continue
		}

		toFetch = append(toFetch, id)
	}

	if len(toFetch) > 0 {
		fetchedItems, missing, err := fetchWorkshopItems(toFetch)
		if err != nil {
			return nil, nil, err
		}

		for _, item := range *fetchedItems {
			items = append(items, item)
			cache.Set(item.PublishedFileID, item)
		}

		return &items, missing, nil
	}

	return &items, &[]string{}, nil
}

func fetchWorkshopItems(ids []string) (*[]WorkshopItem, *[]string, error) {
	var items []WorkshopItem
	var missing []string
	chunks := chunkSlice(ids, 10)
	for _, chunk := range chunks {
		chunkItems, err := fetchWorkshopItemsChunk(chunk)
		if err != nil {
			return nil, nil, err
		}

		for _, item := range *chunkItems {
			if item.Result != 1 {
				missing = append(missing, item.PublishedFileID)
				continue
			}

			items = append(items, item)
		}
	}

	return &items, &missing, nil
}

func fetchWorkshopItemsChunk(ids []string) (*[]WorkshopItem, error) {
	client := newHttpClient()
	req, err := client.Get(constructWorkshopItemURL(ids))
	if err != nil {
		return nil, err
	}

	if req.StatusCode != 200 {
		if req.StatusCode == 401 {
			return nil, fmt.Errorf("steam api key is invalid")
		}

		return nil, fmt.Errorf("workshop item request failed with status code %s", req.Status)
	}

	defer req.Body.Close()
	var response WorkshopItemResponse
	err = json.NewDecoder(req.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response.Response.PublishedFileDetails, nil
}

func constructWorkshopItemURL(ids []string) string {
	url, query := constructSteamApiUrl("/IPublishedFileService/GetDetails/v1")
	query.Add("includechildren", "true")

	for idx, id := range ids {
		idxStr := fmt.Sprint(idx)
		query.Add("publishedfileids["+idxStr+"]", id)
	}

	url.RawQuery = query.Encode()
	return url.String()
}

func chunkSlice(slice []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}
