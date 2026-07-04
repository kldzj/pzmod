package domain

import "strings"

// ModRef is a parsed Mods= entry. Build 42 entries use "[workshopID]\modID";
// Build 41 entries are a plain "modID" (Workshop == "").
type ModRef struct {
	Workshop string // optional Steam Workshop file ID pinning the provider
	ID       string // the logical mod ID (what Steam descriptions declare)
	Raw      string // the exact original token (preserved for byte-exact writes)
}

// ParseModRef splits a Mods= token on the FIRST backslash into an optional
// workshop pin and the mod ID. No backslash ⇒ a plain mod ID (Workshop == "").
// Workshop and ID are trimmed; Raw keeps the original token verbatim.
func ParseModRef(token string) ModRef {
	t := strings.TrimSpace(token)
	if i := strings.IndexByte(t, '\\'); i >= 0 {
		return ModRef{Workshop: strings.TrimSpace(t[:i]), ID: strings.TrimSpace(t[i+1:]), Raw: token}
	}
	return ModRef{Workshop: "", ID: t, Raw: token}
}

// ModID returns just the logical mod ID of a Mods= token.
func ModID(token string) string { return ParseModRef(token).ID }

// FormatModRef renders a Mods= token. When explicit (a Build 42 profile) and a
// workshop ID is known it returns "workshop\id"; when explicit without a
// workshop it returns "\id"; otherwise (Build 41) it returns plain "id". domain
// stays build-free: callers pass explicit = (build == B42).
func FormatModRef(workshop, id string, explicit bool) string {
	if id == "" {
		return ""
	}
	if explicit {
		if workshop != "" {
			return workshop + `\` + id
		}
		return `\` + id
	}
	return id
}

// refKey is the dedup identity of a token: (workshop, id). "\X" and "X" share
// {"", "X"}; "W1\X" and "W2\X" are distinct.
func refKey(token string) string {
	r := ParseModRef(token)
	return r.Workshop + "\x00" + r.ID
}

// DedupeMods removes duplicate Mods= tokens keyed on (workshop, id), preserving
// first-seen order and the raw token, dropping tokens with an empty mod ID.
func DedupeMods(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if ParseModRef(t).ID == "" {
			continue
		}
		k := refKey(t)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, t)
	}
	return out
}
