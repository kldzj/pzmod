// Package modinfo optionally enriches load-order analysis by reading mod.info
// files from a downloaded Workshop content directory. It is absent-safe: when
// the content path doesn't exist, the Provider simply returns nothing, so the
// tool works whether or not it runs on the same machine as the server.
package modinfo

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ModInfo is the subset of a mod.info file relevant to load order.
type ModInfo struct {
	ID      string   // the "id=" / "name=" field used in the Mods= list
	Name    string   // human-friendly name
	Require []string // mod IDs this mod requires (must load first)
}

// Provider resolves mod.info data for installed mods.
type Provider interface {
	// Lookup returns mod.info for the given mod IDs found under the content
	// root, keyed by mod ID. Missing mods are simply absent from the map.
	Lookup(modIDs []string) map[string]ModInfo
}

// DiskProvider reads mod.info files under a Workshop content root of the form
// <root>/<workshopID>/mods/<modName>/mod.info.
type DiskProvider struct {
	Root string
}

// NewProvider returns a Provider for root. An empty root yields a no-op
// provider so callers need not branch.
func NewProvider(root string) Provider {
	if strings.TrimSpace(root) == "" {
		return nopProvider{}
	}
	return &DiskProvider{Root: root}
}

type nopProvider struct{}

func (nopProvider) Lookup([]string) map[string]ModInfo { return map[string]ModInfo{} }

// Lookup scans the content root for mod.info files and returns those whose id
// is among the requested mod IDs.
func (p *DiskProvider) Lookup(modIDs []string) map[string]ModInfo {
	want := make(map[string]bool, len(modIDs))
	for _, id := range modIDs {
		want[id] = true
	}

	out := make(map[string]ModInfo)
	infos, err := p.scan()
	if err != nil {
		return out
	}
	for _, mi := range infos {
		if want[mi.ID] {
			out[mi.ID] = mi
		}
	}
	return out
}

// scan walks <root>/*/mods/*/mod.info.
func (p *DiskProvider) scan() ([]ModInfo, error) {
	matches, err := filepath.Glob(filepath.Join(p.Root, "*", "mods", "*", "mod.info"))
	if err != nil {
		return nil, err
	}
	// Some installs nest under "Contents/mods"; include that layout too.
	more, _ := filepath.Glob(filepath.Join(p.Root, "*", "*", "mods", "*", "mod.info"))
	matches = append(matches, more...)

	var out []ModInfo
	for _, path := range matches {
		if mi, ok := parseFile(path); ok {
			out = append(out, mi)
		}
	}
	return out, nil
}

func parseFile(path string) (ModInfo, bool) {
	f, err := os.Open(path)
	if err != nil {
		return ModInfo{}, false
	}
	defer f.Close()

	var mi ModInfo
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, val, ok := splitKV(scanner.Text())
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "id":
			mi.ID = val
		case "name":
			mi.Name = val
			if mi.ID == "" {
				mi.ID = val // PZ uses the name field as the load ID when id is absent
			}
		case "require":
			for _, r := range strings.Split(val, ",") {
				if r = strings.TrimSpace(r); r != "" {
					mi.Require = append(mi.Require, r)
				}
			}
		}
	}
	if mi.ID == "" {
		return ModInfo{}, false
	}
	return mi, true
}

func splitKV(line string) (key, val string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	i := strings.IndexByte(line, '=')
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}
