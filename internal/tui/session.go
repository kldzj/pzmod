package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kldzj/pzmod/pkg/build"
	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/pkg/serverconfig"
	"github.com/kldzj/pzmod/pkg/service"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/store"
)

// Session is the shared, mutable context threaded to every screen. It holds the
// injected services and the currently-open profile/config.
type Session struct {
	Svc   *service.Services
	Store *store.Store
	Ctx   context.Context

	Theme Theme
	Keys  KeyMap

	Width, Height int

	// Set when a profile is opened.
	Profile *store.Profile
	Cfg     *serverconfig.Config

	// NewSteam builds a Steam client for a resolved key. Overridable in tests.
	NewSteam func(key string) steam.API

	// LastValidation caches the most recent validation report this session, so
	// the dashboard can show an at-a-glance status.
	LastValidation *domain.Report

	// Validated guards the one-shot auto-validate on profile open.
	Validated bool

	// UpdateLatest holds the newest released version when an update is available
	// (empty otherwise, and always empty on dev builds). Set once by a background
	// check at startup and shown as a hint in the top bar.
	UpdateLatest string
}

// OpenProfile loads a profile's config and rebuilds the service layer with the
// profile's resolved Steam API key, making it the active session target.
func (s *Session) OpenProfile(p store.Profile) error {
	cfg, err := serverconfig.Load(p.IniPath)
	if err != nil {
		return err
	}
	key, _ := s.Store.APIKey(p.ID)
	s.Svc = service.New(s.newSteam(key), s.Store)
	pp := p
	s.Profile = &pp
	s.Cfg = cfg
	s.Validated = false
	return nil
}

func (s *Session) newSteam(key string) steam.API {
	if s.NewSteam != nil {
		return s.NewSteam(key)
	}
	return steam.New(key)
}

// Build returns the active profile's build (Unknown when no profile is open).
func (s *Session) Build() build.Build {
	if s.Profile == nil {
		return build.Unknown
	}
	return build.Parse(s.Profile.Build)
}

// Dirty reports whether the open config has unsaved changes.
func (s *Session) Dirty() bool {
	return s.Cfg != nil && s.Cfg.HasUnsavedChanges()
}

// BodyHeight is the height available to a screen's body, excluding the top bar
// (1) and footer (toast + status bar = 2).
func (s *Session) BodyHeight() int {
	h := s.Height - 3
	if h < 1 {
		return 1
	}
	return h
}

// ContentWidth is the width available to a screen's content, accounting for the
// 1-column gutter that pad() adds on each side.
func (s *Session) ContentWidth() int {
	w := s.Width - 2
	if w < 12 {
		return 12
	}
	return w
}

// Screen is one view in the navigation stack.
type Screen interface {
	Init(s *Session) tea.Cmd
	Update(s *Session, msg tea.Msg) (Screen, tea.Cmd)
	View(s *Session) string
	Title() string
}

// --- Navigation & status messages (handled centrally by the root model) ---

// PushMsg pushes a new screen onto the stack.
type PushMsg struct{ Screen Screen }

// PopMsg pops the current screen.
type PopMsg struct{}

// ReplaceMsg replaces the current screen.
type ReplaceMsg struct{ Screen Screen }

// ToastMsg shows a transient status line.
type ToastMsg struct{ Text string }

// ErrMsg surfaces an error as a toast.
type ErrMsg struct{ Err error }

// ConfirmMsg opens a yes/no modal; OnConfirm runs when the user confirms.
type ConfirmMsg struct {
	Prompt    string
	OnConfirm tea.Cmd
}

// Push returns a command that pushes screen.
func Push(screen Screen) tea.Cmd { return func() tea.Msg { return PushMsg{Screen: screen} } }

// Pop returns a command that pops the current screen.
func Pop() tea.Cmd { return func() tea.Msg { return PopMsg{} } }

// Replace returns a command that replaces the current screen.
func Replace(screen Screen) tea.Cmd { return func() tea.Msg { return ReplaceMsg{Screen: screen} } }

// Toast returns a command that shows text.
func Toast(text string) tea.Cmd { return func() tea.Msg { return ToastMsg{Text: text} } }

// Fail returns a command that surfaces err (nil-safe).
func Fail(err error) tea.Cmd {
	return func() tea.Msg {
		if err == nil {
			return nil
		}
		return ErrMsg{Err: err}
	}
}

// Confirm returns a command that opens a confirmation modal.
func Confirm(prompt string, onConfirm tea.Cmd) tea.Cmd {
	return func() tea.Msg { return ConfirmMsg{Prompt: prompt, OnConfirm: onConfirm} }
}

// Do runs fn with the session context off the UI thread and delivers its msg.
func (s *Session) Do(fn func(ctx context.Context) tea.Msg) tea.Cmd {
	ctx := s.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg { return fn(ctx) }
}
