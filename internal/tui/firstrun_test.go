package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/internal/service"
	"github.com/kldzj/pzmod/internal/steam"
	"github.com/kldzj/pzmod/internal/steam/steamtest"
	"github.com/kldzj/pzmod/internal/store"
)

// TestSettingsSaveReturnsToLauncher covers the first-run flow: with no API key
// configured, the Settings screen is pushed on top of the launcher; after the
// user saves a valid key it should pop back to the profile menu automatically,
// without an extra esc.
func TestSettingsSaveReturnsToLauncher(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	st, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	// A profile exists so the launcher shows its "enter: open" chrome; no API
	// key -> the Settings prompt pops up first.
	ini := filepath.Join(t.TempDir(), "server.ini")
	if err := os.WriteFile(ini, []byte("Mods=\nWorkshopItems=\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddProfile(store.Profile{Name: "Demo Server", IniPath: ini, Build: "b41"}); err != nil {
		t.Fatal(err)
	}

	m := New(service.New(steamtest.New(), st), st, context.Background(), NewLauncher())
	m.s.NewSteam = func(string) steam.API { return steamtest.New() }
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})

	// First run: the Settings prompt is on top.
	waitForText(t, tm, "Steam Web API key")

	// Type a valid 32-char key and save.
	for _, r := range "0123456789abcdef0123456789abcdef" {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// We should be back on the profile menu without pressing esc: "enter: open"
	// is launcher-only chrome (Settings shows "enter: save").
	waitForText(t, tm, "enter: open")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestLauncherReloadsOnResume covers the add-profile refresh: a profile added
// while a child screen was on top must appear when we return to the launcher.
// The refresh must ride the reliable post-Pop resumedMsg, not a racy
// profilesChangedMsg batched alongside Pop().
func TestLauncherReloadsOnResume(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	st, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	s := &Session{Store: st, Theme: DefaultTheme()}
	l := &launcher{}

	// Initial load: no profiles yet.
	msg := l.Init(s)()
	l.Update(s, msg)
	if len(l.profiles) != 0 {
		t.Fatalf("expected 0 profiles initially, got %d", len(l.profiles))
	}

	// A profile is added while the add-profile screen sits on top.
	ini := filepath.Join(t.TempDir(), "server.ini")
	if err := os.WriteFile(ini, []byte("Mods=\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddProfile(store.Profile{Name: "Added", IniPath: ini, Build: "b41"}); err != nil {
		t.Fatal(err)
	}

	// Returning to the launcher (Pop -> resumedMsg) must refresh the list.
	_, cmd := l.Update(s, resumedMsg{})
	if cmd == nil {
		t.Fatal("launcher did not reload on resumedMsg")
	}
	l.Update(s, cmd())
	if len(l.profiles) != 1 {
		t.Fatalf("expected 1 profile after resume-refresh, got %d", len(l.profiles))
	}
}
