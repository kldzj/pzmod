package tui

import (
	"context"
	"os"

	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// backupDiff shows a unified diff between a backup and the current config file.
type backupDiff struct {
	profileID, backupID, path string
	diff                      string
	vp                        viewport.Model
	loading                   bool
	ready                     bool
}

// NewBackupDiff returns a screen diffing a backup against the live config.
func NewBackupDiff(profileID, backupID, path string) Screen {
	return &backupDiff{profileID: profileID, backupID: backupID, path: path, loading: true}
}

func (bd *backupDiff) Title() string { return "Backup diff" }

type diffLoadedMsg struct {
	diff string
	err  error
}

func (bd *backupDiff) Init(s *Session) tea.Cmd {
	return s.Do(func(ctx context.Context) tea.Msg {
		backupBytes, err := s.Store.ReadBackup(bd.profileID, bd.backupID)
		if err != nil {
			return diffLoadedMsg{err: err}
		}
		current, err := os.ReadFile(bd.path)
		if err != nil {
			return diffLoadedMsg{err: err}
		}
		d := udiff.Unified("backup", "current", string(backupBytes), string(current))
		return diffLoadedMsg{diff: d}
	})
}

func (bd *backupDiff) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		bd.resize(s)
		return bd, nil
	case diffLoadedMsg:
		if msg.err != nil {
			return bd, tea.Batch(Fail(msg.err), Pop())
		}
		bd.loading = false
		bd.diff = msg.diff
		bd.resize(s)
		return bd, nil
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return bd, Pop()
		}
	}
	if bd.ready {
		var cmd tea.Cmd
		bd.vp, cmd = bd.vp.Update(msg)
		return bd, cmd
	}
	return bd, nil
}

func (bd *backupDiff) resize(s *Session) {
	bd.vp = viewport.New(max(20, s.Width-2), max(3, s.BodyHeight()-2))
	bd.ready = true
	bd.vp.SetContent(bd.renderDiff(s))
}

func (bd *backupDiff) renderDiff(s *Session) string {
	th := s.Theme
	if bd.diff == "" {
		return th.OK.Render("identical - this backup matches the current config")
	}
	var out []byte
	for _, line := range splitLinesKeep(bd.diff) {
		switch {
		case len(line) > 0 && line[0] == '+':
			out = append(out, th.OK.Render(line)...)
		case len(line) > 0 && line[0] == '-':
			out = append(out, th.Error.Render(line)...)
		case len(line) > 0 && line[0] == '@':
			out = append(out, th.Subtitle.Render(line)...)
		default:
			out = append(out, th.Muted.Render(line)...)
		}
		out = append(out, '\n')
	}
	return string(out)
}

func (bd *backupDiff) View(s *Session) string {
	th := s.Theme
	if bd.loading {
		return pad(th.Muted.Render("computing diff…"))
	}
	return pad(bd.vp.View() + "\n" + th.Muted.Render("↑/↓: scroll   esc: back"))
}

func splitLinesKeep(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
