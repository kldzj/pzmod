package service

import (
	"context"

	"github.com/kldzj/pzmod/internal/domain"
	"github.com/kldzj/pzmod/internal/steam"
)

// ResolvePlan is a dependency-resolution Plan plus the fetched items, so the
// presentation layer can show titles and sizes without re-fetching.
type ResolvePlan struct {
	domain.Plan
	Items map[string]steam.WorkshopItem
}

// Resolve computes the transitive closure of seeds (Workshop IDs the user wants
// to add, or the installed set when checking for missing deps) and returns the
// additions needed beyond what is already installed.
//
// Collections are expanded for their members but never added to WorkshopItems;
// classification is by the FETCHED item's own file type, so a collection nested
// in a collection is still expanded, not installed. A visited set bounds the
// BFS so cyclic dependencies terminate; cycles are reported, not failed.
func (s *Services) Resolve(ctx context.Context, seeds []string, installed domain.ServerMods) (ResolvePlan, error) {
	items := map[string]steam.WorkshopItem{}
	edges := map[string][]string{} // itemID -> child IDs (for cycle detection)

	installedItems := toSet(installed.WorkshopItems)
	installedMods := modIDSet(installed.Mods)
	installedMaps := toSet(installed.Maps)

	var plan domain.Plan
	plan.AddModSources = map[string]string{}
	addItem := newOrderedSet()
	addMod := newOrderedSet()
	addMap := newOrderedSet()
	missing := newOrderedSet()

	visited := map[string]bool{}
	var frontier []string
	for _, id := range domain.Dedupe(seeds) {
		if !visited[id] {
			visited[id] = true
			frontier = append(frontier, id)
		}
	}

	for len(frontier) > 0 {
		fetched, miss, err := s.Steam.GetDetails(ctx, frontier)
		if err != nil {
			return ResolvePlan{}, err
		}
		for _, m := range miss {
			missing.add(m)
		}

		var next []string
		enqueue := func(ids []string) {
			for _, id := range ids {
				if id != "" && !visited[id] {
					visited[id] = true
					next = append(next, id)
				}
			}
		}

		for _, item := range fetched {
			items[item.PublishedFileID] = item
			children := item.GetChildIDs()
			edges[item.PublishedFileID] = children

			if item.IsCollection() {
				enqueue(children) // expand members; do NOT install the collection
				continue
			}

			// Content item: install it (unless already present) and parse mods.
			if !installedItems[item.PublishedFileID] {
				addItem.add(item.PublishedFileID)
			}
			parsed := item.Parse()
			switch len(parsed.Mods) {
			case 0:
				plan.NoModID = append(plan.NoModID, item.PublishedFileID)
			case 1:
				// single mod id, nothing special
			default:
				plan.MultiMod = append(plan.MultiMod, domain.MultiModItem{
					ItemID: item.PublishedFileID,
					ModIDs: parsed.Mods,
				})
			}
			for _, mod := range parsed.Mods {
				if !installedMods[mod] {
					addMod.add(mod)
				}
				if _, ok := plan.AddModSources[mod]; !ok {
					plan.AddModSources[mod] = item.PublishedFileID
				}
			}
			for _, mp := range parsed.Maps {
				if !installedMaps[mp] {
					addMap.add(mp)
				}
			}
			enqueue(children) // transitive dependencies
		}
		frontier = next
	}

	plan.AddWorkshopItems = addItem.slice()
	plan.AddMods = addMod.slice()
	plan.AddMaps = addMap.slice()
	plan.Missing = missing.slice()
	plan.Cycles = domain.DetectCycles(edges)

	return ResolvePlan{Plan: plan, Items: items}, nil
}

func toSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

// modIDSet is toSet over the logical mod IDs of Mods= tokens.
func modIDSet(tokens []string) map[string]bool {
	m := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		if id := domain.ParseModRef(t).ID; id != "" {
			m[id] = true
		}
	}
	return m
}

// orderedSet preserves insertion order while deduping.
type orderedSet struct {
	seen  map[string]bool
	order []string
}

func newOrderedSet() *orderedSet { return &orderedSet{seen: map[string]bool{}} }

func (o *orderedSet) add(v string) {
	if v == "" || o.seen[v] {
		return
	}
	o.seen[v] = true
	o.order = append(o.order, v)
}

func (o *orderedSet) slice() []string { return o.order }
