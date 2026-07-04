package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/steam/steamtest"
)

// TestE2EAddAndSave drives a full happy path: launcher -> open profile -> search
// -> detail -> add a mod -> save -> verify the file changed and a backup exists.
func TestE2EAddAndSave(t *testing.T) {
	fake := steamtest.New(steam.WorkshopItem{
		Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100",
		Title: "BaseMod", Description: "Mod ID: Base\n",
	})
	tm, m, _ := testModel(t, fake)

	waitForText(t, tm, "Demo Server")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open profile
	waitForText(t, tm, "Search Workshop")

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // -> Search Workshop
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open search
	waitForText(t, tm, "BaseMod")

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // enter = open detail
	waitForText(t, tm, "Mod ID: Base")      // detail loaded
	tm.Send(keyRune('a'))                   // add from detail
	waitForText(t, tm, "added")

	// Global ctrl+s opens the save confirm screen; 's' confirms.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlS})
	waitForText(t, tm, "view diff") // unique to the save-confirm screen
	tm.Send(keyRune('s'))
	waitForText(t, tm, "saved")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // saved -> not dirty -> quits
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	// The config file now contains the added item and its mod.
	data, err := os.ReadFile(m.s.Profile.IniPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "WorkshopItems=100") {
		t.Errorf("saved file missing WorkshopItems=100:\n%s", content)
	}
	if !strings.Contains(content, "Mods=Base") {
		t.Errorf("saved file missing Mods=Base:\n%s", content)
	}

	// A pre-save backup was created.
	backups, err := m.s.Store.Backups(m.s.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) == 0 {
		t.Error("expected a backup to be created before saving")
	}
}

// TestE2EAddSheetAndLoadOrder exercises the smart add sheet and the Load order
// Maps tab. An item with one mod and one map routes through quickAdd to the add
// sheet; the map is toggled off before applying. The test then opens Load order,
// switches to the Maps tab to verify the base map renders, and finally goes
// through ctrl+s → save confirm → save, asserting the written file and backup.
func TestE2EAddSheetAndLoadOrder(t *testing.T) {
	// "Addon Pack" has one mod and one map, so quickAdd routes to the add sheet.
	fake := steamtest.New(steam.WorkshopItem{
		Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
		Title: "Addon Pack", Description: "Mod ID: AddonMod\nMap Folder: SomeTown\n",
	})
	// Seed the config with a base map so the Maps tab has content even after
	// the add-sheet map is toggled off.
	tm, m := openProfileModelWith(t, fake, "Mods=\nWorkshopItems=\nMap=Muldraugh, KY\n")
	waitForText(t, tm, "Server info") // dashboard ready

	// -- Search → smart add sheet --
	tm.Send(keyRune('s'))            // dashboard shortcut: open Search Workshop
	waitForText(t, tm, "Addon Pack") // search results loaded

	// Enter opens the detail screen; 'a' triggers quickAdd; item has 1 mod +
	// 1 map → routes to AddSheet.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForText(t, tm, "Mod ID: AddonMod") // detail loaded
	tm.Send(keyRune('a'))                  // add from detail → routes to AddSheet
	waitForText(t, tm, "adds to rotation") // add sheet map row rendered (sync)

	// Cursor starts on the mod row (AddonMod); move down to the map row (SomeTown)
	// and toggle it off before applying.
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(keyRune(' '))                   // toggle map off
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // apply
	waitForText(t, tm, "added")             // toast: "added Addon Pack (unsaved)"

	// Esc back to dashboard (add sheet already popped to search; esc from search).
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitForText(t, tm, "Installed Mods") // dashboard menu visible

	// -- Load order: Maps tab --
	tm.Send(keyRune('l'))            // dashboard shortcut: open Load order
	waitForText(t, tm, "switch tab") // hint unique to Load order screen

	tm.Send(tea.KeyMsg{Type: tea.KeyRight}) // switch to Maps tab
	waitForText(t, tm, "Muldraugh")         // base map visible in Maps tab

	// -- ctrl+s → save confirm → save --
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlS})
	waitForText(t, tm, "view diff") // unique to the save-confirm screen
	tm.Send(keyRune('s'))
	waitForText(t, tm, "saved")

	// Quit (clean after save).
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	// File must contain the added item/mod and not the toggled-off map.
	data, err := os.ReadFile(m.s.Profile.IniPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "WorkshopItems=200") {
		t.Errorf("saved file missing WorkshopItems=200:\n%s", content)
	}
	if !strings.Contains(content, "Mods=AddonMod") {
		t.Errorf("saved file missing Mods=AddonMod:\n%s", content)
	}
	if strings.Contains(content, "SomeTown") {
		t.Errorf("saved file contains SomeTown which was toggled off:\n%s", content)
	}

	// A pre-save backup was created.
	backups, err := m.s.Store.Backups(m.s.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) == 0 {
		t.Error("expected a backup to be created before saving")
	}
}
