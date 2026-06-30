package tui

import (
	"context"
	"fmt"
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

// openedModelAt builds a model with a profile already open, starting at initial.
func openedModelAt(t *testing.T, fake *steamtest.Fake, iniContent string, initial Screen) (*teatest.TestModel, *model) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	st, err := store.New(store.WithRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	ini := filepath.Join(t.TempDir(), "server.ini")
	if err := os.WriteFile(ini, []byte(iniContent), 0644); err != nil {
		t.Fatal(err)
	}
	p, _ := st.AddProfile(store.Profile{Name: "Demo", IniPath: ini})
	_ = st.SetGlobalKey("0123456789abcdef0123456789abcdef") // avoid the first-run key prompt

	m := New(nil, st, context.Background(), initial)
	m.s.NewSteam = func(string) steam.API { return fake }
	if err := m.s.OpenProfile(p); err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})
	return tm, m
}

func TestDepsResolveAndApply(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
			Title: "Weapons Pack", Description: "Mod ID: Weapons\n",
			Children: []steam.WorkshopItemChild{{PublishedFileID: "100"}}},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100",
			Title: "Core Library", Description: "Mod ID: CoreLib\n"},
	)

	tm, m := openedModelAt(t, fake, "Mods=\nWorkshopItems=\n", NewDeps([]string{"200"}))

	waitForText(t, tm, "Core Library") // the transitive dependency is listed

	tm.Send(keyRune('a')) // apply all selected
	waitForText(t, tm, "added 2")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // dirty -> confirm
	waitForText(t, tm, "unsaved changes")
	tm.Send(keyRune('y'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	sm := m.s.Cfg.ServerMods()
	if !contains(sm.WorkshopItems, "100") || !contains(sm.WorkshopItems, "200") {
		t.Errorf("WorkshopItems = %v; want 100 and 200", sm.WorkshopItems)
	}
	if !contains(sm.Mods, "Weapons") || !contains(sm.Mods, "CoreLib") {
		t.Errorf("Mods = %v; want Weapons and CoreLib", sm.Mods)
	}
}

// TestDepsUnavailableCollapsed verifies that missing deps are collapsed to a
// summary line rather than listed inline, and that pressing 'u' opens the
// InfoList screen showing the unavailable IDs.
//
// RED state: 'u' is not yet handled; waitForText("999") times out because the
// InfoList is not pushed and the ANSI-compressed output contains no new data.
func TestDepsUnavailableCollapsed(t *testing.T) {
	// Item "200" depends on "999" which is not in the fake → plan.Missing = ["999"]
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
			Title: "Weapons Pack", Description: "Mod ID: Weapons\n",
			Children: []steam.WorkshopItemChild{{PublishedFileID: "999"}}},
	)

	tm, _ := openedModelAt(t, fake, "Mods=\nWorkshopItems=\n", NewDeps([]string{"200"}))

	// The collapsed summary must appear (not each ID as a raw error line).
	waitForText(t, tm, "unavailable")

	// Press 'u' to open the InfoList of unavailable IDs.
	tm.Send(keyRune('u'))
	// InfoList pushes a new full-screen render containing the missing IDs.
	waitForText(t, tm, "999")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestDepsFooterVisibleWhenScrolled verifies that the "selected total" line is
// not cut off when there are many rows plus several unavailable items.
//
// Setup: 25 addable items, 5 unavailable (missing children of item 1000).
// With OLD code: missing items are listed inline (5 lines), pushing "selected total"
// beyond BodyHeight(27) so it is truncated and never written to tm.out.
// With NEW code (Changes 3+4): missing items collapse to 1 line and h is reduced,
// so "selected total" fits within the body and appears in the initial frame.
func TestDepsFooterVisibleWhenScrolled(t *testing.T) {
	// Item 1000 has 5 missing children; items 1001-1024 are normal.
	children := []steam.WorkshopItemChild{
		{PublishedFileID: "m1"}, {PublishedFileID: "m2"},
		{PublishedFileID: "m3"}, {PublishedFileID: "m4"},
		{PublishedFileID: "m5"},
	}
	items := make([]steam.WorkshopItem, 25)
	seeds := make([]string, 25)
	for i := 0; i < 25; i++ {
		id := fmt.Sprintf("%d", 1000+i)
		it := steam.WorkshopItem{
			Result:          1,
			FileType:        steam.FileTypeMod,
			PublishedFileID: id,
			Title:           "Item " + id,
			Description:     "Mod ID: Mod" + id + "\n",
			FileSize:        steam.ItemSize(1024 * (i + 1)), // non-zero so humanize renders bytes
		}
		if i == 0 {
			it.Children = children
		}
		items[i] = it
		seeds[i] = id
	}
	fake := steamtest.New(items...)

	tm, _ := openedModelAt(t, fake, "Mods=\nWorkshopItems=\n", NewDeps(seeds))

	// "selected total" must be visible in the initial resolved frame.
	// With old code it is beyond BodyHeight and is truncated → timeout (RED).
	// With new code the footer fits → it appears in tm.out → GREEN.
	waitForText(t, tm, "selected total")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
