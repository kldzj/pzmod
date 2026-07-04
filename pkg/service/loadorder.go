package service

import (
	"context"
	"strings"

	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/store"
)

// SuggestLoadOrder proposes a load order for the enabled mods. It derives
// dependency edges from Workshop "required items" and (when a content path is
// configured and present) from on-disk mod.info "require=" fields, then biases
// framework/library mods toward the front. It only suggests; the caller applies.
func (s *Services) SuggestLoadOrder(ctx context.Context, sm domain.ServerMods, profile store.Profile) (domain.OrderPlan, error) {
	items, _, err := s.Steam.GetDetails(ctx, sm.WorkshopItems)
	if err != nil {
		return domain.OrderPlan{}, err
	}

	// Fetch dependency children too, so we know which mods they provide.
	var childIDs []string
	for _, item := range items {
		childIDs = append(childIDs, item.GetChildIDs()...)
	}
	children, _, err := s.Steam.GetDetails(ctx, domain.Dedupe(childIDs))
	if err != nil {
		return domain.OrderPlan{}, err
	}

	// Map logical mod IDs to the raw token that carries them, so edges/framework
	// are expressed in raw tokens (what TopoOrder reorders) while matching uses
	// the logical ID.
	present := map[string]bool{}   // mod IDs present
	idToRaw := map[string]string{} // mod ID -> raw token (first occurrence)
	for _, raw := range sm.Mods {
		id := domain.ParseModRef(raw).ID
		if id == "" {
			continue
		}
		if _, ok := idToRaw[id]; !ok {
			idToRaw[id] = raw
			present[id] = true
		}
	}
	providerMods := map[string][]string{}        // itemID -> mod IDs it provides
	modToItem := map[string]steam.WorkshopItem{} // mod ID -> providing item
	record := func(item steam.WorkshopItem) {
		mods := item.Parse().Mods
		providerMods[item.PublishedFileID] = mods
		for _, m := range mods {
			if _, ok := modToItem[m]; !ok {
				modToItem[m] = item
			}
		}
	}
	for _, item := range items {
		record(item)
	}
	for _, c := range children {
		record(c)
	}

	edges := map[string][]string{} // dependent RAW token -> prerequisite RAW tokens
	addEdge := func(dependentID, prereqID string) {
		if dependentID == prereqID || !present[dependentID] || !present[prereqID] {
			return
		}
		edges[idToRaw[dependentID]] = append(edges[idToRaw[dependentID]], idToRaw[prereqID])
	}

	// Workshop-level dependencies: a mod from an item depends on the mods from
	// that item's required children.
	for _, item := range items {
		for _, childID := range item.GetChildIDs() {
			for _, dependent := range providerMods[item.PublishedFileID] {
				for _, prereq := range providerMods[childID] {
					addEdge(dependent, prereq)
				}
			}
		}
	}

	// Optional on-disk enrichment via mod.info require= edges (keyed by mod ID).
	for mod, info := range s.providerFor(profile).Lookup(modIDsOf(sm.Mods)) {
		for _, req := range info.Require {
			addEdge(mod, req)
		}
	}

	framework := map[string]bool{} // keyed by RAW token (TopoOrder iterates raw tokens)
	for _, raw := range sm.Mods {
		id := domain.ParseModRef(raw).ID
		if isFramework(id, modToItem[id]) {
			framework[raw] = true
		}
	}

	return domain.TopoOrder(sm.Mods, edges, framework), nil
}

func isFramework(modID string, item steam.WorkshopItem) bool {
	hay := strings.ToLower(modID + " " + item.Title)
	for _, kw := range domain.FrameworkKeywords {
		if strings.Contains(hay, kw) {
			return true
		}
	}
	return false
}

// modIDsOf returns the logical mod IDs of Mods= tokens (dropping empties).
func modIDsOf(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if id := domain.ParseModRef(t).ID; id != "" {
			out = append(out, id)
		}
	}
	return out
}
