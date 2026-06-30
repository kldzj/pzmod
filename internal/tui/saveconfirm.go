package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/internal/domain"
	"github.com/kldzj/pzmod/internal/pathutil"
	"github.com/kldzj/pzmod/internal/serverconfig"
)

// saveConfirm shows a change summary before writing, with a diff one key away.
type saveConfirm struct {
	summary    serverconfig.Summary
	disk       string // on-disk file content, cached for the diff view
	unreadable bool
}

// NewSaveConfirm returns the pre-save confirmation screen.
func NewSaveConfirm() Screen { return &saveConfirm{} }

func (sc *saveConfirm) Title() string { return "Save changes" }

func (sc *saveConfirm) Init(s *Session) tea.Cmd {
	if s.Cfg == nil {
		return nil
	}
	// One cheap local read: used both for the summary baseline and the diff view.
	b, err := os.ReadFile(s.Cfg.Path())
	if err != nil {
		sc.unreadable = true
		b = nil // treat as all-added
	}
	sc.disk = string(b)
	sc.summary = serverconfig.Summarize(serverconfig.FromBytes(s.Cfg.Path(), b), s.Cfg)
	return nil
}

func (sc *saveConfirm) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return sc, nil
	}
	switch key.String() {
	case "esc", "n":
		return sc, Pop()
	case "s", "enter", "y":
		return sc, tea.Batch(Pop(), saveCmd(s))
	case "d":
		return sc, Push(NewDiffView("Diff", sc.disk, s.Cfg.String()))
	}
	return sc, nil
}

func (sc *saveConfirm) View(s *Session) string {
	th := s.Theme
	var b strings.Builder
	b.WriteString(th.Subtitle.Render("Save changes to "+pathutil.Abbreviate(s.Cfg.Path())+"?") + "\n\n")
	if sc.unreadable {
		b.WriteString(th.Warn.Render("could not read existing file - treating as new") + "\n\n")
	}
	if sc.summary.Empty() {
		b.WriteString(th.Muted.Render("no changes") + "\n\n")
	} else {
		b.WriteString(deltaLine(th, "Mods", sc.summary.Mods))
		b.WriteString(deltaLine(th, "WorkshopItems", sc.summary.WorkshopItems))
		b.WriteString(deltaLine(th, "Map", sc.summary.Maps))
		for _, f := range sc.summary.ChangedFields {
			b.WriteString("  " + th.Item.Render(f) + th.Muted.Render(" changed") + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(th.Muted.Render("s: save   d: view diff   esc: cancel"))
	return pad(b.String())
}

func deltaLine(th Theme, label string, d domain.Delta) string {
	if d.Empty() {
		return ""
	}
	var parts []string
	if d.Added > 0 {
		parts = append(parts, th.OK.Render(fmt.Sprintf("+%d", d.Added)))
	}
	if d.Removed > 0 {
		parts = append(parts, th.Error.Render(fmt.Sprintf("-%d", d.Removed)))
	}
	if d.Reordered {
		parts = append(parts, th.Warn.Render("reordered"))
	}
	return "  " + th.Item.Render(fmt.Sprintf("%-14s", label)) + strings.Join(parts, " ") + "\n"
}
