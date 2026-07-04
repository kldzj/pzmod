package cli

import (
	"context"

	"github.com/kldzj/pzmod/pkg/service"
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/kldzj/pzmod/internal/tui"
	"github.com/spf13/cobra"
)

// launchTUI starts the interactive terminal app. With --file or --profile it
// opens that config directly; otherwise it shows the profile launcher.
func launchTUI(st *store.Store, cmd *cobra.Command) error {
	file, _ := cmd.Flags().GetString("file")
	profile, _ := cmd.Flags().GetString("profile")
	mouse, _ := cmd.Flags().GetBool("mouse")

	// The TUI manages ctrl+c itself (so the unsaved-changes guard always runs),
	// so it does not use the command's interrupt-cancelled context; in-flight
	// Steam work uses a plain background context and is bounded by quitting.
	tuiCtx := context.Background()

	if file != "" || profile != "" {
		t, err := resolveTarget(cmd, st)
		if err != nil {
			return err
		}
		svc := t.services(st)
		return tui.RunOpen(svc, st, tuiCtx, t.profile, mouse)
	}

	key, _ := st.APIKey("")
	svc := service.New(steamFactory(key), st)
	return tui.Run(svc, st, tuiCtx, tui.NewLauncher(), mouse)
}
