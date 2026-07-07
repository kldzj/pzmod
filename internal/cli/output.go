package cli

import (
	"encoding/json"

	"github.com/charmbracelet/lipgloss"
	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/spf13/cobra"
)

// jsonEnabled reports whether the global --json flag is set. It is a persistent
// flag on the root command, so every subcommand inherits it via cmd.Flags().
func jsonEnabled(cmd *cobra.Command) bool {
	b, _ := cmd.Flags().GetBool("json")
	return b
}

// WantsJSON reports whether --json was requested, read from the root command's
// persistent flags. main.go uses this to format a failing command's error as a
// JSON envelope on stderr.
func WantsJSON(root *cobra.Command) bool {
	b, _ := root.PersistentFlags().GetBool("json")
	return b
}

// emitJSON writes v as indented JSON to the command's stdout.
func emitJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// orEmpty returns s, or an empty (non-nil) slice when s is nil, so it marshals
// as [] rather than null for machine consumers.
func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

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
