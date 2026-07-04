package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/internal/pathutil"
	"github.com/kldzj/pzmod/pkg/store"
)

// launcher is the first screen: choose or create a profile.
type launcher struct {
	menu      menu
	profiles  []store.Profile
	defaultID string
	loaded    bool
}

// NewLauncher returns the profile-selection screen.
func NewLauncher() Screen { return &launcher{} }

func (l *launcher) Title() string { return "Profiles" }

type profilesLoadedMsg struct {
	profiles  []store.Profile
	defaultID string
}

// profilesChangedMsg asks the launcher to reload (after add/remove).
type profilesChangedMsg struct{}

func (l *launcher) Init(s *Session) tea.Cmd { return l.load(s) }

func (l *launcher) load(s *Session) tea.Cmd {
	return func() tea.Msg {
		ps, _ := s.Store.Profiles()
		def, _ := s.Store.DefaultProfile()
		return profilesLoadedMsg{profiles: ps, defaultID: def.ID}
	}
}

func (l *launcher) rebuild() {
	items := make([]menuItem, 0, len(l.profiles)+1)
	for _, p := range l.profiles {
		name := p.Name
		if name == "" {
			name = p.ID
		}
		items = append(items, menuItem{Label: name, Desc: pathutil.Abbreviate(p.IniPath)})
	}
	items = append(items, menuItem{Label: "＋ New profile…"})
	l.menu.setItems(items)
}

func (l *launcher) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		l.profiles = msg.profiles
		l.defaultID = msg.defaultID
		l.loaded = true
		l.rebuild()
		return l, nil
	case profilesChangedMsg:
		return l, l.load(s)
	case resumedMsg:
		// Returning from a pushed child (e.g. the add-profile form): reload so a
		// newly added or removed profile shows up. Cheap and idempotent.
		return l, l.load(s)
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			l.menu.up()
		case "down", "j":
			l.menu.down()
		case "n":
			return l, Push(NewProfileForm())
		case "d":
			if l.onProfile() {
				p := l.profiles[l.menu.selected()]
				st := s.Store
				del := func() tea.Msg {
					if err := st.RemoveProfile(p.ID); err != nil {
						return ErrMsg{Err: err}
					}
					return profilesChangedMsg{}
				}
				return l, Confirm(fmt.Sprintf("Delete profile %q?", p.ID), del)
			}
		case "enter":
			if l.onProfile() {
				p := l.profiles[l.menu.selected()]
				if err := s.OpenProfile(p); err != nil {
					return l, Fail(err)
				}
				return l, Push(NewDashboard())
			}
			return l, Push(NewProfileForm())
		}
	}
	return l, nil
}

func (l *launcher) onProfile() bool {
	return l.menu.selected() < len(l.profiles)
}

func (l *launcher) View(s *Session) string {
	th := s.Theme
	if !l.loaded {
		return pad(th.Muted.Render("Loading…"))
	}
	var b strings.Builder
	b.WriteString(renderLogo(th, s.ContentWidth(), s.BodyHeight()) + "\n\n")
	if len(l.profiles) == 0 {
		b.WriteString(th.Subtitle.Render("Welcome!") + "\n")
		b.WriteString(th.Muted.Render("No profiles yet - create one to manage a server's mods.") + "\n\n")
	} else {
		b.WriteString(th.Muted.Render("enter: open   n: new   d: delete") + "\n\n")
	}
	b.WriteString(l.menu.View(th, s.ContentWidth()))
	return pad(b.String())
}
