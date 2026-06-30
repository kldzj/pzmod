package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// addByID accepts a numeric Workshop ID or a Workshop URL and adds it.
type addByID struct {
	input textinput.Model
	err   string
}

var reWorkshopID = regexp.MustCompile(`\d{6,}`)

// NewAddByID returns the add-by-ID input screen.
func NewAddByID() Screen {
	ti := textinput.New()
	ti.Placeholder = "Workshop ID or URL"
	ti.Focus()
	ti.Width = 50
	return &addByID{input: ti}
}

func (a *addByID) Title() string { return "Add by ID" }

func (a *addByID) Init(s *Session) tea.Cmd { return textinput.Blink }

func (a *addByID) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	// Handle modsChangedMsg before key processing: quickAdd took the instant-add
	// path and has already mutated the config; close this screen so the installed
	// list behind us resumes (and refreshes via resumedMsg).
	if _, ok := msg.(modsChangedMsg); ok {
		return a, Pop()
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return a, Pop()
		case "enter":
			id := parseWorkshopID(a.input.Value())
			if id == "" {
				a.err = "enter a numeric ID or a Workshop URL"
				return a, nil
			}
			// replaceTop=true: sheet/deps replaces the addByID screen atomically,
			// eliminating the Pop()+PushMsg race. Instant-add returns modsChangedMsg
			// which is caught above to pop this screen.
			return a, quickAdd(s, id, true)
		}
	}
	var cmd tea.Cmd
	a.input, cmd = a.input.Update(msg)
	return a, cmd
}

func parseWorkshopID(in string) string {
	in = strings.TrimSpace(in)
	if m := reWorkshopID.FindString(in); m != "" {
		return m
	}
	return ""
}

func (a *addByID) View(s *Session) string {
	th := s.Theme
	var b strings.Builder
	b.WriteString(th.Subtitle.Render("Add a mod or collection by ID") + "\n\n")
	b.WriteString(a.input.View() + "\n")
	if a.err != "" {
		b.WriteString("\n" + th.Error.Render(a.err) + "\n")
	}
	b.WriteString("\n" + th.Muted.Render("enter: add   esc: cancel"))
	return pad(b.String())
}
