package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/pkg/steam/steamtest"
)

func TestMapsTabSuggestPutsBaseLast(t *testing.T) {
	tm, m := openProfileModelWith(t, steamtest.New(), "Mods=A\nWorkshopItems=\nMap=Muldraugh, KY;Riverside\n")
	tm.Send(PushMsg{Screen: NewLoadOrder()})
	waitForText(t, tm, "Load order")

	// Switch to the Maps tab.
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	waitForText(t, tm, "Riverside")

	// Suggest -> preview -> apply.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	waitForText(t, tm, "Suggested")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	waitForText(t, tm, "applied")

	maps := m.s.Cfg.Maps()
	if len(maps) != 2 || maps[len(maps)-1] != "Muldraugh, KY" {
		t.Fatalf("expected base map last, got %v", maps)
	}
}
