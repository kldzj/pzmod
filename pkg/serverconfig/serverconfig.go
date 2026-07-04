// Package serverconfig is the Project Zomboid-aware wrapper over a byte-exact
// ini.Document. It exposes typed accessors for the mod lists and common server
// info while leaving all formatting fidelity to the underlying document.
package serverconfig

import (
	"os"
	"strings"

	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/pkg/ini"
)

// Config wraps an ini.Document with PZ semantics and a backing file path.
type Config struct {
	doc  *ini.Document
	path string
}

// Load reads and parses the servertest.ini at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &Config{doc: ini.Parse(data), path: path}, nil
}

// FromBytes parses raw bytes, associating the given path (which need not exist).
func FromBytes(path string, data []byte) *Config {
	return &Config{doc: ini.Parse(data), path: path}
}

// Path returns the backing file path.
func (c *Config) Path() string { return c.path }

// Document exposes the underlying document (e.g. to render a diff).
func (c *Config) Document() *ini.Document { return c.doc }

// Get returns a raw key value and whether it was present.
func (c *Config) Get(key string) (string, bool) { return c.doc.Get(key) }

// GetOr returns a raw key value or def.
func (c *Config) GetOr(key, def string) string { return c.doc.GetOr(key, def) }

// Set writes a raw key value.
func (c *Config) Set(key, value string) { c.doc.Set(key, value) }

// Bytes renders the config to bytes.
func (c *Config) Bytes() []byte { return c.doc.Bytes() }

// String renders the config to a string.
func (c *Config) String() string { return c.doc.String() }

// HasUnsavedChanges reports in-memory edits not yet persisted.
func (c *Config) HasUnsavedChanges() bool { return c.doc.HasUnsavedChanges() }

// Save writes the config back to its path (0644) and resets the dirty state.
func (c *Config) Save() error { return c.SaveTo(c.path) }

// SaveTo writes the config to path (0644). When path is the backing path it
// resets the dirty baseline.
func (c *Config) SaveTo(path string) error {
	if err := os.WriteFile(path, c.doc.Bytes(), 0644); err != nil {
		return err
	}
	if path == c.path {
		c.doc.MarkSaved()
	}
	return nil
}

// --- Mod lists ---------------------------------------------------------------

// Mods returns the load-order list. Values are ';'-separated and may contain
// stray ','-separated groups; both are split, trimmed, and deduped.
func (c *Config) Mods() []string {
	return splitFixed(c.doc.GetOr(KeyMods, ""))
}

// WorkshopItems returns the download set, parsed like Mods.
func (c *Config) WorkshopItems() []string {
	return splitFixed(c.doc.GetOr(KeyWorkshop, ""))
}

// Maps returns the Map= folders. Only ';' separates them: commas are part of
// map names (e.g. "Muldraugh, KY").
func (c *Config) Maps() []string {
	return splitMaps(c.doc.GetOr(KeyMap, ""))
}

// SetMods replaces the load order (deduped by mod ref, ';'-joined).
func (c *Config) SetMods(mods []string) {
	c.doc.Set(KeyMods, strings.Join(domain.DedupeMods(mods), listSep))
}

// SetWorkshopItems replaces the download set.
func (c *Config) SetWorkshopItems(ids []string) {
	c.doc.Set(KeyWorkshop, strings.Join(domain.Dedupe(ids), listSep))
}

// SetMaps replaces the Map= value.
func (c *Config) SetMaps(maps []string) {
	c.doc.Set(KeyMap, strings.Join(domain.Dedupe(maps), listSep))
}

// ServerMods extracts the three lists as a domain value.
func (c *Config) ServerMods() domain.ServerMods {
	return domain.ServerMods{
		Mods:          c.Mods(),
		WorkshopItems: c.WorkshopItems(),
		Maps:          c.Maps(),
	}
}

// ApplyServerMods writes the three lists back to the document.
func (c *Config) ApplyServerMods(m domain.ServerMods) {
	c.SetMods(m.Mods)
	c.SetWorkshopItems(m.WorkshopItems)
	c.SetMaps(m.Maps)
}

// --- Server info -------------------------------------------------------------

// Name returns the public server name.
func (c *Config) Name() string { return c.doc.GetOr(KeyName, "") }

// SetName sets the public server name.
func (c *Config) SetName(v string) { c.doc.Set(KeyName, v) }

// Description returns the public description (raw, with PZ's <LINE> tokens).
func (c *Config) Description() string { return c.doc.GetOr(KeyDescription, "") }

// SetDescription sets the public description (raw).
func (c *Config) SetDescription(v string) { c.doc.Set(KeyDescription, v) }

// Public reports whether the server is listed publicly.
func (c *Config) Public() bool { return c.doc.GetOr(KeyPublic, "false") == "true" }

// SetPublic sets the public-listing flag.
func (c *Config) SetPublic(v bool) {
	if v {
		c.doc.Set(KeyPublic, "true")
	} else {
		c.doc.Set(KeyPublic, "false")
	}
}

// Password returns the server password.
func (c *Config) Password() string { return c.doc.GetOr(KeyPassword, "") }

// SetPassword sets the server password.
func (c *Config) SetPassword(v string) { c.doc.Set(KeyPassword, v) }

// MaxPlayers returns the configured slot count (raw string).
func (c *Config) MaxPlayers() string { return c.doc.GetOr(KeyMaxPlayers, "") }

// SetMaxPlayers sets the slot count.
func (c *Config) SetMaxPlayers(v string) { c.doc.Set(KeyMaxPlayers, v) }

// --- helpers -----------------------------------------------------------------

// splitFixed splits on ';' and ',', trims, drops empties, and dedupes.
func splitFixed(value string) []string {
	var out []string
	for _, group := range strings.Split(value, ";") {
		for _, id := range strings.Split(group, ",") {
			if id = strings.TrimSpace(id); id != "" {
				out = append(out, id)
			}
		}
	}
	return domain.Dedupe(out)
}

// splitMaps splits on ';' only (commas belong to map names).
func splitMaps(value string) []string {
	var out []string
	for _, name := range strings.Split(value, ";") {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, name)
		}
	}
	return domain.Dedupe(out)
}

// --- Summary -----------------------------------------------------------------

// Summary describes how one config differs from another.
type Summary struct {
	Mods          domain.Delta
	WorkshopItems domain.Delta
	Maps          domain.Delta
	ChangedFields []string // human labels of changed scalar keys
}

// Empty reports whether nothing changed.
func (s Summary) Empty() bool {
	return s.Mods.Empty() && s.WorkshopItems.Empty() && s.Maps.Empty() && len(s.ChangedFields) == 0
}

// Summarize compares the managed lists and scalar fields of old vs new.
func Summarize(old, new *Config) Summary {
	s := Summary{
		Mods:          domain.ListDelta(old.Mods(), new.Mods()),
		WorkshopItems: domain.ListDelta(old.WorkshopItems(), new.WorkshopItems()),
		Maps:          domain.ListDelta(old.Maps(), new.Maps()),
	}
	type field struct {
		label string
		get   func(*Config) string
	}
	fields := []field{
		{"Server name", func(c *Config) string { return c.Name() }},
		{"Description", func(c *Config) string { return c.Description() }},
		{"Password", func(c *Config) string { return c.Password() }},
		{"Max players", func(c *Config) string { return c.MaxPlayers() }},
		{"Public listing", func(c *Config) string { return c.GetOr(KeyPublic, "") }},
	}
	for _, f := range fields {
		if f.get(old) != f.get(new) {
			s.ChangedFields = append(s.ChangedFields, f.label)
		}
	}
	return s
}
