package cli

import (
	"github.com/kldzj/pzmod/pkg/service"
	"github.com/kldzj/pzmod/pkg/store"
)

// This file holds the JSON output shapes for --json mode. Keys are camelCase and
// each command emits its own natural object (no universal envelope). Errors are
// reported separately by main.go as {"error": "..."} on stderr.

// modsListJSON is the shape of `mods list --json`.
type modsListJSON struct {
	Mods          []string `json:"mods"`
	WorkshopItems []string `json:"workshopItems"`
	Maps          []string `json:"maps"`
}

// getJSON is the shape of `get <key> --json`.
type getJSON struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// findingJSON is one validation finding in JSON form.
type findingJSON struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Subject  string `json:"subject,omitempty"`
	Message  string `json:"message"`
}

// validateJSON is the shape of `validate --json`.
type validateJSON struct {
	Findings []findingJSON `json:"findings"`
	Summary  struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Info     int `json:"info"`
	} `json:"summary"`
	OK bool `json:"ok"`
}

// searchItemJSON is one Workshop search hit. It uses our own field names rather
// than steam.WorkshopItem's Steam-API json tags.
type searchItemJSON struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	FileSize int64  `json:"fileSize"`
}

// searchJSON is the shape of `search --json`.
type searchJSON struct {
	Total int              `json:"total"`
	Items []searchItemJSON `json:"items"`
}

// profileJSON embeds store.Profile (already json-tagged) and marks the default.
type profileJSON struct {
	store.Profile
	Default bool `json:"default"`
}

// profileListJSON is the shape of `profile list --json`.
type profileListJSON struct {
	Profiles  []profileJSON `json:"profiles"`
	DefaultID string        `json:"defaultId,omitempty"`
}

// backupListJSON is the shape of `backup list --json`.
type backupListJSON struct {
	Backups []store.BackupEntry `json:"backups"`
}

// multiModJSON mirrors domain.MultiModItem for output.
type multiModJSON struct {
	ItemID string   `json:"itemId"`
	ModIDs []string `json:"modIds"`
}

// resolveJSON is the shape of `mods add --resolve-deps --json`.
type resolveJSON struct {
	Resolved         bool           `json:"resolved"`
	AddWorkshopItems []string       `json:"addWorkshopItems"`
	AddMods          []string       `json:"addMods"`
	AddMaps          []string       `json:"addMaps"`
	Missing          []string       `json:"missing"`
	MultiMod         []multiModJSON `json:"multiMod"`
	Cycles           [][]string     `json:"cycles"`
}

// newResolveJSON builds a resolveJSON from a resolution plan.
func newResolveJSON(plan service.ResolvePlan) resolveJSON {
	mm := make([]multiModJSON, 0, len(plan.MultiMod))
	for _, m := range plan.MultiMod {
		mm = append(mm, multiModJSON{ItemID: m.ItemID, ModIDs: orEmpty(m.ModIDs)})
	}
	cycles := plan.Cycles
	if cycles == nil {
		cycles = [][]string{}
	}
	return resolveJSON{
		Resolved:         true,
		AddWorkshopItems: orEmpty(plan.AddWorkshopItems),
		AddMods:          orEmpty(plan.AddMods),
		AddMaps:          orEmpty(plan.AddMaps),
		Missing:          orEmpty(plan.Missing),
		MultiMod:         mm,
		Cycles:           cycles,
	}
}

// shallowAddJSON is the shape of `mods add --json` without --resolve-deps.
type shallowAddJSON struct {
	Added   []string `json:"added"`
	Missing []string `json:"missing"`
}
