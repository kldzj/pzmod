package store

import (
	"encoding/json"
	"os"
	"strings"
)

// envAPIKey lets the Steam API key be supplied through the environment, which is
// the natural way to pass it to a container or CI job without mounting a file.
const envAPIKey = "PZMOD_STEAM_KEY"

// credentials is the on-disk credential file (mode 0600). The Steam API key is
// never stored in profiles.json.
type credentials struct {
	Global   string            `json:"global,omitempty"`
	Profiles map[string]string `json:"profiles,omitempty"`
}

func (s *Store) loadCredentials() (credentials, error) {
	var c credentials
	data, err := os.ReadFile(s.credentialsPath())
	if os.IsNotExist(err) {
		return credentials{}, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

func (s *Store) saveCredentials(c credentials) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.credentialsPath(), data, 0600)
}

// APIKey resolves the key for a profile, in order: a per-profile override, the
// PZMOD_STEAM_KEY environment variable, then the stored global key. profileID may
// be "" to skip the per-profile lookup.
func (s *Store) APIKey(profileID string) (string, error) {
	c, err := s.loadCredentials()
	if err != nil {
		return "", err
	}
	if profileID != "" {
		if k := c.Profiles[profileID]; k != "" {
			return k, nil
		}
	}
	if k := strings.TrimSpace(os.Getenv(envAPIKey)); k != "" {
		return k, nil
	}
	if c.Global == "" {
		return "", ErrNoKey
	}
	return c.Global, nil
}

// HasAPIKey reports whether any usable key is configured for the profile.
func (s *Store) HasAPIKey(profileID string) bool {
	k, err := s.APIKey(profileID)
	return err == nil && k != ""
}

// SetGlobalKey stores the global Steam API key.
func (s *Store) SetGlobalKey(key string) error {
	c, err := s.loadCredentials()
	if err != nil {
		return err
	}
	c.Global = strings.TrimSpace(key)
	return s.saveCredentials(c)
}

// SetProfileKey stores a per-profile key override.
func (s *Store) SetProfileKey(profileID, key string) error {
	c, err := s.loadCredentials()
	if err != nil {
		return err
	}
	if c.Profiles == nil {
		c.Profiles = map[string]string{}
	}
	c.Profiles[profileID] = strings.TrimSpace(key)
	return s.saveCredentials(c)
}

// ClearKey removes the global key (profileID == "") or a profile override.
func (s *Store) ClearKey(profileID string) error {
	c, err := s.loadCredentials()
	if err != nil {
		return err
	}
	if profileID == "" {
		c.Global = ""
	} else {
		delete(c.Profiles, profileID)
	}
	return s.saveCredentials(c)
}

// migrateLegacyKey copies a legacy ~/.pzmod plaintext key into the new
// credentials file once, when no global key is set yet. The old file is left
// untouched. The migration is idempotent.
func (s *Store) migrateLegacyKey() error {
	c, err := s.loadCredentials()
	if err != nil {
		return err
	}
	if c.Global != "" {
		return nil // already have a key; nothing to do
	}

	legacy, err := legacyKeyPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(legacy)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return nil
	}
	c.Global = key
	return s.saveCredentials(c)
}
