// Package store persists pzmod's state under the user config directory
// (~/.config/pzmod on Linux, %AppData%/pzmod on Windows): server profiles,
// the Steam API key, and timestamped config backups. It also migrates the
// legacy ~/.pzmod plaintext key on first run.
package store

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

var (
	// ErrNoProfile is returned when a profile ID is not found.
	ErrNoProfile = errors.New("profile not found")
	// ErrNoBackup is returned when a backup ID is not found.
	ErrNoBackup = errors.New("backup not found")
	// ErrNoKey is returned when no Steam API key is configured.
	ErrNoKey = errors.New("no Steam API key set")
)

// Store is the on-disk state manager. Construct it with New.
type Store struct {
	root string
	now  func() time.Time
}

// Option configures a Store.
type Option func(*Store)

// WithRoot overrides the config root directory (used in tests).
func WithRoot(root string) Option { return func(s *Store) { s.root = root } }

// WithClock overrides the clock used for backup timestamps.
func WithClock(now func() time.Time) Option { return func(s *Store) { s.now = now } }

// New resolves the config root, creates it, and runs one-time legacy migration.
func New(opts ...Option) (*Store, error) {
	s := &Store{now: time.Now}
	for _, o := range opts {
		o(s)
	}
	if s.root == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		s.root = filepath.Join(dir, "pzmod")
	}
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return nil, err
	}
	if err := s.migrateLegacyKey(); err != nil {
		return nil, err
	}
	return s, nil
}

// Root returns the config root directory.
func (s *Store) Root() string { return s.root }

func (s *Store) profilesPath() string    { return filepath.Join(s.root, "profiles.json") }
func (s *Store) credentialsPath() string { return filepath.Join(s.root, "credentials.json") }
func (s *Store) backupsRoot() string     { return filepath.Join(s.root, "backups") }

// legacyKeyPath returns the v2 plaintext key location (~/.pzmod).
func legacyKeyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pzmod"), nil
}
