package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAddSheetAppliesSelected(t *testing.T) {
	tm, m := openProfileModel(t, false)
	waitForText(t, tm, "Server info")

	sheet := newAddSheetWith("100", "Pack", []addRow{
		{label: "ModA", value: "ModA", isMap: false, on: true},
		{label: "ModB", value: "ModB", isMap: false, on: true},
		{label: "MapX", value: "MapX", isMap: true, on: true},
	})
	tm.Send(PushMsg{Screen: sheet})
	waitForText(t, tm, "Pack")

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}) // toggle MapX off
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitForText(t, tm, "added")

	sm := m.s.Cfg.ServerMods()
	if !sm.HasItem("100") || !sm.HasMod("ModA") || !sm.HasMod("ModB") {
		t.Fatalf("expected item+mods added: %+v", sm)
	}
	if sm.HasMap("MapX") {
		t.Fatal("MapX was toggled off and must not be added")
	}
}
