package domain

// Delta summarizes how one ordered list changed.
type Delta struct {
	Added     int
	Removed   int
	Reordered bool
}

// Empty reports whether nothing changed.
func (d Delta) Empty() bool { return d.Added == 0 && d.Removed == 0 && !d.Reordered }

// ListDelta compares two ordered lists by set membership and order. Reordered is
// true only when the elements common to both appear in a different relative order.
func ListDelta(old, nw []string) Delta {
	oldSet := make(map[string]struct{}, len(old))
	for _, v := range old {
		oldSet[v] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(nw))
	for _, v := range nw {
		newSet[v] = struct{}{}
	}
	var d Delta
	for _, v := range nw {
		if _, ok := oldSet[v]; !ok {
			d.Added++
		}
	}
	for _, v := range old {
		if _, ok := newSet[v]; !ok {
			d.Removed++
		}
	}
	// Compare relative order of the intersection.
	var oc, nc []string
	for _, v := range old {
		if _, ok := newSet[v]; ok {
			oc = append(oc, v)
		}
	}
	for _, v := range nw {
		if _, ok := oldSet[v]; ok {
			nc = append(nc, v)
		}
	}
	for i := range oc {
		if oc[i] != nc[i] {
			d.Reordered = true
			break
		}
	}
	return d
}
