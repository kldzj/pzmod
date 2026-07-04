package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir()) // isolate legacy ~/.pzmod migration from the real home
	s, err := New(WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestProfilesCRUD(t *testing.T) {
	s := newTestStore(t)

	p, err := s.AddProfile(Profile{Name: "My Server", IniPath: "server.ini"})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "my-server" {
		t.Errorf("ID = %q; want my-server", p.ID)
	}
	if !filepath.IsAbs(p.IniPath) {
		t.Errorf("IniPath should be absolute, got %q", p.IniPath)
	}

	// First profile becomes default.
	def, err := s.DefaultProfile()
	if err != nil || def.ID != "my-server" {
		t.Errorf("default = %v, %v; want my-server", def.ID, err)
	}

	// Duplicate name gets a unique ID.
	p2, _ := s.AddProfile(Profile{Name: "My Server", IniPath: "b.ini"})
	if p2.ID != "my-server-2" {
		t.Errorf("dup ID = %q; want my-server-2", p2.ID)
	}

	all, _ := s.Profiles()
	if len(all) != 2 {
		t.Fatalf("profiles = %d; want 2", len(all))
	}

	if err := s.SetDefaultProfile("my-server-2"); err != nil {
		t.Fatal(err)
	}
	if def, _ := s.DefaultProfile(); def.ID != "my-server-2" {
		t.Errorf("default after set = %v", def.ID)
	}

	if err := s.RemoveProfile("my-server-2"); err != nil {
		t.Fatal(err)
	}
	// Default falls back to remaining profile.
	if def, _ := s.DefaultProfile(); def.ID != "my-server" {
		t.Errorf("default after remove = %v; want my-server", def.ID)
	}
	if _, err := s.Profile("my-server-2"); err != ErrNoProfile {
		t.Errorf("removed profile lookup err = %v; want ErrNoProfile", err)
	}
}

func TestCredentialsAndPerms(t *testing.T) {
	t.Setenv(envAPIKey, "") // ignore any ambient PZMOD_STEAM_KEY
	s := newTestStore(t)

	if _, err := s.APIKey(""); err != ErrNoKey {
		t.Errorf("APIKey on empty = %v; want ErrNoKey", err)
	}
	if err := s.SetGlobalKey("GLOBALKEY"); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.APIKey(""); k != "GLOBALKEY" {
		t.Errorf("global key = %q", k)
	}

	// Per-profile override wins.
	if err := s.SetProfileKey("p1", "PROFILEKEY"); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.APIKey("p1"); k != "PROFILEKEY" {
		t.Errorf("profile key = %q; want PROFILEKEY", k)
	}
	if k, _ := s.APIKey("other"); k != "GLOBALKEY" {
		t.Errorf("fallback key = %q; want GLOBALKEY", k)
	}

	info, err := os.Stat(s.credentialsPath())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("credentials perms = %o; want 600", perm)
	}
}

func TestAPIKeyFromEnv(t *testing.T) {
	s := newTestStore(t)

	t.Setenv(envAPIKey, "ENVKEY")
	if k, _ := s.APIKey(""); k != "ENVKEY" {
		t.Errorf("APIKey with env set = %q; want ENVKEY", k)
	}

	// The env var takes precedence over a stored global key.
	if err := s.SetGlobalKey("GLOBALKEY"); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.APIKey(""); k != "ENVKEY" {
		t.Errorf("APIKey = %q; want the env var to win over the global key", k)
	}

	// A per-profile override still wins over the env var.
	if err := s.SetProfileKey("p1", "PROFILEKEY"); err != nil {
		t.Fatal(err)
	}
	if k, _ := s.APIKey("p1"); k != "PROFILEKEY" {
		t.Errorf("APIKey(p1) = %q; want the per-profile key to win over the env var", k)
	}

	// With the env var cleared, fall back to the stored global key.
	t.Setenv(envAPIKey, "")
	if k, _ := s.APIKey(""); k != "GLOBALKEY" {
		t.Errorf("APIKey with empty env = %q; want GLOBALKEY", k)
	}
}

func TestLegacyMigrationIdempotent(t *testing.T) {
	t.Setenv(envAPIKey, "") // ignore any ambient PZMOD_STEAM_KEY
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home) // legacyKeyPath uses UserHomeDir

	legacy := filepath.Join(home, ".pzmod")
	if err := os.WriteFile(legacy, []byte("LEGACYKEY1234567890123456789012\n"), 0644); err != nil {
		t.Fatal(err)
	}

	s, err := New(WithRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	if k, _ := s.APIKey(""); k != "LEGACYKEY1234567890123456789012" {
		t.Errorf("migrated key = %q", k)
	}
	// Legacy file must be left intact.
	if _, err := os.Stat(legacy); err != nil {
		t.Errorf("legacy file should remain: %v", err)
	}

	// Changing the new key then re-running New must NOT clobber it from legacy.
	if err := s.SetGlobalKey("NEWKEY"); err != nil {
		t.Fatal(err)
	}
	s2, err := New(WithRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	if k, _ := s2.APIKey(""); k != "NEWKEY" {
		t.Errorf("re-migration clobbered key: %q; want NEWKEY", k)
	}
}

func TestBackupSnapshotRestorePrune(t *testing.T) {
	root := t.TempDir()
	tick := time.Unix(1700000000, 0)
	clock := func() time.Time { t := tick; tick = tick.Add(time.Second); return t }
	s, err := New(WithRoot(root), WithClock(clock))
	if err != nil {
		t.Fatal(err)
	}

	cfg := filepath.Join(t.TempDir(), "server.ini")
	original := []byte("PublicName=v1\r\nMods=a;b\r\n") // CRLF to prove raw-byte fidelity
	if err := os.WriteFile(cfg, original, 0644); err != nil {
		t.Fatal(err)
	}

	entry, err := s.Snapshot("p1", cfg, "first", "manual")
	if err != nil {
		t.Fatal(err)
	}
	// Stored bytes must equal source bytes exactly.
	got, err := s.ReadBackup("p1", entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Errorf("snapshot bytes differ from source")
	}

	// Mutate the file, then restore: a pre-restore safety snapshot is taken.
	if err := os.WriteFile(cfg, []byte("PublicName=v2\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Restore("p1", entry.ID, cfg); err != nil {
		t.Fatal(err)
	}
	restored, _ := os.ReadFile(cfg)
	if string(restored) != string(original) {
		t.Errorf("restore not byte-exact: %q", restored)
	}
	backups, _ := s.Backups("p1")
	if len(backups) != 2 { // original + pre-restore
		t.Fatalf("backups = %d; want 2", len(backups))
	}
	if backups[0].Kind != "pre-restore" { // newest first
		t.Errorf("newest backup kind = %q; want pre-restore", backups[0].Kind)
	}

	// Pile up snapshots and prune to 3.
	for i := 0; i < 5; i++ {
		if _, err := s.Snapshot("p1", cfg, "", "auto"); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Prune("p1", 3); err != nil {
		t.Fatal(err)
	}
	backups, _ = s.Backups("p1")
	if len(backups) != 3 {
		t.Errorf("after prune = %d; want 3", len(backups))
	}
}

func TestEphemeralProfileIDStable(t *testing.T) {
	a := EphemeralProfileID("server.ini")
	b := EphemeralProfileID("server.ini")
	if a != b {
		t.Errorf("ephemeral id not stable: %q vs %q", a, b)
	}
	if EphemeralProfileID("other.ini") == a {
		t.Error("different paths should yield different ids")
	}
}
