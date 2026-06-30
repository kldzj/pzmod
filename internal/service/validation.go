package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kldzj/pzmod/internal/build"
	"github.com/kldzj/pzmod/internal/domain"
	"github.com/kldzj/pzmod/internal/steam"
)

// Validate checks a ServerMods against the Steam Workshop and returns a report.
// It is pure with respect to disk: it never writes, so dry-run validation simply
// calls it on a projected ServerMods.
func (s *Services) Validate(ctx context.Context, sm domain.ServerMods, b build.Build) (domain.Report, error) {
	var report domain.Report

	items, missing, err := s.Steam.GetDetails(ctx, sm.WorkshopItems)
	if err != nil {
		return report, err
	}

	for _, id := range missing {
		report.AddFinding(domain.SeverityError, domain.CodeDelisted, id,
			"workshop item "+id+" could not be fetched (delisted, private, or removed)")
	}

	declaredMods := map[string]bool{}
	declaredMaps := map[string]bool{}
	declaredBy := map[string][]string{} // modID -> item IDs that declare it
	installedItems := toSet(sm.WorkshopItems)

	for _, item := range items {
		if item.Banned {
			report.AddFinding(domain.SeverityError, domain.CodeBanned, item.PublishedFileID,
				title(item)+" is banned on the Workshop")
		}
		parsed := item.Parse()
		if !item.IsCollection() && len(parsed.Mods) == 0 {
			report.AddFinding(domain.SeverityInfo, domain.CodeNoModID, item.PublishedFileID,
				title(item)+" declares no Mod ID (you may need to set it manually)")
		}
		for _, m := range parsed.Mods {
			declaredMods[m] = true
			declaredBy[m] = append(declaredBy[m], item.PublishedFileID)
			if !sm.HasMod(m) {
				report.AddFinding(domain.SeverityWarning, domain.CodeUnusedModID, m,
					"mod ID "+m+" (from "+title(item)+") is not enabled in Mods=")
			}
		}
		for _, mp := range parsed.Maps {
			declaredMaps[mp] = true
			if !sm.HasMap(mp) {
				report.AddFinding(domain.SeverityInfo, domain.CodeUnusedMap, mp,
					"map "+mp+" (from "+title(item)+") is not enabled in Map=")
			}
		}
	}

	// Mods enabled but not provided by any installed item (matched on logical ID).
	for _, raw := range sm.Mods {
		id := domain.ParseModRef(raw).ID
		if id != "" && !declaredMods[id] {
			report.AddFinding(domain.SeverityWarning, domain.CodeUnknownModID, raw,
				"mod ID "+id+" is enabled but not provided by any workshop item")
		}
	}

	// Clash detection: a mod ID declared by two or more installed items.
	var clashIDs []string
	for id, providers := range declaredBy {
		if len(providers) >= 2 {
			clashIDs = append(clashIDs, id)
		}
	}
	sort.Strings(clashIDs)
	for _, id := range clashIDs {
		pin := "unpinned in Mods="
		for _, raw := range sm.Mods {
			if r := domain.ParseModRef(raw); r.ID == id && r.Workshop != "" {
				pin = "pinned to item " + r.Workshop
				break
			}
		}
		report.AddFinding(domain.SeverityWarning, domain.CodeModIDClash, id,
			fmt.Sprintf("mod ID %s is declared by %d items (%s) - %s",
				id, len(declaredBy[id]), strings.Join(declaredBy[id], ", "), pin))
	}

	// Missing dependencies: an item's required children that aren't installed.
	if err := s.appendMissingDeps(ctx, items, installedItems, &report); err != nil {
		return report, err
	}

	// Build compatibility warnings.
	for _, f := range build.CompatWarnings(b, items) {
		report.Add(f)
	}

	return report, nil
}

func (s *Services) appendMissingDeps(ctx context.Context, items []steam.WorkshopItem, installed map[string]bool, report *domain.Report) error {
	var childIDs []string
	seen := map[string]bool{}
	for _, item := range items {
		for _, c := range item.GetChildIDs() {
			if !seen[c] {
				seen[c] = true
				childIDs = append(childIDs, c)
			}
		}
	}
	if len(childIDs) == 0 {
		return nil
	}

	children, _, err := s.Steam.GetDetails(ctx, childIDs)
	if err != nil {
		return err
	}
	childByID := map[string]steam.WorkshopItem{}
	for _, c := range children {
		childByID[c.PublishedFileID] = c
	}

	for _, item := range items {
		for _, c := range item.GetChildIDs() {
			if installed[c] {
				continue
			}
			depName := c
			if ci, ok := childByID[c]; ok {
				depName = title(ci)
			}
			report.AddFinding(domain.SeverityError, domain.CodeMissingDependency, c,
				"missing dependency "+depName+" ("+c+") required by "+title(item))
		}
	}
	return nil
}

func title(item steam.WorkshopItem) string {
	if item.Title != "" {
		return item.Title
	}
	return item.PublishedFileID
}
