package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/kldzj/pzmod/internal/pathutil"
	"github.com/kldzj/pzmod/internal/store"
)

// profileform creates a new profile via a huh form (with a file browser for the
// config path).
type profileform struct {
	form *huh.Form
	done bool // guards against re-adding when the form stays Completed

	name, file, build, workshop string
}

// NewProfileForm returns the add-profile screen.
func NewProfileForm() Screen { return &profileform{} }

func (pf *profileform) Title() string { return "New profile" }

func (pf *profileform) Init(s *Session) tea.Cmd {
	start, _ := os.UserHomeDir()
	if start == "" {
		start, _ = os.Getwd()
	}

	pf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Name").Placeholder("My Server").
				Value(&pf.name).Validate(required("name")),
			huh.NewFilePicker().Title("Config file (servertest.ini)").
				CurrentDirectory(start).
				ShowHidden(true).
				FileAllowed(true).DirAllowed(false).
				AllowedTypes([]string{".ini"}).
				Height(10).
				Value(&pf.file).
				Validate(fileExists),
			huh.NewSelect[string]().Title("Game build").
				Options(
					huh.NewOption("Build 41", "b41"),
					huh.NewOption("Build 42", "b42"),
					huh.NewOption("Unspecified", ""),
				).Value(&pf.build),
			huh.NewInput().Title("Workshop content path (optional)").
				Value(&pf.workshop).Validate(dirOrEmpty),
		),
	).WithWidth(min(72, max(24, s.Width-4))).WithHeight(min(20, max(10, s.BodyHeight()))).WithShowHelp(true)

	return pf.form.Init()
}

func (pf *profileform) Update(s *Session, msg tea.Msg) (Screen, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" && !pf.formActive() {
		return pf, Pop()
	}
	if pf.form == nil || pf.done {
		return pf, nil
	}

	fm, cmd := pf.form.Update(msg)
	if f, ok := fm.(*huh.Form); ok {
		pf.form = f
	}

	switch pf.form.State {
	case huh.StateCompleted:
		pf.done = true
		return pf, pf.create(s)
	case huh.StateAborted:
		pf.done = true
		return pf, Pop()
	}
	return pf, cmd
}

// formActive reports whether the form is still collecting input (so esc is the
// form's to handle, e.g. to step out of the file picker).
func (pf *profileform) formActive() bool {
	return pf.form != nil && pf.form.State == huh.StateNormal
}

func (pf *profileform) create(s *Session) tea.Cmd {
	file := pathutil.Expand(pf.file)
	if !pathutil.FileExists(file) {
		return tea.Batch(Fail(fmt.Errorf("config file not found: %s", file)), Pop())
	}
	workshop := ""
	if strings.TrimSpace(pf.workshop) != "" {
		workshop = pathutil.Expand(pf.workshop)
	}
	if _, err := s.Store.AddProfile(store.Profile{
		Name:                pf.name,
		IniPath:             file,
		Build:               pf.build,
		WorkshopContentPath: workshop,
	}); err != nil {
		return tea.Batch(Fail(err), Pop())
	}
	return tea.Batch(Toast("profile added"), Pop(), func() tea.Msg { return profilesChangedMsg{} })
}

func (pf *profileform) View(s *Session) string {
	if pf.form == nil {
		return ""
	}
	return pad(pf.form.View())
}

func required(label string) func(string) error {
	return func(v string) error {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%s is required", label)
		}
		return nil
	}
}

func fileExists(v string) error {
	if !pathutil.FileExists(pathutil.Expand(v)) {
		return errors.New("file does not exist")
	}
	return nil
}

func dirOrEmpty(v string) error {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	if !pathutil.DirExists(pathutil.Expand(v)) {
		return errors.New("directory does not exist")
	}
	return nil
}
