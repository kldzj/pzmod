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
	// esc is "back": keep whatever the user edited (huh binds the fields live) and
	// return, rather than discarding it. Unchanged fields are left untouched so
	// merely opening this screen and backing out never fabricates unsaved changes.
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" {
		return si, si.commit(s)
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
		return si, si.commit(s)
	case huh.StateAborted:
		return si, Pop()
	}
	return si, cmd
}

// commit writes back any fields that actually changed and returns the command to
// leave the screen (with a toast only when something was applied). Comparing
// against the current values keeps an untouched exit from marking the config
// dirty, since the setter re-renders lines and can differ from the on-disk bytes.
func (si *serverinfo) commit(s *Session) tea.Cmd {
	cfg := s.Cfg
	changed := false
	set := func(cur, next string, apply func(string)) {
		if next != cur {
			apply(next)
			changed = true
		}
	}
	set(cfg.Name(), si.name, cfg.SetName)
	set(cfg.Description(), encodeLINE(si.desc), cfg.SetDescription)
	set(cfg.Password(), si.password, cfg.SetPassword)
	set(cfg.MaxPlayers(), si.slots, cfg.SetMaxPlayers)
	if si.public != cfg.Public() {
		cfg.SetPublic(si.public)
		changed = true
	}
	if changed {
		return tea.Batch(Toast("server info updated (unsaved)"), Pop())
	}
	return Pop()
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
