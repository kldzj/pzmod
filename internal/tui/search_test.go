package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/steam/steamtest"
)

func keyRune(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

// TestSearchAddFlow drives the new flow: enter opens the highlighted result's
// detail screen, and 'a' adds it from there.
func TestSearchAddFlow(t *testing.T) {
	fake := steamtest.New(steam.WorkshopItem{
		Result:          1,
		FileType:        steam.FileTypeMod,
		PublishedFileID: "100",
		Title:           "Hydrocraft",
		Description:     "A big mod.\nMod ID: HydroMod\n",
	})
	tm, m, _ := testModel(t, fake)

	waitForText(t, tm, "Demo Server")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open profile -> dashboard
	waitForText(t, tm, "Search Workshop")

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // move to "Search Workshop"
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open search
	waitForText(t, tm, "Hydrocraft")        // results from the fake

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // enter = open the detail screen
	waitForText(t, tm, "Mod ID: HydroMod")  // detail description marker
	tm.Send(keyRune('a'))                   // add from detail
	waitForText(t, tm, "added")

	// cfg is now dirty; ctrl+c prompts, confirm with y.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	waitForText(t, tm, "unsaved changes")
	tm.Send(keyRune('y'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	// The shallow add updated the in-memory config.
	sm := m.s.Cfg.ServerMods()
	if !contains(sm.WorkshopItems, "100") {
		t.Errorf("WorkshopItems = %v; want it to contain 100", sm.WorkshopItems)
	}
	if !contains(sm.Mods, "HydroMod") {
		t.Errorf("Mods = %v; want it to contain HydroMod", sm.Mods)
	}
}

// TestSearchCtrlOAddsID verifies that typing a 6+ digit Workshop ID into the
// search box and pressing ctrl+o adds the item directly without navigating to
// the detail screen.
func TestSearchCtrlOAddsID(t *testing.T) {
	fake := steamtest.New(steam.WorkshopItem{
		Result:          1,
		FileType:        steam.FileTypeMod,
		PublishedFileID: "200200",
		Title:           "DirectAddMod",
		Description:     "Mod ID: DirectAddMod\n",
	})
	tm, m := openProfileModelWith(t, fake, "WorkshopItems=\nMods=\nMap=\n")
	waitForText(t, tm, "Server info") // dashboard ready

	tm.Send(keyRune('s'))              // dashboard shortcut: open Search Workshop
	waitForText(t, tm, "DirectAddMod") // initial results from the fake

	// Type the 6-digit workshop ID into the search box.
	for _, r := range "200200" {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	waitForText(t, tm, "ctrl+o") // hint appears when a parseable ID is in the box

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlO})
	waitForText(t, tm, "added")

	sm := m.s.Cfg.ServerMods()
	if !contains(sm.WorkshopItems, "200200") {
		t.Errorf("WorkshopItems = %v; want it to contain 200200", sm.WorkshopItems)
	}
	if !contains(sm.Mods, "DirectAddMod") {
		t.Errorf("Mods = %v; want it to contain DirectAddMod", sm.Mods)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
