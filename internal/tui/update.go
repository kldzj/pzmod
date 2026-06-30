package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/kldzj/pzmod/internal/version"
)

// updateAvailableMsg reports that a newer release than the running build exists.
type updateAvailableMsg struct{ latest string }

// checkUpdateCmd looks up the latest GitHub release in the background so startup
// is never blocked on a network call. It is a no-op on dev builds (no embedded
// version), so `go test` and unreleased binaries never reach out to GitHub.
func checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		if !version.IsSet() {
			return nil
		}
		updater, err := version.NewUpdater()
		if err != nil {
			return nil
		}
		latest, err := version.GetLatestRelease(updater)
		if err != nil {
			return nil
		}
		if version.IsLatest(version.Get(), latest) {
			return nil
		}
		return updateAvailableMsg{latest: latest.Version()}
	}
}
