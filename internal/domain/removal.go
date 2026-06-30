package domain

// ModDecl is what a single Workshop item declares (its mod IDs and map folders).
type ModDecl struct {
	Mods []string
	Maps []string
}

// RemovalPlan lists what removing an item should take with it.
type RemovalPlan struct {
	Item string
	Mods []string
	Maps []string
}

// PlanRemoval computes the mods/maps to drop when removing target. It removes
// only the mods/maps the target uniquely owns among the supplied declarations:
// anything also declared by another *installed* item, or not declared by any
// item at all (manually added), is left in place.
func PlanRemoval(target string, declarations map[string]ModDecl, current ServerMods) RemovalPlan {
	plan := RemovalPlan{Item: target}
	td, ok := declarations[target]
	if !ok {
		return plan
	}
	installed := make(map[string]struct{}, len(current.WorkshopItems))
	for _, id := range current.WorkshopItems {
		installed[id] = struct{}{}
	}
	ownedByOther := func(pick func(ModDecl) []string, value string) bool {
		for id, d := range declarations {
			if id == target {
				continue
			}
			if _, still := installed[id]; !still {
				continue
			}
			if contains(pick(d), value) {
				return true
			}
		}
		return false
	}
	for _, m := range td.Mods {
		if current.HasMod(m) && !ownedByOther(func(d ModDecl) []string { return d.Mods }, m) {
			plan.Mods = append(plan.Mods, m)
		}
	}
	for _, mp := range td.Maps {
		if current.HasMap(mp) && !ownedByOther(func(d ModDecl) []string { return d.Maps }, mp) {
			plan.Maps = append(plan.Maps, mp)
		}
	}
	return plan
}

// Apply returns current with the plan's item, mods, and maps removed.
func (p RemovalPlan) Apply(current ServerMods) ServerMods {
	out := current.RemoveItem(p.Item)
	for _, m := range p.Mods {
		out = out.RemoveMod(m)
	}
	for _, mp := range p.Maps {
		out = out.RemoveMap(mp)
	}
	return out
}
