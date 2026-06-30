package tui

import (
	"bytes"
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

// openProfileModel builds a model sitting on the dashboard with a profile open.
// When dirty is true, the config has an unsaved in-memory edit.
func openProfileModel(t *testing.T, dirty bool) (*teatest.TestModel, *model) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	st, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	ini := filepath.Join(t.TempDir(), "server.ini")
	if err := os.WriteFile(ini, []byte("PublicName=Demo\nMods=\nWorkshopItems=\n"), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := st.AddProfile(store.Profile{Name: "Demo", IniPath: ini, Build: "b41"})
	if err != nil {
		t.Fatal(err)
	}
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")

	fake := steamtest.New()
	m := New(service.New(fake, st), st, context.Background(), NewDashboard())
	m.s.NewSteam = func(string) steam.API { return fake }
	if err := m.s.OpenProfile(p); err != nil {
		t.Fatal(err)
	}
	if dirty {
		m.s.Cfg.SetMods([]string{"SomeMod"})
		if !m.s.Dirty() {
			t.Fatal("precondition: expected config to be dirty")
		}
	}

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})
	return tm, m
}

func openProfileModelWith(t *testing.T, fake *steamtest.Fake, iniBody string) (*teatest.TestModel, *model) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	st, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	ini := filepath.Join(t.TempDir(), "server.ini")
	if err := os.WriteFile(ini, []byte("PublicName=Demo\n"+iniBody), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := st.AddProfile(store.Profile{Name: "Demo", IniPath: ini, Build: "b41"})
	if err != nil {
		t.Fatal(err)
	}
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	m := New(service.New(fake, st), st, context.Background(), NewDashboard())
	m.s.NewSteam = func(string) steam.API { return fake }
	if err := m.s.OpenProfile(p); err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})
	return tm, m
}

func waitForGuard(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("unsaved changes"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))
}

// TestQuitGuardWhenDirtyInsideProfile: ctrl+c arriving as a key message warns
// about unsaved changes while inside a profile (the raw-mode path).
func TestQuitGuardWhenDirtyInsideProfile(t *testing.T) {
	tm, _ := openProfileModel(t, true)
	waitForText(t, tm, "Server info") // dashboard is ready
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	waitForGuard(t, tm)
}

// TestInterruptShowsGuardWhenDirty reproduces the real-terminal bug: an OS
// interrupt (SIGINT) must route through the guard, not tear the program down.
func TestInterruptShowsGuardWhenDirty(t *testing.T) {
	tm, _ := openProfileModel(t, true)
	waitForText(t, tm, "Server info")
	tm.Send(interruptMsg{})
	waitForGuard(t, tm)
}

// TestInterruptQuitsWhenClean: with no unsaved changes, an interrupt quits.
func TestInterruptQuitsWhenClean(t *testing.T) {
	tm, _ := openProfileModel(t, false)
	waitForText(t, tm, "Server info")
	tm.Send(interruptMsg{})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
