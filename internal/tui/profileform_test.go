package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kldzj/pzmod/pkg/store"
)

// TestProfileFormCreate covers create(): ~ expansion, missing-file rejection,
// and that a valid form adds exactly one profile. (The form's `done` guard,
// which prevents a second add when huh stays in the Completed state, is covered
// by the interactive flow; here we exercise the create logic deterministically.)
func TestProfileFormCreate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ini := filepath.Join(home, "server.ini")
	if err := os.WriteFile(ini, []byte("Mods=\n"), 0644); err != nil {
		t.Fatal(err)
	}

	st, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	s := &Session{Store: st}

	// Valid: a ~ path is expanded and one profile is created.
	pf := &profileform{name: "My Server", file: "~/server.ini", build: "b41"}
	if cmd := pf.create(s); cmd == nil {
		t.Fatal("create returned nil for a valid profile")
	}
	profiles, _ := st.Profiles()
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].IniPath != ini {
		t.Errorf("IniPath = %q; want expanded %q", profiles[0].IniPath, ini)
	}

	// Missing file: rejected, nothing added.
	bad := &profileform{name: "Bad", file: "~/does-not-exist.ini", build: "b41"}
	bad.create(s)
	profiles, _ = st.Profiles()
	if len(profiles) != 1 {
		t.Errorf("a non-existent config must not create a profile; have %d", len(profiles))
	}
}
