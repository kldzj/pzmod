// Package domain holds pure value types and transforms for Project Zomboid mod
// management. It performs no I/O and imports no other internal package, which
// keeps it trivially unit-testable and free of side effects.
package domain

// ServerMods is the mod-relevant slice of a server config: the three lists that
// pzmod manages. They are independent lists, NOT a 1:1 mapping:
//
//   - Mods is the load order (left-to-right; later entries override earlier).
//   - WorkshopItems is the download set (numeric Steam Workshop IDs); a single
//     item can declare multiple mod IDs.
//   - Maps is the Map= value (folder names; commas are part of names, not
//     separators).
type ServerMods struct {
	Mods          []string
	WorkshopItems []string
	Maps          []string
}

// Clone returns a deep copy so callers can mutate without aliasing.
func (s ServerMods) Clone() ServerMods {
	return ServerMods{
		Mods:          cloneSlice(s.Mods),
		WorkshopItems: cloneSlice(s.WorkshopItems),
		Maps:          cloneSlice(s.Maps),
	}
}

// HasMod reports whether the load order enables modID, matching on the logical
// mod ID so "\modID", "modID", and "workshop\modID" all count.
func (s ServerMods) HasMod(modID string) bool {
	for _, t := range s.Mods {
		if ParseModRef(t).ID == modID {
			return true
		}
	}
	return false
}

// HasItem reports whether a Workshop ID is present.
func (s ServerMods) HasItem(id string) bool { return contains(s.WorkshopItems, id) }

// HasMap reports whether a map folder is present.
func (s ServerMods) HasMap(name string) bool { return contains(s.Maps, name) }

// AddMod enables a mod. token is a fully-formed Mods= entry (build it with
// FormatModRef). No-op when a mod with the same logical ID is already enabled in
// any form; otherwise the raw token is appended.
func (s ServerMods) AddMod(token string) ServerMods {
	out := s.Clone()
	id := ParseModRef(token).ID
	if id == "" || out.HasMod(id) {
		return out
	}
	out.Mods = append(out.Mods, token)
	return out
}

// AddItem appends a Workshop ID if absent.
func (s ServerMods) AddItem(id string) ServerMods {
	out := s.Clone()
	if id != "" && !contains(out.WorkshopItems, id) {
		out.WorkshopItems = append(out.WorkshopItems, id)
	}
	return out
}

// AddMap appends a map folder if absent. Custom maps are inserted before the
// first vanilla base map (so base maps stay last and custom tiles win); base
// maps are appended at the end.
func (s ServerMods) AddMap(name string) ServerMods {
	out := s.Clone()
	if name == "" || contains(out.Maps, name) {
		return out
	}
	if IsBaseMap(name) {
		out.Maps = append(out.Maps, name)
		return out
	}
	idx := len(out.Maps)
	for i, m := range out.Maps {
		if IsBaseMap(m) {
			idx = i
			break
		}
	}
	// Safe insert: the inner append copies the tail into a fresh slice before the
	// outer append can overwrite it.
	out.Maps = append(out.Maps[:idx], append([]string{name}, out.Maps[idx:]...)...)
	return out
}

// RemoveMod disables a mod by logical ID, dropping every Mods= token whose ID
// matches (arg may be a bare ID or a full token).
func (s ServerMods) RemoveMod(arg string) ServerMods {
	id := ParseModRef(arg).ID
	out := s.Clone()
	kept := out.Mods[:0:0]
	for _, t := range out.Mods {
		if ParseModRef(t).ID != id {
			kept = append(kept, t)
		}
	}
	out.Mods = kept
	return out
}

// RemoveItem drops a Workshop ID.
func (s ServerMods) RemoveItem(id string) ServerMods {
	out := s.Clone()
	out.WorkshopItems = remove(out.WorkshopItems, id)
	return out
}

// RemoveMap drops a map folder.
func (s ServerMods) RemoveMap(name string) ServerMods {
	out := s.Clone()
	out.Maps = remove(out.Maps, name)
	return out
}

// WithMods returns a copy whose load order is replaced (deduped by mod ref).
func (s ServerMods) WithMods(mods []string) ServerMods {
	out := s.Clone()
	out.Mods = DedupeMods(mods)
	return out
}

func cloneSlice(s []string) []string {
	if s == nil {
		return nil
	}
	return append([]string(nil), s...)
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func remove(s []string, v string) []string {
	out := s[:0:0]
	for _, x := range s {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// Dedupe returns s with duplicates removed, preserving first-seen order and
// dropping empty strings.
func Dedupe(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	out := make([]string, 0, len(s))
	for _, x := range s {
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}
