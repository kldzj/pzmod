package steam

import (
	"fmt"
	"strings"

	"github.com/kldzj/pzmod/pkg/domain"
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
// uses. Some descriptions advertise *other* mods by reproducing their PZ footer
// (a "Workshop ID:" line paired with a "Mod ID:" / "Map Folder:" line), which
// would otherwise leak those foreign IDs into this item. To guard against that,
// each Mod ID / Map Folder is attributed to the nearest "Workshop ID:" line in
// either direction (authors write the pair in both orders, sometimes separated
// by a blank line; ties favour the item's own ID) and is dropped only when that
// nearest Workshop ID belongs to a *different* item. Attribution only kicks in
// once the item's own Workshop ID is seen; descriptions that never state it, or
// omit Workshop IDs entirely, keep every entry as before. It tolerates the
// casing variants ("Mod ID:"/"ModID:", "Workshop ID:"/"WorkshopID:") authors
// use in practice.
func (w *WorkshopItem) Parse() *ParsedItem {
	lines := strings.Split(w.Description, "\n")

	// First pass: locate every Workshop ID line and note whether our own appears.
	type widLine struct {
		idx int
		own bool
	}
	var wids []widLine
	ownSeen := false
	for i, line := range lines {
		if v, ok := fieldValue(strings.TrimSpace(line), "Workshop ID:", "WorkshopID:"); ok {
			own := v != "" && v == w.PublishedFileID
			wids = append(wids, widLine{idx: i, own: own})
			ownSeen = ownSeen || own
		}
	}

	// keep reports whether the footer entry on line idx should be attributed to
	// this item. With no own Workshop ID to anchor against, keep everything.
	keep := func(idx int) bool {
		if !ownSeen {
			return true
		}
		bestDist, nearestOwn := -1, true
		for _, wl := range wids {
			d := wl.idx - idx
			if d < 0 {
				d = -d
			}
			if bestDist == -1 || d < bestDist || (d == bestDist && wl.own) {
				bestDist, nearestOwn = d, wl.own
			}
		}
		return nearestOwn
	}

	var mods, maps []string
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if v, ok := fieldValue(line, "Mod ID:", "ModID:"); ok {
			if keep(i) {
				mods = append(mods, v)
			}
		} else if v, ok := fieldValue(line, "Map Folder:"); ok {
			if keep(i) {
				maps = append(maps, v)
			}
		}
	}
	return &ParsedItem{Mods: domain.Dedupe(mods), Maps: domain.Dedupe(maps)}
}

// fieldValue returns the trimmed value of the first "Key value" line among keys,
// matching each key with a trailing space per the PZ footer convention (so
// "Mod ID:Foo" without the space is not treated as a footer line), and whether
// any key matched.
func fieldValue(line string, keys ...string) (string, bool) {
	for _, k := range keys {
		p := k + " "
		if strings.HasPrefix(line, p) {
			return strings.TrimSpace(strings.TrimPrefix(line, p)), true
		}
	}
	return "", false
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
