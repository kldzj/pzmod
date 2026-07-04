package domain

// baseMaps are the vanilla Project Zomboid map entries. Custom maps should
// precede these so their tiles take precedence; the base map(s) belong last.
var baseMaps = map[string]struct{}{
	"Muldraugh, KY":  {},
	"West Point, KY": {},
	"Riverside, KY":  {},
	"Rosewood, KY":   {},
}

// IsBaseMap reports whether name is a vanilla base map.
func IsBaseMap(name string) bool {
	_, ok := baseMaps[name]
	return ok
}

// SuggestMapOrder returns a load order with vanilla base maps moved to the end,
// preserving the relative order of everything else (minimal churn).
func SuggestMapOrder(maps []string) OrderPlan {
	var custom, bases []string
	for _, m := range maps {
		if IsBaseMap(m) {
			bases = append(bases, m)
		} else {
			custom = append(custom, m)
		}
	}
	ordered := append(append([]string(nil), custom...), bases...)

	// Build a map of original indices.
	idx := make(map[string]int, len(maps))
	for i, m := range maps {
		idx[m] = i
	}

	// Compute Moved as the set of base maps whose index changed.
	// Moved lists only base maps we deliberately sent to the end. Custom maps that drift forward as a side-effect are not flagged, unlike the generic OrderPlan.Moved.
	var moved []string
	for i, m := range ordered {
		if IsBaseMap(m) && idx[m] != i {
			moved = append(moved, m)
		}
	}

	// Set reasons for all base maps.
	// Every base map gets a reason, even if it was already last (stable intent).
	reasons := make(map[string]string, len(bases))
	for _, m := range bases {
		reasons[m] = "vanilla base map - kept last so custom maps win"
	}

	return OrderPlan{Ordered: ordered, Moved: moved, Reasons: reasons}
}
