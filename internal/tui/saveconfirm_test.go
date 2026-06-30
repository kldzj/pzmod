package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSaveConfirmShowsSummaryAndSaves(t *testing.T) {
	tm, m := openProfileModel(t, true) // dirty: Mods=[SomeMod]
	waitForText(t, tm, "Server info")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlS})
	waitForText(t, tm, "Mods") // summary frame (contains "Save changes" and "Mods")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	waitForText(t, tm, "saved")

	if m.s.Dirty() {
		t.Fatal("expected config saved (not dirty)")
	}
}
