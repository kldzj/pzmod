package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/internal/steam"
	"github.com/kldzj/pzmod/internal/steam/steamtest"
	"github.com/kldzj/pzmod/internal/store"
)

// openB42Model opens a profile whose build is b42, starting at Installed Mods.
func openB42Model(t *testing.T, fake *steamtest.Fake, iniBody string) (*teatest.TestModel, *model) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	st, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	ini := filepath.Join(t.TempDir(), "server.ini")
	if err := os.WriteFile(ini, []byte(iniBody), 0644); err != nil {
		t.Fatal(err)
	}
	p, _ := st.AddProfile(store.Profile{Name: "B42", IniPath: ini, Build: "b42"})
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef")
	m := New(nil, st, context.Background(), NewInstalled())
	m.s.NewSteam = func(string) steam.API { return fake }
	if err := m.s.OpenProfile(p); err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})
	return tm, m
}

func TestB42AddWritesPinnedToken(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100100",
			Title: "Cool Mod", Description: "Mod ID: CoolMod\n"},
	)
	tm, m := openB42Model(t, fake, "Mods=\nWorkshopItems=\nMap=\n")
	waitForText(t, tm, "nothing installed yet")

	tm.Send(keyRune('a')) // add-by-ID
	waitForText(t, tm, "Workshop ID")
	for _, r := range "100100" {
		tm.Send(keyRune(r))
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // single mod, no maps -> instant add
	waitForText(t, tm, "Cool Mod")

	if got := m.s.Cfg.ServerMods().Mods; len(got) != 1 || got[0] != `100100\CoolMod` {
		t.Errorf("B42 add must write the pinned token; Mods = %v; want [100100\\CoolMod]", got)
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	waitForText(t, tm, "unsaved changes")
	tm.Send(keyRune('y'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
