// Package build models Project Zomboid build awareness (B41 vs B42). It stays
// intentionally modest: a Build type, the relevant Workshop tags, and a pure
// function that produces compatibility warnings. It never branches parsing or
// blocks operations.
package build

import (
	"strings"

	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/pkg/steam"
)

// Build identifies a Project Zomboid version line.
type Build string

const (
	// Unknown means the profile has not declared a build.
	Unknown Build = ""
	// B41 is the stable line with mature multiplayer.
	B41 Build = "b41"
	// B42 is the (as of writing) unstable line; its multiplayer still disables mods.
	B42 Build = "b42"
)

// Parse normalizes a free-form build string.
func Parse(s string) Build {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "b41", "41", "41.78", "stable":
		return B41
	case "b42", "42", "unstable":
		return B42
	default:
		return Unknown
	}
}

// Label returns a human-readable build name.
func (b Build) Label() string {
	switch b {
	case B41:
		return "Build 41"
	case B42:
		return "Build 42"
	default:
		return "Unspecified"
	}
}

// Workshop tag names used to infer per-item build support.
const (
	tagB41 = "Build 41"
	tagB42 = "Build 42"
)

// itemTags lowercases an item's tag set for matching.
func itemTags(item steam.WorkshopItem) map[string]bool {
	tags := make(map[string]bool, len(item.Tags))
	for _, t := range item.Tags {
		tags[strings.ToLower(t.Tag)] = true
	}
	return tags
}

// CompatWarnings returns findings for items whose Workshop tags suggest they do
// not support the profile's build. When the profile build is Unknown, or an item
// carries no build tags, no warning is produced (we avoid false positives).
func CompatWarnings(b Build, items []steam.WorkshopItem) []domain.Finding {
	if b == Unknown {
		return nil
	}
	var findings []domain.Finding
	for _, item := range items {
		tags := itemTags(item)
		has41 := tags[strings.ToLower(tagB41)]
		has42 := tags[strings.ToLower(tagB42)]
		if !has41 && !has42 {
			continue // no build tags: can't judge
		}
		supported := (b == B41 && has41) || (b == B42 && has42)
		if !supported {
			findings = append(findings, domain.Finding{
				Severity: domain.SeverityWarning,
				Code:     domain.CodeBuildCompat,
				Subject:  item.PublishedFileID,
				Message:  itemTitle(item) + " is not tagged for " + b.Label(),
			})
		}
	}
	return findings
}

func itemTitle(item steam.WorkshopItem) string {
	if item.Title != "" {
		return item.Title
	}
	return item.PublishedFileID
}
