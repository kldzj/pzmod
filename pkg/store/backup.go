package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// BackupEntry is the metadata for a single config snapshot.
type BackupEntry struct {
	ID        string `json:"id"`        // sortable timestamp stem, also the lookup key
	File      string `json:"file"`      // filename within the profile backup dir
	Timestamp string `json:"timestamp"` // RFC3339 UTC
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	Note      string `json:"note,omitempty"`
	Kind      string `json:"kind"` // "auto" | "manual" | "pre-restore"
}

// DefaultBackupRetention is used when a profile sets no explicit retention.
const DefaultBackupRetention = 10

func (s *Store) profileBackupDir(profileID string) string {
	return filepath.Join(s.backupsRoot(), profileID)
}

func (s *Store) backupIndexPath(profileID string) string {
	return filepath.Join(s.profileBackupDir(profileID), "index.json")
}

func (s *Store) loadBackupIndex(profileID string) ([]BackupEntry, error) {
	var entries []BackupEntry
	data, err := os.ReadFile(s.backupIndexPath(profileID))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) saveBackupIndex(profileID string, entries []BackupEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.backupIndexPath(profileID), data, 0644)
}

// Snapshot copies the raw bytes of srcPath into the profile's backup store and
// records an index entry. Raw bytes are stored (never re-serialized), so a
// restore is byte-identical to what was saved.
func (s *Store) Snapshot(profileID, srcPath, note, kind string) (BackupEntry, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return BackupEntry{}, err
	}

	dir := s.profileBackupDir(profileID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return BackupEntry{}, err
	}

	entries, err := s.loadBackupIndex(profileID)
	if err != nil {
		return BackupEntry{}, err
	}

	stem := s.now().UTC().Format("20060102-150405.000000000")
	stem = uniqueStem(stem, entries)
	sum := sha256.Sum256(data)
	entry := BackupEntry{
		ID:        stem,
		File:      stem + ".ini",
		Timestamp: s.now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		SHA256:    hex.EncodeToString(sum[:]),
		Size:      int64(len(data)),
		Note:      note,
		Kind:      kind,
	}

	if err := os.WriteFile(filepath.Join(dir, entry.File), data, 0644); err != nil {
		return BackupEntry{}, err
	}
	entries = append(entries, entry)
	if err := s.saveBackupIndex(profileID, entries); err != nil {
		return BackupEntry{}, err
	}
	return entry, nil
}

// Backups returns a profile's snapshots, newest first.
func (s *Store) Backups(profileID string) ([]BackupEntry, error) {
	entries, err := s.loadBackupIndex(profileID)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID > entries[j].ID })
	return entries, nil
}

// ReadBackup returns the raw bytes of a stored snapshot (e.g. to render a diff).
func (s *Store) ReadBackup(profileID, backupID string) ([]byte, error) {
	entry, err := s.findBackup(profileID, backupID)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(s.profileBackupDir(profileID), entry.File))
}

// Restore writes a snapshot back to destPath. When destPath exists it first
// takes a "pre-restore" safety snapshot so the restore itself is reversible.
func (s *Store) Restore(profileID, backupID, destPath string) error {
	entry, err := s.findBackup(profileID, backupID)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(s.profileBackupDir(profileID), entry.File))
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(destPath); statErr == nil {
		if _, err := s.Snapshot(profileID, destPath, "before restore of "+backupID, "pre-restore"); err != nil {
			return err
		}
	}
	return os.WriteFile(destPath, data, 0644)
}

// DeleteBackup removes a snapshot and its index entry.
func (s *Store) DeleteBackup(profileID, backupID string) error {
	entries, err := s.loadBackupIndex(profileID)
	if err != nil {
		return err
	}
	idx := -1
	for i, e := range entries {
		if e.ID == backupID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrNoBackup
	}
	_ = os.Remove(filepath.Join(s.profileBackupDir(profileID), entries[idx].File))
	entries = append(entries[:idx], entries[idx+1:]...)
	return s.saveBackupIndex(profileID, entries)
}

// Prune keeps the newest keep snapshots and deletes the rest. keep <= 0 uses
// DefaultBackupRetention.
func (s *Store) Prune(profileID string, keep int) error {
	if keep <= 0 {
		keep = DefaultBackupRetention
	}
	entries, err := s.Backups(profileID) // newest first
	if err != nil {
		return err
	}
	if len(entries) <= keep {
		return nil
	}
	for _, e := range entries[keep:] {
		_ = os.Remove(filepath.Join(s.profileBackupDir(profileID), e.File))
	}
	return s.saveBackupIndex(profileID, entries[:keep])
}

func (s *Store) findBackup(profileID, backupID string) (BackupEntry, error) {
	entries, err := s.loadBackupIndex(profileID)
	if err != nil {
		return BackupEntry{}, err
	}
	for _, e := range entries {
		if e.ID == backupID {
			return e, nil
		}
	}
	return BackupEntry{}, ErrNoBackup
}

func uniqueStem(stem string, entries []BackupEntry) string {
	taken := make(map[string]bool, len(entries))
	for _, e := range entries {
		taken[e.ID] = true
	}
	if !taken[stem] {
		return stem
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d", stem, i)
		if !taken[candidate] {
			return candidate
		}
	}
}
