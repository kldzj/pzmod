package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// infoList is a read-only, scrollable list of lines (e.g. unavailable item IDs).
type infoList struct {
	title string
	lines []string
	vp    viewport.Model
	ready bool
}

// NewInfoList returns a scrollable read-only list screen.
func NewInfoList(title string, lines []string) Screen { return &infoList{title: title, lines: lines} }

func (l *infoList) Title() string { return l.title }

func (l *infoList) Init(s *Session) tea.Cmd { return nil }

func (l *infoList) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.resize(s)
		return l, nil
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return l, Pop()
		}
	}
	if l.ready {
		var cmd tea.Cmd
		l.vp, cmd = l.vp.Update(msg)
		return l, cmd
	}
	return l, nil
}

func (l *infoList) resize(s *Session) {
	l.vp = viewport.New(max(20, s.Width-2), max(3, s.BodyHeight()-2))
	l.vp.SetContent(strings.Join(l.lines, "\n"))
	l.ready = true
}

func (l *infoList) View(s *Session) string {
	if !l.ready {
		l.resize(s)
	}
	return pad(l.vp.View() + "\n" + s.Theme.Muted.Render("↑/↓: scroll   esc: back"))
}
