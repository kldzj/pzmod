package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/kldzj/pzmod/internal/steam/steamtest"
)

func TestBackupsSnapshotAndRestore(t *testing.T) {
	tm, m := openedModelAt(t, steamtest.New(), "Mods=a\nWorkshopItems=\n", NewBackups())

	waitForText(t, tm, "no backups yet")

	tm.Send(keyRune('s')) // snapshot now
	waitForText(t, tm, "[manual]")

	tm.Send(keyRune('r')) // restore -> confirm modal
	waitForText(t, tm, "Restore")
	tm.Send(keyRune('y')) // confirm
	waitForText(t, tm, "restored")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // restored == same bytes -> not dirty
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	// At least the manual snapshot (plus a pre-restore safety snapshot) exist.
	entries, err := m.s.Store.Backups(m.s.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Errorf("expected >=2 backups (manual + pre-restore), got %d", len(entries))
	}
}
