// Package cli builds pzmod's cobra command tree. Commands are thin wrappers
// over the service layer; this and main.go are the only places that print to
// the user or exit the process.
package cli

import (
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

// NewRootCommand assembles the full command tree. ver is the embedded build
// version (empty in dev builds).
func NewRootCommand(st *store.Store, ver string) *cobra.Command {
	displayVer := ver
	if displayVer == "" {
		displayVer = "dev"
	}

	root := &cobra.Command{
		Use:     "pzmod",
		Short:   "pzmod manages Project Zomboid server mods",
		Version: displayVer,
		Args:    cobra.NoArgs,
		Example: "  pzmod                       # launch the interactive terminal app\n" +
			"  pzmod --file server.ini     # open a specific config\n" +
			"  pzmod validate              # validate the default profile\n" +
			"  pzmod search hydrocraft     # search the Workshop",
		SilenceUsage:  true,
		SilenceErrors: true, // main.go is the sole error reporter
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchTUI(st, cmd)
		},
	}

	addTargetFlags(root)
	root.Flags().Bool("mouse", false, "enable mouse support in the terminal app (wheel scroll; may affect text selection)")
	root.AddCommand(
		newGetCmd(st),
		newSetCmd(st),
		newCopyCmd(st),
		newAPIKeyCmd(st),
		newUpdateCmd(),
		newProfileCmd(st),
		newValidateCmd(st),
		newSearchCmd(st),
		newBackupCmd(st),
		newModsCmd(st),
	)
	return root
}
