package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/internal/service"
	"github.com/kldzj/pzmod/internal/steam"
)

// settings edits the Steam API key (masked) with a live test ping.
type settings struct {
	input     textinput.Model
	hasKey    bool
	status    string
	statusErr bool
}

// NewSettings returns the API-key settings screen.
func NewSettings() Screen {
	ti := textinput.New()
	ti.Placeholder = "32-character Steam Web API key"
	ti.EchoMode = textinput.EchoPassword
	ti.CharLimit = 64
	ti.Width = 40
	ti.Focus()
	return &settings{input: ti}
}

func (st *settings) Title() string { return "Settings" }

func (st *settings) Init(s *Session) tea.Cmd {
	st.hasKey = s.Store.HasAPIKey("")
	return textinput.Blink
}

type settingsTestMsg struct{ err error }

func (st *settings) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case settingsTestMsg:
		if msg.err != nil {
			st.status, st.statusErr = "test failed: "+msg.err.Error(), true
		} else {
			st.status, st.statusErr = "API key works", false
		}
		return st, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return st, Pop()
		case "ctrl+t":
			return st, st.test(s)
		case "enter":
			return st, st.save(s)
		}
	}
	var cmd tea.Cmd
	st.input, cmd = st.input.Update(msg)
	return st, cmd
}

func (st *settings) keyToUse(s *Session) string {
	if v := strings.TrimSpace(st.input.Value()); v != "" {
		return v
	}
	k, _ := s.Store.APIKey("")
	return k
}

func (st *settings) test(s *Session) tea.Cmd {
	key := st.keyToUse(s)
	if key == "" {
		st.status, st.statusErr = "enter a key first", true
		return nil
	}
	return s.Do(func(ctx context.Context) tea.Msg {
		_, err := steam.New(key).QueryFiles(ctx, steam.Query{PerPage: 1})
		return settingsTestMsg{err: err}
	})
}

func (st *settings) save(s *Session) tea.Cmd {
	key := strings.TrimSpace(st.input.Value())
	if len(key) != 32 {
		st.status, st.statusErr = "key must be exactly 32 characters", true
		return nil
	}
	if err := s.Store.SetGlobalKey(key); err != nil {
		return Fail(err)
	}
	// Rebuild the service layer so subsequent Steam calls use the new key.
	s.Svc = service.New(steam.New(key), s.Store)
	// Return to the previous screen (the profile menu on first run, the dashboard
	// from the settings jump) and confirm via a toast, since this screen goes away.
	return tea.Batch(Toast("API key saved"), Pop())
}

func (st *settings) View(s *Session) string {
	th := s.Theme
	var b strings.Builder
	if st.hasKey {
		b.WriteString(th.OK.Render("✓ an API key is configured") + "\n\n")
	} else {
		b.WriteString(th.Warn.Render("no API key set") + "\n\n")
	}
	b.WriteString(th.Muted.Render("Enter a new key (https://steamcommunity.com/dev/apikey):") + "\n")
	b.WriteString(st.input.View() + "\n\n")
	b.WriteString(th.Muted.Render("enter: save   ctrl+t: test   esc: back") + "\n")
	if st.status != "" {
		if st.statusErr {
			b.WriteString("\n" + th.Error.Render(st.status))
		} else {
			b.WriteString("\n" + th.OK.Render(st.status))
		}
	}
	return pad(b.String())
}
