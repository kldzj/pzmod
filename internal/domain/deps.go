package domain

// MultiModItem is a Workshop item that declares more than one mod ID. The user
// may not want all of them enabled, so the presentation layer surfaces these.
type MultiModItem struct {
	ItemID string
	ModIDs []string
}

// Plan is the outcome of dependency resolution: the additions needed to satisfy
// a set of seed items plus everything they (transitively) require.
type Plan struct {
	AddWorkshopItems []string          // Workshop IDs to add (excludes collections)
	AddMods          []string          // mod IDs to add to the load order
	AddMaps          []string          // map folders to add
	Missing          []string          // required IDs that could not be fetched
	MultiMod         []MultiModItem    // items declaring multiple mod IDs
	NoModID          []string          // content items with no parseable mod ID
	Cycles           [][]string        // dependency cycles detected during closure
	AddModSources    map[string]string // mod ID -> Workshop ID that provides it
}

// Empty reports whether the plan adds nothing.
func (p Plan) Empty() bool {
	return len(p.AddWorkshopItems) == 0 && len(p.AddMods) == 0 && len(p.AddMaps) == 0
}

// HasItem reports whether the plan adds the given Workshop ID.
func (p Plan) HasItem(id string) bool {
	return contains(p.AddWorkshopItems, id)
}

// Apply returns sm with the plan's additions merged in. explicit selects the
// Build 42 pinned write form ("workshop\modID"); when false, plain mod IDs are
// written (Build 41). Mod tokens are formatted via FormatModRef using the
// provider recorded in AddModSources.
func (p Plan) Apply(sm ServerMods, explicit bool) ServerMods {
	out := sm.Clone()
	for _, id := range p.AddWorkshopItems {
		out = out.AddItem(id)
	}
	for _, m := range p.AddMods {
		out = out.AddMod(FormatModRef(p.AddModSources[m], m, explicit))
	}
	for _, mp := range p.AddMaps {
		out = out.AddMap(mp)
	}
	return out
}
