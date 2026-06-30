package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/internal/domain"
	"github.com/kldzj/pzmod/internal/steam"
	"github.com/kldzj/pzmod/internal/steam/steamtest"
)

// TestFindingItemID verifies that findingItemID extracts a numeric workshop ID
// from the finding's Subject when present, and returns "" for non-numeric subjects.
func TestFindingItemID(t *testing.T) {
	cases := []struct {
		subject string
		want    string
	}{
		{"123456", "123456"},
		{"2898007556", "2898007556"},
		{"Basements", ""},
		{"Muldraugh, KY", ""},
		{"", ""},
		{"abc123", ""},
		{"12345", ""}, // < 6 digits → not a valid workshop ID
	}
	for _, tc := range cases {
		f := domain.Finding{Subject: tc.subject}
		got := findingItemID(f)
		if got != tc.want {
			t.Errorf("findingItemID(%q) = %q; want %q", tc.subject, got, tc.want)
		}
	}
}

// TestValidateManualLabel drives the validate screen with a non-actionable finding
// (CodeNoModID - item with no Mod ID in description) and asserts in one pass that:
//   - the row renders "manual" on the right (instead of an empty column or "↵ fix")
//   - the footer shows "o: open page"
//
// Both strings are in the same static render frame, so we use a single WaitFor
// with a combined condition to avoid the "two waits on the same static frame" hang.
func TestValidateManualLabel(t *testing.T) {
	// Item 300 has no Mod ID in its description → CodeNoModID (non-actionable).
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "300",
			Title: "No ID Mod", Description: "This mod has no mod ID line\n"},
	)
	tm, _ := openedModelAt(t, fake, "Mods=\nWorkshopItems=300\n", NewValidate())

	// Wait for one render frame that contains BOTH the "manual" row label and the
	// "o: open page" footer hint (they appear together once the validate result arrives).
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("manual")) &&
			bytes.Contains(b, []byte("o: open page"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestValidateShowsMissingDependency(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
			Title: "Weapons Pack", Description: "Mod ID: Weapons\n",
			Children: []steam.WorkshopItemChild{{PublishedFileID: "100"}}},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100",
			Title: "Core Library", Description: "Mod ID: CoreLib\n"},
	)
	// 200 is installed but its dependency 100 is not -> a dry-run error.
	tm, _ := openedModelAt(t, fake, "Mods=Weapons\nWorkshopItems=200\n", NewValidate())

	waitForText(t, tm, "missing dependency")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // not dirty -> immediate quit
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestValidateFixMissingDependency(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
			Title: "Weapons Pack", Description: "Mod ID: Weapons\n",
			Children: []steam.WorkshopItemChild{{PublishedFileID: "100"}}},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100",
			Title: "Core Library", Description: "Mod ID: CoreLib\n"},
	)
	tm, m := openedModelAt(t, fake, "Mods=Weapons\nWorkshopItems=200\n", NewValidate())

	waitForText(t, tm, "missing dependency")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // fix selected finding -> auto-resolve + revalidate
	// After the dependency is added (Weapons;CoreLib, 200;100), the service report is
	// clean. However SuggestLoadOrder now finds that CoreLib (prerequisite) should load
	// before Weapons (dependent) - so a load-order advisory appears instead of "no problems".
	waitForText(t, tm, "could be reordered")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // dirty now -> confirm
	waitForText(t, tm, "unsaved changes")
	tm.Send(keyRune('y'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	if !contains(m.s.Cfg.ServerMods().WorkshopItems, "100") {
		t.Errorf("WorkshopItems = %v; want 100 added by the fix", m.s.Cfg.ServerMods().WorkshopItems)
	}
}

// TestValidateMapOrderWarning drives the validate screen with a config whose
// Map= has a vanilla base map before a custom map (wrong order). It asserts:
//  1. A "Map order" warning row appears.
//  2. Pressing enter (fix) applies base-last ordering and shows a toast.
//  3. s.Cfg.Maps() is now base-last after the fix.
func TestValidateMapOrderWarning(t *testing.T) {
	// "Muldraugh, KY" is a vanilla base map; "Riverside" is a custom map name.
	// Base map is listed first - wrong order for map loading priority.
	tm, m := openedModelAt(t, steamtest.New(),
		"Mods=\nWorkshopItems=\nMap=Muldraugh, KY;Riverside\n",
		NewValidate())

	// Wait for the map-order warning row.
	waitForText(t, tm, "Map order")

	// Enter applies the fix.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// "map order updated" toast confirms the fix was applied.
	waitForText(t, tm, "map order updated")

	// Verify the base map is now last in the session config.
	maps := m.s.Cfg.Maps()
	if len(maps) != 2 {
		t.Fatalf("expected 2 maps after fix, got %v", maps)
	}
	if maps[len(maps)-1] != "Muldraugh, KY" {
		t.Errorf("expected base map last, got %v", maps)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // dirty → confirm
	waitForText(t, tm, "unsaved changes")
	tm.Send(keyRune('y'))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestValidateModOrderInfo drives the validate screen with a config where Weapons
// (dependent) appears before CoreLib (prerequisite) in the mod load order. It
// asserts the Info advisory appears, then presses enter - which navigates to the
// Load order screen without auto-rewriting the config.
//
// The scenario is fully deterministic: steamtest.Fake returns the seeded items
// synchronously, so SuggestLoadOrder always finds CoreLib should precede Weapons.
func TestValidateModOrderInfo(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "200",
			Title: "Weapons Pack", Description: "Mod ID: Weapons\n",
			Children: []steam.WorkshopItemChild{{PublishedFileID: "100"}}},
		steam.WorkshopItem{Result: 1, FileType: steam.FileTypeMod, PublishedFileID: "100",
			Title: "Core Library", Description: "Mod ID: CoreLib\n"},
	)
	// Wrong order: Weapons (dependent) before CoreLib (prerequisite).
	tm, m := openedModelAt(t, fake,
		"Mods=Weapons;CoreLib\nWorkshopItems=200;100\n",
		NewValidate())

	// Wait for the load-order info advisory.
	waitForText(t, tm, "could be reordered")

	// Enter navigates to the Load order screen (no auto-rewrite).
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForText(t, tm, "switch tab") // hint unique to the Load order screen

	// The mod-order advisory must NAVIGATE, never rewrite Mods= itself.
	if mods := m.s.Cfg.ServerMods().Mods; len(mods) != 2 || mods[0] != "Weapons" || mods[1] != "CoreLib" {
		t.Errorf("modorder advisory must not rewrite Mods=; got %v", mods)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestValidateFilterNarrows(t *testing.T) {
	// Two unknown-mod-ID findings: "mod ID Foo ..." and "mod ID Bar ...".
	tm, _ := openedModelAt(t, steamtest.New(),
		"Mods=Foo;Bar\nWorkshopItems=\nMap=\n", NewValidate())
	waitForText(t, tm, "Foo")

	tm.Send(keyRune('/'))
	for _, r := range "bar" {
		tm.Send(keyRune(r))
	}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("filter: bar")) && bytes.Contains(b, []byte("mod ID Bar"))
	}, teatest.WithDuration(3*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
