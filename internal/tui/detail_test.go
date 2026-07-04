package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/steam/steamtest"
)

// TestDetailRemoveWhenInstalled verifies that pressing x on a detail screen for
// an installed item opens a confirmation modal and removes the item (plus its
// uniquely-owned mods/maps) on confirmation.
func TestDetailRemoveWhenInstalled(t *testing.T) {
	fake := steamtest.New()
	fake.Items["100001"] = steam.WorkshopItem{
		Result:          1,
		FileType:        steam.FileTypeMod,
		PublishedFileID: "100001",
		Title:           "FooMod",
		Description:     "Mod ID: FooMod\nMap Folder: FooMap\n",
	}

	tm, m := openProfileModelWith(t, fake, "WorkshopItems=100001\nMods=FooMod\nMap=FooMap\n")
	tm.Send(PushMsg{Screen: NewDetail("100001")})

	// Wait for the detail screen to load and show the remove option in the footer.
	waitForText(t, tm, "x: remove")

	tm.Send(keyRune('x'))        // press x to remove
	waitForText(t, tm, "Remove") // confirm modal

	tm.Send(keyRune('y'))         // confirm
	waitForText(t, tm, "removed") // toast

	sm := m.s.Cfg.ServerMods()
	if sm.HasItem("100001") || sm.HasMod("FooMod") || sm.HasMap("FooMap") {
		t.Fatalf("expected item+owned mod+map removed: %+v", sm)
	}
}

// TestDetailNoRemoveWhenNotInstalled verifies that the remove key (x) is not
// offered in the footer when the item is not installed.
func TestDetailNoRemoveWhenNotInstalled(t *testing.T) {
	fake := steamtest.New(steam.WorkshopItem{
		Result:          1,
		FileType:        steam.FileTypeMod,
		PublishedFileID: "200001",
		Title:           "BarMod",
		Description:     "Mod ID: BarMod\n",
	})

	tm, _ := openProfileModelWith(t, fake, "WorkshopItems=\nMods=\nMap=\n")
	tm.Send(PushMsg{Screen: NewDetail("200001")})

	// Wait until the detail is loaded (footer shows "a: add") and verify that
	// "x: remove" is absent for a non-installed item.
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("a: add")) && !bytes.Contains(b, []byte("x: remove"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestSearchEnterOpensDetail verifies that pressing enter in the search results
// list navigates to the detail screen (not quick-add).
func TestSearchEnterOpensDetail(t *testing.T) {
	fake := steamtest.New(steam.WorkshopItem{
		Result:          1,
		FileType:        steam.FileTypeMod,
		PublishedFileID: "100",
		Title:           "Hydrocraft",
		Description:     "A big mod.\nMod ID: HydroMod\n",
	})
	tm, _, _ := testModel(t, fake)

	waitForText(t, tm, "Demo Server")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open profile -> dashboard
	waitForText(t, tm, "Search Workshop")

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // move to "Search Workshop"
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open search
	waitForText(t, tm, "Hydrocraft")        // results from the fake

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // enter = open detail (not quick-add)
	waitForText(t, tm, "a: add")            // detail screen footer

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
