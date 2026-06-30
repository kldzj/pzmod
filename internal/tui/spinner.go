package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// loader is a tiny wrapper around a bubbles spinner for consistent loading
// indicators across screens.
type loader struct {
	sp spinner.Model
}

func newLoader() loader {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return loader{sp: sp}
}

// tick starts the spinner animation.
func (l loader) tick() tea.Cmd { return l.sp.Tick }

// update advances the spinner if msg is a spinner tick, reporting whether it
// handled the message (so the caller can return early).
func (l *loader) update(msg tea.Msg) (tea.Cmd, bool) {
	if _, ok := msg.(spinner.TickMsg); !ok {
		return nil, false
	}
	var cmd tea.Cmd
	l.sp, cmd = l.sp.Update(msg)
	return cmd, true
}

// view renders the spinner followed by a muted label.
func (l loader) view(th Theme, label string) string {
	return l.sp.View() + th.Muted.Render(" "+label)
}
