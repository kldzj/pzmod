package tui

import (
	"strings"

	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// diffView is a scrollable unified-diff viewer.
type diffView struct {
	title string
	old   string
	new   string
	vp    viewport.Model
	ready bool
}

// NewDiffView shows a unified diff of old vs new under title.
func NewDiffView(title, old, new string) Screen {
	return &diffView{title: title, old: old, new: new}
}

func (d *diffView) Title() string { return d.title }

func (d *diffView) Init(s *Session) tea.Cmd { return nil }

func (d *diffView) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.resize(s)
		return d, nil
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return d, Pop()
		}
	}
	if d.ready {
		var cmd tea.Cmd
		d.vp, cmd = d.vp.Update(msg)
		return d, cmd
	}
	return d, nil
}

func (d *diffView) resize(s *Session) {
	d.vp = viewport.New(max(20, s.Width-2), max(3, s.BodyHeight()-2))
	d.vp.SetContent(d.render(s))
	d.ready = true
}

func (d *diffView) render(s *Session) string {
	th := s.Theme
	unified := udiff.Unified("on disk", "in memory", d.old, d.new)
	var b strings.Builder
	for _, line := range splitLinesKeep(unified) {
		switch {
		case strings.HasPrefix(line, "+"):
			b.WriteString(th.OK.Render(line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(th.Error.Render(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(th.Subtitle.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (d *diffView) View(s *Session) string {
	if !d.ready {
		d.resize(s)
	}
	return pad(d.vp.View() + "\n" + s.Theme.Muted.Render("↑/↓: scroll   esc: back"))
}
