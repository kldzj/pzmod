package tui

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/internal/steam"
	"github.com/kldzj/pzmod/internal/steam/steamtest"
)

func TestLoadOrderSuggestAndApply(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
			Title: "Weapons Pack", Description: "Mod ID: Weapons\n",
			Children: []steam.WorkshopItemChild{{PublishedFileID: "100"}}},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100",
			Title: "Core Library", Description: "Mod ID: CoreLib\n"},
	)
	// Wrong order on disk: Weapons before its dependency/library CoreLib.
	tm, m := openedModelAt(t, fake, "Mods=Weapons;CoreLib\nWorkshopItems=200;100\n", NewModList())

	// Suggestion arrives -> hint shows ✦ ready indicator.
	waitForText(t, tm, "✦")

	tm.Send(keyRune('s')) // preview suggestion
	waitForText(t, tm, "Suggested order")

	tm.Send(keyRune('y')) // apply
	waitForText(t, tm, "applied")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // dirty -> confirm
	waitForText(t, tm, "unsaved changes")
	tm.Send(keyRune('y'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	if got := m.s.Cfg.Mods(); !reflect.DeepEqual(got, []string{"CoreLib", "Weapons"}) {
		t.Errorf("Mods = %v; want [CoreLib Weapons] after applying suggestion", got)
	}
}

func TestLoadOrderManualMove(t *testing.T) {
	fake := steamtest.New() // no steam needed for manual move
	tm, m := openedModelAt(t, fake, "Mods=a;b;c\nWorkshopItems=\n", NewModList())

	waitForText(t, tm, "1. ") // list rendered

	// Grab the first item (a) and move it down twice -> b, c, a.
	tm.Send(keyRune(' '))
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(keyRune(' ')) // drop

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	waitForText(t, tm, "unsaved changes")
	tm.Send(keyRune('y'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	if got := m.s.Cfg.Mods(); !reflect.DeepEqual(got, []string{"b", "c", "a"}) {
		t.Errorf("Mods = %v; want [b c a] after manual move", got)
	}
}

func TestLoadOrderFilterLocate(t *testing.T) {
	tm, _ := openedModelAt(t, steamtest.New(),
		"Mods=Alpha;Bravo;Charlie\nWorkshopItems=\nMap=\n", NewLoadOrder())
	waitForText(t, tm, "Alpha")

	tm.Send(keyRune('/'))
	for _, r := range "charlie" {
		tm.Send(keyRune(r))
	}
	// Only Charlie shown, with its real full-list ordinal (3.).
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("filter: charlie")) &&
			bytes.Contains(b, []byte("3.")) && bytes.Contains(b, []byte("Charlie"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	// enter jumps to the item and clears the filter; the full list returns.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForText(t, tm, "Alpha")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestLoadOrderFilterTypesJK(t *testing.T) {
	// "Jukebox" needs both 'j' and 'k' typed into the query; if they navigated
	// instead, the query would be "uebox" which is NOT a substring of "jukebox".
	tm, _ := openedModelAt(t, steamtest.New(),
		"Mods=Alpha;Jukebox;Bravo\nWorkshopItems=\nMap=\n", NewLoadOrder())
	waitForText(t, tm, "Jukebox")

	tm.Send(keyRune('/'))
	for _, r := range "jukebox" {
		tm.Send(keyRune(r))
	}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("filter: jukebox")) && bytes.Contains(b, []byte("Jukebox"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestLoadOrderFilterClearsGrab(t *testing.T) {
	tm, m := openedModelAt(t, steamtest.New(),
		"Mods=Alpha;Bravo;Charlie\nWorkshopItems=\nMap=\n", NewLoadOrder())
	waitForText(t, tm, "Alpha")

	tm.Send(keyRune(' ')) // grab the item at cursor 0 (Alpha)
	waitForText(t, tm, "MOVING")

	tm.Send(keyRune('/')) // entering the filter must clear the grab
	waitForText(t, tm, "filter:")

	tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // must NOT shift (grab cleared)
	tm.Send(keyRune('a'))                  // barrier: ensures the Down above was processed
	waitForText(t, tm, "filter: a")

	if got := m.s.Cfg.Mods(); len(got) != 3 || got[0] != "Alpha" || got[1] != "Bravo" || got[2] != "Charlie" {
		t.Errorf("entering filter must clear grab; order changed: %v", got)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestLoadOrderFilterTypesReorderChars(t *testing.T) {
	// "Big Town" contains 'g', a space, and 't' - all reorder shortcuts in normal
	// mode. While typing a filter they must be typed into the query, not intercepted.
	tm, _ := openedModelAt(t, steamtest.New(),
		"Mods=Alpha;Big Town;Charlie\nWorkshopItems=\nMap=\n", NewLoadOrder())
	waitForText(t, tm, "Big Town")

	tm.Send(keyRune('/'))
	for _, r := range "ig to" { // i, g, space, t, o -> matches "Big Town"
		tm.Send(keyRune(r))
	}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("filter: ig to")) && bytes.Contains(b, []byte("Big Town"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestLoadOrderB42DisplayAndFilter(t *testing.T) {
	tm, _ := openedModelAt(t, steamtest.New(),
		"Mods=\\tsarslib;2392709985\\Containers\nWorkshopItems=\nMap=\n", NewLoadOrder())
	// The clean mod ID is shown (not the raw "\tsarslib"); the pin is visible.
	// Both assertions are in one WaitFor because tm.Output() is a consuming stream.
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("tsarslib")) && bytes.Contains(b, []byte("pinned 2392709985"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	// Filtering by the logical mod ID (no backslash) matches.
	tm.Send(keyRune('/'))
	for _, r := range "containers" {
		tm.Send(keyRune(r))
	}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("filter: containers")) && bytes.Contains(b, []byte("Containers"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestLoadOrderB42SuggestCleanDisplay(t *testing.T) {
	// The suggestion preview must render the clean mod ID (no backslash). The "→"
	// moved-marker makes "→ tsarslib" provable: a raw token would render
	// "→ \tsarslib" (a backslash between the marker and the id).
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "10",
			Title: "Lib", Description: "Mod ID: tsarslib\n"},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "20",
			Title: "Big Guns", Description: "Mod ID: BigGuns\n"},
	)
	tm, _ := openedModelAt(t, fake,
		"Mods=\\BigGuns;\\tsarslib\nWorkshopItems=20;10\nMap=\n", NewLoadOrder())
	waitForText(t, tm, "suggest ✦") // suggestion computed (sync point)
	tm.Send(keyRune('s'))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Suggested order")) && bytes.Contains(b, []byte("→ tsarslib"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
