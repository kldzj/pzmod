package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/steam/steamtest"
)

// TestAddByIDRefreshesInstalledList asserts on a list-only token (the new
// item's file size "5.2 MB") that the success toast never contains, so it
// actually guards the refresh regression rather than matching the
// "added NewMod (unsaved)" toast string. The item keeps a single mod ID and no
// maps so quickAdd takes the instant-add path (not the add sheet).
// IDs are ≥6 digits to satisfy the reWorkshopID `\d{6,}` regex.
func TestAddByIDRefreshesInstalledList(t *testing.T) {
	fake := steamtest.New(
		steam.WorkshopItem{PublishedFileID: "100100", Title: "Basements", Description: "Mod ID: Basements"},
		// FileSize renders as "5.2 MB" in the installed row only; it never
		// appears in the toast, and Basements (size 0) renders "0 B".
		steam.WorkshopItem{PublishedFileID: "200200", Title: "NewMod", Description: "Mod ID: NewMod", FileSize: 5242880},
	)
	tm, _ := openProfileModelWith(t, fake, "WorkshopItems=100100\nMods=Basements\nMap=\n")
	tm.Send(PushMsg{Screen: NewInstalled()})
	waitForText(t, tm, "Basements")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	waitForText(t, tm, "Workshop ID")

	for _, r := range "200200" {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// "5.2 MB" can only render from the refreshed installed list row.
	waitForText(t, tm, "5.2 MB")
}
