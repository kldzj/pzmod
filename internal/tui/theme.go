package tui

import "github.com/charmbracelet/lipgloss"

// Palette is the adaptive color set; each color adapts to a light or dark
// terminal background so the UI looks intentional everywhere.
var (
	colAccent   = lipgloss.AdaptiveColor{Light: "#5A4FCF", Dark: "#9A8CFF"}
	colOnAccent = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#0B0B12"}
	colText     = lipgloss.AdaptiveColor{Light: "#1F2328", Dark: "#E6E6E6"}
	colMuted    = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8B8B96"}
	colFaint    = lipgloss.AdaptiveColor{Light: "#9AA0A6", Dark: "#5B5B66"}
	colOK       = lipgloss.AdaptiveColor{Light: "#117A37", Dark: "#3FB950"}
	colWarn     = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E3B341"}
	colError    = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F85149"}
	colBarBG    = lipgloss.AdaptiveColor{Light: "#EBECF0", Dark: "#1B1B22"}
	colSelBG    = lipgloss.AdaptiveColor{Light: "#E3E0FB", Dark: "#2C2740"}
	colChipBG   = lipgloss.AdaptiveColor{Light: "#E7E9EE", Dark: "#26262F"}
)

// Theme holds the lipgloss styles used across the TUI.
type Theme struct {
	Accent lipgloss.TerminalColor

	// Content styles.
	Title    lipgloss.Style // accent heading / cursor marker
	Subtitle lipgloss.Style
	Item     lipgloss.Style
	Muted    lipgloss.Style
	Faint    lipgloss.Style
	Error    lipgloss.Style
	Warn     lipgloss.Style
	OK       lipgloss.Style
	Badge    lipgloss.Style
	Chip     lipgloss.Style
	Box      lipgloss.Style

	// Row selection (full width applied at render time).
	SelectedItem lipgloss.Style // inline highlight
	SelectedRow  lipgloss.Style // full-width row highlight

	// Chrome.
	Brand     lipgloss.Style // app name in the top bar
	Crumb     lipgloss.Style // breadcrumb / screen title in the top bar
	TopBar    lipgloss.Style // top bar fill
	BottomBar lipgloss.Style // status bar fill
	ToastOK   lipgloss.Style
	ToastErr  lipgloss.Style
}

// DefaultTheme returns the standard pzmod theme.
func DefaultTheme() Theme {
	return Theme{
		Accent:       colAccent,
		Title:        lipgloss.NewStyle().Bold(true).Foreground(colAccent),
		Subtitle:     lipgloss.NewStyle().Bold(true).Foreground(colText),
		Item:         lipgloss.NewStyle().Foreground(colText),
		Muted:        lipgloss.NewStyle().Foreground(colMuted),
		Faint:        lipgloss.NewStyle().Foreground(colFaint),
		Error:        lipgloss.NewStyle().Bold(true).Foreground(colError),
		Warn:         lipgloss.NewStyle().Foreground(colWarn),
		OK:           lipgloss.NewStyle().Foreground(colOK),
		Badge:        lipgloss.NewStyle().Bold(true).Foreground(colOnAccent).Background(colAccent).Padding(0, 1),
		Chip:         lipgloss.NewStyle().Foreground(colText).Background(colChipBG).Padding(0, 1),
		Box:          lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colAccent).Padding(1, 2),
		SelectedItem: lipgloss.NewStyle().Bold(true).Foreground(colOnAccent).Background(colAccent),
		SelectedRow:  lipgloss.NewStyle().Bold(true).Foreground(colText).Background(colSelBG),
		Brand:        lipgloss.NewStyle().Bold(true).Foreground(colOnAccent).Background(colAccent).Padding(0, 1),
		Crumb:        lipgloss.NewStyle().Foreground(colMuted),
		TopBar:       lipgloss.NewStyle().Background(colBarBG),
		BottomBar:    lipgloss.NewStyle().Foreground(colMuted).Background(colBarBG),
		ToastOK:      lipgloss.NewStyle().Bold(true).Foreground(colOnAccent).Background(colOK).Padding(0, 1),
		ToastErr:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(colError).Padding(0, 1),
	}
}
