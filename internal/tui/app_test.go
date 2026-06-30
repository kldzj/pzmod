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

func testModel(t *testing.T, fake *steamtest.Fake) (*teatest.TestModel, *model, *store.Store) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	st, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	ini := filepath.Join(t.TempDir(), "server.ini")
	if err := os.WriteFile(ini, []byte("PublicName=Demo Server\nMods=\nWorkshopItems=\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddProfile(store.Profile{Name: "Demo Server", IniPath: ini, Build: "b41"}); err != nil {
		t.Fatal(err)
	}
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef") // avoid the first-run key prompt

	svc := service.New(fake, st)
	m := New(svc, st, context.Background(), NewLauncher())
	m.s.NewSteam = func(string) steam.API { return fake }

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})
	return tm, m, st
}

func waitForText(t *testing.T, tm *teatest.TestModel, text string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(text))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))
}

func TestLauncherToDashboard(t *testing.T) {
	tm, _, _ := testModel(t, steamtest.New())

	// Launcher lists the profile.
	waitForText(t, tm, "Demo Server")

	// Open it -> dashboard.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForText(t, tm, "Server info") // a dashboard menu entry

	// Quit (no unsaved changes -> immediate).
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestStartupPromptsForAPIKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	st, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	// No API key configured -> the Settings screen should pop up on start.
	m := New(service.New(steamtest.New(), st), st, context.Background(), NewLauncher())
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})

	waitForText(t, tm, "Steam Web API key")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestDashboardShowsCounts(t *testing.T) {
	tm, _, _ := testModel(t, steamtest.New())
	waitForText(t, tm, "Demo Server")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	// Dashboard header summarizes the config.
	waitForText(t, tm, "build Build 41")
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
