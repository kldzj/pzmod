package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/pkg/build"
	"github.com/kldzj/pzmod/pkg/domain"
)

// dashboard is the main menu for the open profile.
type dashboard struct {
	menu      menu
	actions   []func(s *Session) tea.Cmd
	shortcuts map[string]int // key -> entry index
}

// NewDashboard returns the main menu screen. A profile must already be open.
func NewDashboard() Screen {
	d := &dashboard{shortcuts: map[string]int{}}
	type entry struct {
		key, label, desc string
		action           func(s *Session) tea.Cmd
	}
	entries := []entry{
		{"m", "Installed Mods", "view, remove, add by ID", func(s *Session) tea.Cmd { return Push(NewInstalled()) }},
		{"s", "Search Workshop", "find and add mods", func(s *Session) tea.Cmd { return Push(NewSearch()) }},
		{"l", "Load order", "suggest and apply a load order", func(s *Session) tea.Cmd { return Push(NewLoadOrder()) }},
		{"v", "Validate", "check dependencies and problems", func(s *Session) tea.Cmd { return Push(NewValidate()) }},
		{"b", "Backups", "snapshot and restore", func(s *Session) tea.Cmd { return Push(NewBackups()) }},
		{"i", "Server info", "name, description, slots", func(s *Session) tea.Cmd { return Push(NewServerInfo()) }},
		{",", "Settings", "Steam API key", func(s *Session) tea.Cmd { return Push(NewSettings()) }},
		{"ctrl+s", "Save config", "write changes to disk", func(s *Session) tea.Cmd {
			if s.Cfg == nil {
				return nil
			}
			if !s.Cfg.HasUnsavedChanges() {
				return Toast("nothing to save")
			}
			return Push(NewSaveConfirm())
		}},
		{"esc", "Switch profile", "back to the profile list", func(s *Session) tea.Cmd { return Pop() }},
	}
	items := make([]menuItem, len(entries))
	d.actions = make([]func(s *Session) tea.Cmd, len(entries))
	for i, e := range entries {
		items[i] = menuItem{Label: fmt.Sprintf("%-5s %s", "["+e.key+"]", e.label), Desc: e.desc}
		d.actions[i] = e.action
		d.shortcuts[e.key] = i
	}
	d.menu.setItems(items)
	return d
}

func (d *dashboard) Title() string { return "Dashboard" }

// dashboardValidatedMsg carries the result of the one-shot background validation
// that fires when a profile is first opened.
type dashboardValidatedMsg struct{ report domain.Report }

func (d *dashboard) Init(s *Session) tea.Cmd {
	if s.Cfg == nil || s.Validated {
		return nil
	}
	s.Validated = true
	sm := s.Cfg.ServerMods()
	b := s.Build()
	return s.Do(func(ctx context.Context) tea.Msg {
		report, err := s.Svc.Validate(ctx, sm, b)
		if err != nil {
			return nil
		}
		return dashboardValidatedMsg{report: report}
	})
}

func (d *dashboard) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	if vm, ok := msg.(dashboardValidatedMsg); ok {
		r := vm.report
		s.LastValidation = &r
		return d, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return d, nil
	}
	switch key.String() {
	case "up", "k":
		d.menu.up()
		return d, nil
	case "down", "j":
		d.menu.down()
		return d, nil
	case "enter":
		return d, d.actions[d.menu.selected()](s)
	}
	if idx, ok := d.shortcuts[key.String()]; ok {
		return d, d.actions[idx](s)
	}
	return d, nil
}

func (d *dashboard) View(s *Session) string {
	th := s.Theme
	var b strings.Builder
	b.WriteString(logoCompact(th) + "\n\n")

	if s.Build() == build.B42 {
		b.WriteString(th.Badge.Render("BUILD 42") + " " +
			th.Muted.Render("multiplayer still disables mods as of now") + "\n\n")
	}
	if s.Cfg != nil {
		sm := s.Cfg.ServerMods()
		b.WriteString(th.Subtitle.Render(s.Cfg.Name()))
		b.WriteString("  ")
		b.WriteString(th.Muted.Render(fmt.Sprintf("%d mods · %d items · build %s",
			len(sm.Mods), len(sm.WorkshopItems), s.Build().Label())))
		b.WriteString("\n")
		b.WriteString(validationSummary(s) + "\n\n")
	}
	b.WriteString(d.menu.View(th, s.ContentWidth()))
	return pad(b.String())
}

// validationSummary renders a one-line status from the last validation run.
func validationSummary(s *Session) string {
	th := s.Theme
	r := s.LastValidation
	if r == nil {
		return th.Muted.Render("not validated yet - press v")
	}
	e := r.Count(domain.SeverityError)
	w := r.Count(domain.SeverityWarning)
	switch {
	case e > 0:
		return th.Error.Render(fmt.Sprintf("⚠ %d error(s), %d warning(s) (last check)", e, w))
	case w > 0:
		return th.Warn.Render(fmt.Sprintf("%d warning(s) (last check)", w))
	default:
		return th.OK.Render("✓ no problems (last check)")
	}
}

// saveCmd snapshots the config then writes it.
func saveCmd(s *Session) tea.Cmd {
	return func() tea.Msg {
		if s.Cfg == nil {
			return nil
		}
		if !s.Cfg.HasUnsavedChanges() {
			return ToastMsg{Text: "nothing to save"}
		}
		if s.Profile != nil {
			if _, err := s.Svc.SnapshotProfile(*s.Profile, "before save", "auto"); err != nil {
				return ErrMsg{Err: err}
			}
		}
		if err := s.Cfg.Save(); err != nil {
			return ErrMsg{Err: err}
		}
		return ToastMsg{Text: "saved"}
	}
}
