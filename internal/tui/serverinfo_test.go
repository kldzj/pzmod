package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/pkg/serverconfig"
)

func newServerInfo(t *testing.T, body string) (*Session, *serverinfo) {
	t.Helper()
	cfg := serverconfig.FromBytes("server.ini", []byte(body))
	s := &Session{Cfg: cfg, Theme: DefaultTheme(), Width: 80, Height: 24}
	si := &serverinfo{}
	si.Init(s) // seeds the bound fields and builds the form
	return s, si
}

// TestServerInfoAppliesEditsOnEsc: editing a field then pressing esc ("back")
// must keep the change in the in-memory config, not discard it. huh writes the
// bound variable on every keystroke, so setting si.name mimics a real edit.
func TestServerInfoAppliesEditsOnEsc(t *testing.T) {
	s, si := newServerInfo(t, "PublicName=Old Name\nMods=\n")
	if si.name != "Old Name" {
		t.Fatalf("Init did not seed name: %q", si.name)
	}
	si.name = "New Name" // user typed a new name

	_, cmd := si.Update(s, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc after an edit should apply + pop (non-nil cmd)")
	}
	if got := s.Cfg.Name(); got != "New Name" {
		t.Fatalf("name not applied on esc: got %q want %q", got, "New Name")
	}
	if !s.Cfg.HasUnsavedChanges() {
		t.Fatal("edited config should be dirty after esc")
	}
}

// TestServerInfoEscWithoutEditsStaysClean: opening the editor and backing out
// without touching anything must not fabricate unsaved changes, even when the
// original line has spaces around '=' that the setter would re-render away.
func TestServerInfoEscWithoutEditsStaysClean(t *testing.T) {
	s, si := newServerInfo(t, "PublicName = Old Name\nMods=\n")
	if _, cmd := si.Update(s, tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil {
		t.Fatal("esc should still pop (non-nil cmd)")
	}
	if s.Cfg.HasUnsavedChanges() {
		t.Fatal("esc without edits must not mark the config dirty")
	}
}
