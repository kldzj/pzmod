// Package tui is pzmod's Bubble Tea terminal application. It uses the Elm
// Model-Update-View pattern with a screen stack (router) over a shared Session.
// All Steam and disk work goes through the injected service layer via tea.Cmds,
// keeping the UI responsive.
package tui

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kldzj/pzmod/internal/service"
	"github.com/kldzj/pzmod/internal/store"
)

type confirmState struct {
	prompt    string
	onConfirm tea.Cmd
}

type model struct {
	s     *Session
	stack []Screen

	help       help.Model
	showHelp   bool
	toast      string
	toastErr   bool
	toastToken int
	confirm    *confirmState
	mouse      bool
	quitting   bool
}

// New builds the root model with an initial screen.
func New(svc *service.Services, st *store.Store, ctx context.Context, initial Screen) *model {
	h := help.New()
	return &model{
		s: &Session{
			Svc:    svc,
			Store:  st,
			Ctx:    ctx,
			Theme:  DefaultTheme(),
			Keys:   DefaultKeyMap(),
			Width:  80,
			Height: 24,
		},
		stack: []Screen{initial},
		help:  h,
	}
}

// Run starts the program on the alt screen. The session uses ctx for in-flight
// Steam work; interrupts are handled by the model's guard, not ctx cancellation.
func Run(svc *service.Services, st *store.Store, ctx context.Context, initial Screen, mouse bool) error {
	return runProgram(New(svc, st, ctx, initial), mouse)
}

// RunOpen starts the app with a profile already opened, going straight to the
// dashboard (used for `pzmod --file` / `--profile`).
func RunOpen(svc *service.Services, st *store.Store, ctx context.Context, profile store.Profile, mouse bool) error {
	m := New(svc, st, ctx, NewDashboard())
	if err := m.s.OpenProfile(profile); err != nil {
		return err
	}
	return runProgram(m, mouse)
}

func runProgram(m *model, mouse bool) error {
	m.mouse = mouse
	// Intentionally NOT wiring an os.Interrupt-aware context to tea.WithContext:
	// that would tear the program down on ctrl+c WITHOUT running Update, bypassing
	// the unsaved-changes guard. Instead we catch interrupts ourselves and deliver
	// them as interruptMsg so the guard is always authoritative - whether ctrl+c
	// arrives as a key (raw mode) or as a SIGINT.
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if mouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}
	p := tea.NewProgram(m, opts...)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	go func() {
		for range sig {
			p.Send(interruptMsg{})
		}
	}()

	_, err := p.Run()
	return err
}

func (m *model) active() Screen { return m.stack[len(m.stack)-1] }

func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.active().Init(m.s), checkUpdateCmd()}
	pid := ""
	if m.s.Profile != nil {
		pid = m.s.Profile.ID
	}
	if !m.s.Store.HasAPIKey(pid) {
		cmds = append(cmds, Push(NewSettings()))
	}
	return tea.Batch(cmds...)
}

func (m *model) delegate(msg tea.Msg) tea.Cmd {
	next, cmd := m.active().Update(m.s, msg)
	m.stack[len(m.stack)-1] = next
	return cmd
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.s.Width = msg.Width
		m.s.Height = msg.Height
		m.help.Width = msg.Width
		return m, m.delegate(msg)

	case tea.MouseMsg:
		// Opt-in mouse: translate the wheel to list navigation.
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			return m, m.delegate(tea.KeyMsg{Type: tea.KeyUp})
		case tea.MouseButtonWheelDown:
			return m, m.delegate(tea.KeyMsg{Type: tea.KeyDown})
		}
		return m, nil

	case tea.KeyMsg:
		if m.confirm != nil {
			return m.updateConfirm(msg)
		}
		if m.showHelp {
			m.showHelp = false // any key dismisses help
			return m, nil
		}
		switch {
		case key.Matches(msg, m.s.Keys.Quit):
			return m, m.requestQuit()
		case key.Matches(msg, m.s.Keys.Help):
			m.showHelp = true
			return m, nil
		case key.Matches(msg, m.s.Keys.Save):
			if m.s.Cfg == nil {
				return m, nil
			}
			if _, isConfirm := m.active().(*saveConfirm); isConfirm {
				return m, nil // already confirming
			}
			if !m.s.Cfg.HasUnsavedChanges() {
				return m, Toast("nothing to save")
			}
			return m, Push(NewSaveConfirm())
		}
		return m, m.delegate(msg)

	case PushMsg:
		m.stack = append(m.stack, msg.Screen)
		return m, msg.Screen.Init(m.s)
	case PopMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
		}
		return m, func() tea.Msg { return resumedMsg{} }
	case modsChangedMsg:
		var cmds []tea.Cmd
		if msg.toast != "" {
			m.toast, m.toastErr = msg.toast, false
			m.toastToken++
			cmds = append(cmds, m.scheduleToastClear())
		}
		cmds = append(cmds, m.delegate(msg))
		return m, tea.Batch(cmds...)
	case ReplaceMsg:
		m.stack[len(m.stack)-1] = msg.Screen
		return m, msg.Screen.Init(m.s)
	case ToastMsg:
		m.toast, m.toastErr = msg.Text, false
		m.toastToken++
		return m, m.scheduleToastClear()
	case ErrMsg:
		if msg.Err != nil {
			m.toast, m.toastErr = msg.Err.Error(), true
			m.toastToken++
			return m, m.scheduleToastClear()
		}
		return m, nil
	case clearToastMsg:
		if msg.token == m.toastToken {
			m.toast = ""
		}
		return m, nil
	case ConfirmMsg:
		m.confirm = &confirmState{prompt: msg.Prompt, onConfirm: msg.OnConfirm}
		return m, nil
	case updateAvailableMsg:
		// A newer release exists: keep a persistent hint in the top bar and
		// surface a one-time toast pointing at the self-update command.
		m.s.UpdateLatest = msg.latest
		m.toast, m.toastErr = "update available: "+msg.latest+" - run pzmod update", false
		m.toastToken++
		return m, m.scheduleToastClear()
	case interruptMsg:
		// Route OS interrupts through the guard. A second interrupt while a
		// dialog is up forces an immediate quit (escape hatch).
		if m.confirm != nil {
			return m, tea.Quit
		}
		return m, m.requestQuit()
	case quitMsg:
		m.quitting = true
		return m, tea.Quit

	default:
		return m, m.delegate(msg)
	}
}

func (m *model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "ctrl+c":
		return m, tea.Quit // force quit from any confirmation dialog
	case "y", "enter":
		cmd := m.confirm.onConfirm
		m.confirm = nil
		return m, cmd
	case "n", "esc":
		m.confirm = nil
		return m, nil
	}
	return m, nil
}

func (m *model) requestQuit() tea.Cmd {
	if m.s.Dirty() {
		return Confirm("You have unsaved changes. Quit anyway?", tea.Quit)
	}
	return tea.Quit
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}
	th := m.s.Theme
	w, h := m.s.Width, m.s.Height

	top := m.renderTopBar()
	footer := m.renderFooter()
	bodyH := h - lipgloss.Height(top) - lipgloss.Height(footer)
	if bodyH < 1 {
		bodyH = 1
	}

	var body string
	switch {
	case m.confirm != nil:
		body = lipgloss.Place(w, bodyH, lipgloss.Center, lipgloss.Center, m.confirmBox())
	case m.showHelp:
		help := th.Box.Render(m.help.FullHelpView(m.s.Keys.FullHelp()))
		body = lipgloss.Place(w, bodyH, lipgloss.Center, lipgloss.Center, help)
	default:
		body = lipgloss.NewStyle().Height(bodyH).MaxHeight(bodyH).Render(m.active().View(m.s))
	}

	return lipgloss.JoinVertical(lipgloss.Left, top, body, footer)
}

func (m *model) confirmBox() string {
	th := m.s.Theme
	buttons := th.Badge.Render(" Yes ") + th.Muted.Render("  (y)    ") + th.Subtitle.Render("No") + th.Muted.Render(" (n/esc)")
	return th.Box.Render(m.confirm.prompt + "\n\n" + buttons)
}

func (m *model) renderTopBar() string {
	th := m.s.Theme
	// Each segment carries the bar background so its reset doesn't expose a gap;
	// the Brand badge keeps its own accent background.
	on := func(fg lipgloss.TerminalColor, s string) string {
		return lipgloss.NewStyle().Foreground(fg).Background(colBarBG).Render(s)
	}
	left := th.Brand.Render("pzmod") + on(colMuted, " › "+m.active().Title())

	var right string
	if m.s.UpdateLatest != "" {
		right += on(colAccent, "↑ "+m.s.UpdateLatest)
	}
	if m.s.Profile != nil {
		name := m.s.Profile.Name
		if name == "" {
			name = m.s.Profile.ID
		}
		if right != "" {
			right += on(colMuted, "  ")
		}
		right += on(colMuted, name)
		if m.s.Dirty() {
			right += on(colWarn, "  ● unsaved")
		}
	}

	gap := m.s.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	fill := lipgloss.NewStyle().Background(colBarBG).Render(strings.Repeat(" ", gap))
	return left + fill + right
}

func (m *model) renderFooter() string {
	th := m.s.Theme
	// Toast line (reserved so the layout doesn't jump).
	toast := " "
	if m.toast != "" {
		if m.toastErr {
			toast = th.ToastErr.Render("✗ " + m.toast)
		} else {
			toast = th.ToastOK.Render("✓ " + m.toast)
		}
	}
	hint := "? help · ctrl+s save · esc back · ctrl+c quit"
	bar := th.BottomBar.Width(m.s.Width).Render(" " + hint)
	return lipgloss.JoinVertical(lipgloss.Left, " "+toast, bar)
}

// modsChangedMsg is emitted after an operation mutates the in-memory mod lists.
// It surfaces an optional toast and is delegated to the active screen so a list
// view can refresh itself.
type modsChangedMsg struct{ toast string }

// resumedMsg is delivered to the now-active screen after a Pop, so a screen can
// refresh state that may have changed while a child screen was on top.
type resumedMsg struct{}

// quitMsg requests a clean shutdown.
type quitMsg struct{}

// interruptMsg is delivered when the process receives an OS interrupt (SIGINT,
// e.g. ctrl+c when the terminal generates a signal rather than a key). It is
// routed through the model so the unsaved-changes guard always applies, instead
// of letting the program tear down via context cancellation.
type interruptMsg struct{}

// clearToastMsg expires a toast if it hasn't been superseded.
type clearToastMsg struct{ token int }

func (m *model) scheduleToastClear() tea.Cmd {
	tok := m.toastToken
	d := 4 * time.Second
	if m.toastErr {
		d = 8 * time.Second
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return clearToastMsg{token: tok} })
}
