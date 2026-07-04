package domain

import "sort"

// OrderPlan is the result of a load-order suggestion.
type OrderPlan struct {
	Ordered []string          // the suggested load order
	Moved   []string          // mods whose index changed vs the input
	Reasons map[string]string // mod -> human-readable reason it was placed
	Cycles  [][]string        // dependency cycles that prevented full ordering
}

// TopoOrder computes a stable load order for current.
//
// edges[dependent] lists the prerequisites that must load BEFORE it. framework
// marks library/framework mods that should be biased toward the front. The sort
// is a stable Kahn topological sort: among mods that are ready (all prereqs
// placed) it prefers framework mods, then the mod's original position, so the
// result respects dependencies while minimizing churn. Mods in a cycle are
// appended in their original relative order and reported in Cycles.
func TopoOrder(current []string, edges map[string][]string, framework map[string]bool) OrderPlan {
	index := make(map[string]int, len(current))
	present := make(map[string]bool, len(current))
	for i, m := range current {
		index[m] = i
		present[m] = true
	}

	// Build prereq -> dependents adjacency and indegree, ignoring prereqs that
	// aren't in the current set (those are "missing", a validator concern).
	dependents := make(map[string][]string)
	indegree := make(map[string]int, len(current))
	for _, m := range current {
		indegree[m] = 0
	}
	for dep, prereqs := range edges {
		if !present[dep] {
			continue
		}
		for _, pre := range prereqs {
			if !present[pre] || pre == dep {
				continue
			}
			dependents[pre] = append(dependents[pre], dep)
			indegree[dep]++
		}
	}

	reasons := make(map[string]string)
	better := func(a, b string) bool {
		fa, fb := framework[a], framework[b]
		if fa != fb {
			return fa // framework first
		}
		return index[a] < index[b] // otherwise keep original order
	}

	var ready []string
	for _, m := range current {
		if indegree[m] == 0 {
			ready = append(ready, m)
		}
	}

	var ordered []string
	placed := make(map[string]bool, len(current))
	for len(ready) > 0 {
		// Select the best ready node.
		sort.SliceStable(ready, func(i, j int) bool { return better(ready[i], ready[j]) })
		n := ready[0]
		ready = ready[1:]
		if placed[n] {
			continue
		}
		placed[n] = true
		ordered = append(ordered, n)
		if framework[n] {
			reasons[n] = "framework/library loaded early"
		}
		for _, d := range dependents[n] {
			indegree[d]--
			if indegree[d] == 0 {
				ready = append(ready, d)
				reasons[d] = "after its dependencies"
			}
		}
	}

	// Any unplaced nodes are in cycles; append them in original order.
	if len(ordered) < len(current) {
		var leftover []string
		for _, m := range current {
			if !placed[m] {
				leftover = append(leftover, m)
			}
		}
		ordered = append(ordered, leftover...)
	}

	cycles := DetectCycles(edges)

	var moved []string
	for i, m := range ordered {
		if index[m] != i {
			moved = append(moved, m)
		}
	}

	return OrderPlan{Ordered: ordered, Moved: moved, Reasons: reasons, Cycles: cycles}
}

// FrameworkKeywords are substrings that hint a mod is a library/framework and
// should load early. Matching is case-insensitive against mod ID and title.
var FrameworkKeywords = []string{
	"library", "framework", "api", "common", "core", "patch", "compat", "shared", "lib",
}
