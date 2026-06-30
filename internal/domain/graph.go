package domain

import "sort"

// DetectCycles finds dependency cycles in a directed graph expressed as
// edges[node] = nodes it depends on. Each returned slice is the set of nodes
// participating in one strongly-connected cycle. Used to report (not fail on)
// circular mod dependencies.
func DetectCycles(edges map[string][]string) [][]string {
	const (
		white = 0 // unvisited
		gray  = 1 // on the current stack
		black = 2 // done
	)
	color := make(map[string]int)
	var cycles [][]string
	var stack []string

	// Stable node iteration order for deterministic output.
	nodes := sortedKeys(edges)

	var visit func(n string)
	visit = func(n string) {
		color[n] = gray
		stack = append(stack, n)
		for _, dep := range edges[n] {
			switch color[dep] {
			case white:
				visit(dep)
			case gray:
				// Found a back-edge: extract the cycle from the stack.
				cycles = append(cycles, extractCycle(stack, dep))
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = black
	}

	for _, n := range nodes {
		if color[n] == white {
			visit(n)
		}
	}
	return cycles
}

func extractCycle(stack []string, start string) []string {
	for i, n := range stack {
		if n == start {
			return append([]string(nil), stack[i:]...)
		}
	}
	return append([]string(nil), start)
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
