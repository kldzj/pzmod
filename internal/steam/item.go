package steam

import (
	"fmt"
	"strings"

	"github.com/kldzj/pzmod/internal/domain"
)

// Workshop file types returned by the Steam API.
const (
	FileTypeMod        = 0
	FileTypeCollection = 2
)

// WorkshopItemChild is a member of a collection or a declared dependency.
type WorkshopItemChild struct {
	PublishedFileID string `json:"publishedfileid"`
	FileType        int    `json:"file_type"`
}

// WorkshopTag is a Steam Workshop tag (e.g. "Build 41", "Map").
type WorkshopTag struct {
	Tag string `json:"tag"`
}

// WorkshopItem is a published Workshop file (mod, map, or collection).
type WorkshopItem struct {
	Result          uint8               `json:"result"`
	FileType        uint8               `json:"file_type"`
	FileSize        ItemSize            `json:"file_size"`
	PublishedFileID string              `json:"publishedfileid"`
	Creator         string              `json:"creator,omitempty"`
	Description     string              `json:"file_description,omitempty"`
	ShortDesc       string              `json:"short_description,omitempty"`
	Title           string              `json:"title,omitempty"`
	Banned          bool                `json:"banned,omitempty"`
	PreviewURL      string              `json:"preview_url,omitempty"`
	TimeUpdated     int64               `json:"time_updated,omitempty"`
	Subscriptions   int64               `json:"subscriptions,omitempty"`
	Views           int64               `json:"views,omitempty"`
	Tags            []WorkshopTag       `json:"tags,omitempty"`
	Children        []WorkshopItemChild `json:"children,omitempty"`
}

// ParsedItem holds the mod IDs and map folders scraped from a mod's description.
type ParsedItem struct {
	Mods []string
	Maps []string
}

// Parse scrapes the item description for the mod IDs and map folders the game
// uses. It tolerates the two casing variants Steam mod authors use in practice.
func (w *WorkshopItem) Parse() *ParsedItem {
	var mods, maps []string
	for _, line := range strings.Split(w.Description, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Mod ID: "):
			mods = append(mods, strings.TrimSpace(strings.TrimPrefix(line, "Mod ID: ")))
		case strings.HasPrefix(line, "ModID: "):
			mods = append(mods, strings.TrimSpace(strings.TrimPrefix(line, "ModID: ")))
		case strings.HasPrefix(line, "Map Folder: "):
			maps = append(maps, strings.TrimSpace(strings.TrimPrefix(line, "Map Folder: ")))
		}
	}
	return &ParsedItem{Mods: domain.Dedupe(mods), Maps: domain.Dedupe(maps)}
}

// IsCollection reports whether the item is a Workshop collection.
func (w *WorkshopItem) IsCollection() bool { return w.FileType == FileTypeCollection }

// GetChildIDs returns the published file IDs of the item's children.
func (w *WorkshopItem) GetChildIDs() []string {
	ids := make([]string, len(w.Children))
	for i, child := range w.Children {
		ids[i] = child.PublishedFileID
	}
	return ids
}

// WorkshopURL returns the public Steam Workshop page URL for the item.
func (w *WorkshopItem) WorkshopURL() string {
	return fmt.Sprintf("https://steamcommunity.com/sharedfiles/filedetails/?id=%s", w.PublishedFileID)
}

// FindItemByID returns the item with the given ID, or nil.
func FindItemByID(items []WorkshopItem, id string) *WorkshopItem {
	for i := range items {
		if items[i].PublishedFileID == id {
			return &items[i]
		}
	}
	return nil
}
