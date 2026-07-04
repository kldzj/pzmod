package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Profile is a managed server config: a named pointer to a servertest.ini plus
// optional metadata. The Steam API key is never stored here (see credentials).
type Profile struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	IniPath             string `json:"ini_path"`
	Build               string `json:"build,omitempty"` // "b41" | "b42" | ""
	WorkshopContentPath string `json:"workshop_content_path,omitempty"`
	BackupRetention     int    `json:"backup_retention,omitempty"`
}

type profilesFile struct {
	Profiles  []Profile `json:"profiles"`
	DefaultID string    `json:"default_id,omitempty"`
}

func (s *Store) loadProfiles() (profilesFile, error) {
	var pf profilesFile
	data, err := os.ReadFile(s.profilesPath())
	if os.IsNotExist(err) {
		return profilesFile{}, nil
	}
	if err != nil {
		return pf, err
	}
	if err := json.Unmarshal(data, &pf); err != nil {
		return pf, err
	}
	return pf, nil
}

func (s *Store) saveProfiles(pf profilesFile) error {
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.profilesPath(), data, 0644)
}

// Profiles returns all configured profiles.
func (s *Store) Profiles() ([]Profile, error) {
	pf, err := s.loadProfiles()
	if err != nil {
		return nil, err
	}
	return pf.Profiles, nil
}

// Profile returns the profile with the given ID.
func (s *Store) Profile(id string) (Profile, error) {
	pf, err := s.loadProfiles()
	if err != nil {
		return Profile{}, err
	}
	for _, p := range pf.Profiles {
		if p.ID == id {
			return p, nil
		}
	}
	return Profile{}, ErrNoProfile
}

// DefaultProfile returns the default profile: the explicitly chosen one, or the
// only profile if there is exactly one.
func (s *Store) DefaultProfile() (Profile, error) {
	pf, err := s.loadProfiles()
	if err != nil {
		return Profile{}, err
	}
	if pf.DefaultID != "" {
		for _, p := range pf.Profiles {
			if p.ID == pf.DefaultID {
				return p, nil
			}
		}
	}
	if len(pf.Profiles) == 1 {
		return pf.Profiles[0], nil
	}
	return Profile{}, ErrNoProfile
}

// AddProfile stores a profile, assigning a unique ID from its name when unset.
// The first profile added becomes the default. The stored profile is returned.
func (s *Store) AddProfile(p Profile) (Profile, error) {
	pf, err := s.loadProfiles()
	if err != nil {
		return Profile{}, err
	}
	if abs, err := filepath.Abs(p.IniPath); err == nil {
		p.IniPath = abs
	}
	if p.ID == "" {
		p.ID = uniqueID(slugify(p.Name), pf.Profiles)
	}
	for _, existing := range pf.Profiles {
		if existing.ID == p.ID {
			return Profile{}, fmt.Errorf("profile %q already exists", p.ID)
		}
	}
	pf.Profiles = append(pf.Profiles, p)
	if pf.DefaultID == "" {
		pf.DefaultID = p.ID
	}
	if err := s.saveProfiles(pf); err != nil {
		return Profile{}, err
	}
	return p, nil
}

// UpdateProfile replaces an existing profile by ID.
func (s *Store) UpdateProfile(p Profile) error {
	pf, err := s.loadProfiles()
	if err != nil {
		return err
	}
	for i := range pf.Profiles {
		if pf.Profiles[i].ID == p.ID {
			pf.Profiles[i] = p
			return s.saveProfiles(pf)
		}
	}
	return ErrNoProfile
}

// RemoveProfile deletes a profile and fixes up the default if needed.
func (s *Store) RemoveProfile(id string) error {
	pf, err := s.loadProfiles()
	if err != nil {
		return err
	}
	idx := -1
	for i, p := range pf.Profiles {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrNoProfile
	}
	pf.Profiles = append(pf.Profiles[:idx], pf.Profiles[idx+1:]...)
	if pf.DefaultID == id {
		pf.DefaultID = ""
		if len(pf.Profiles) > 0 {
			pf.DefaultID = pf.Profiles[0].ID
		}
	}
	return s.saveProfiles(pf)
}

// SetDefaultProfile marks a profile as the default.
func (s *Store) SetDefaultProfile(id string) error {
	pf, err := s.loadProfiles()
	if err != nil {
		return err
	}
	for _, p := range pf.Profiles {
		if p.ID == id {
			pf.DefaultID = id
			return s.saveProfiles(pf)
		}
	}
	return ErrNoProfile
}

// EphemeralProfileID derives a stable backup key for an ad-hoc --file path so
// its backups never collide with named profiles or each other.
func EphemeralProfileID(path string) string {
	abs := path
	if a, err := filepath.Abs(path); err == nil {
		abs = a
	}
	sum := sha256.Sum256([]byte(abs))
	return "file-" + hex.EncodeToString(sum[:])[:12]
}

func slugify(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "profile"
	}
	return out
}

func uniqueID(base string, existing []Profile) string {
	taken := make(map[string]bool, len(existing))
	for _, p := range existing {
		taken[p.ID] = true
	}
	if !taken[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !taken[candidate] {
			return candidate
		}
	}
}
