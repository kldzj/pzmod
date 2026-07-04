package cli

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/kldzj/pzmod/pkg/domain"
)

// Severity styles shared by CLI output. lipgloss degrades gracefully on
// non-color terminals and honors NO_COLOR.
var (
	styleError = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	styleWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	styleOK    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func severityTag(s domain.Severity) string {
	switch s {
	case domain.SeverityError:
		return styleError.Render("ERROR")
	case domain.SeverityWarning:
		return styleWarn.Render("WARN")
	default:
		return styleInfo.Render("INFO")
	}
}
