package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// serverinfo edits the common server fields via a huh form.
type serverinfo struct {
	form *huh.Form

	name, desc, password, slots string
	public                      bool
}

// NewServerInfo returns the server-info editor.
func NewServerInfo() Screen { return &serverinfo{} }

func (si *serverinfo) Title() string { return "Server info" }

func (si *serverinfo) Init(s *Session) tea.Cmd {
	cfg := s.Cfg
	si.name = cfg.Name()
	si.desc = decodeLINE(cfg.Description())
	si.public = cfg.Public()
	si.password = cfg.Password()
	si.slots = cfg.MaxPlayers()

	si.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Public name").Value(&si.name),
			huh.NewText().Title("Description").Value(&si.desc),
			huh.NewConfirm().Title("Public listing").Value(&si.public),
			huh.NewInput().Title("Password").Value(&si.password),
			huh.NewInput().Title("Max players").Value(&si.slots),
		),
	).WithWidth(min(64, max(20, s.Width-4))).WithShowHelp(true)

	return si.form.Init()
}

func (si *serverinfo) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" {
		return si, Pop()
	}
	if si.form == nil {
		return si, nil
	}
	fm, cmd := si.form.Update(msg)
	if f, ok := fm.(*huh.Form); ok {
		si.form = f
	}
	switch si.form.State {
	case huh.StateCompleted:
		cfg := s.Cfg
		cfg.SetName(si.name)
		cfg.SetDescription(encodeLINE(si.desc))
		cfg.SetPublic(si.public)
		cfg.SetPassword(si.password)
		cfg.SetMaxPlayers(si.slots)
		return si, tea.Batch(Toast("server info updated (unsaved)"), Pop())
	case huh.StateAborted:
		return si, Pop()
	}
	return si, cmd
}

func (si *serverinfo) View(s *Session) string {
	if si.form == nil {
		return ""
	}
	return pad(si.form.View())
}

func decodeLINE(s string) string {
	r := strings.ReplaceAll(s, "<LINE>", "\n")
	return strings.ReplaceAll(r, "<line>", "\n")
}

func encodeLINE(s string) string {
	return strings.ReplaceAll(s, "\n", "<LINE>")
}
