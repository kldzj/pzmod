package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

func newCopyCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy <dest>",
		Short: "Copy the server config to another path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := resolveTarget(cmd, st)
			if err != nil {
				return err
			}
			cfg, err := t.config()
			if err != nil {
				return err
			}

			dest, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			if _, err := os.Stat(dest); err == nil {
				if force, _ := cmd.Flags().GetBool("force"); !force {
					return fmt.Errorf("%s already exists (use --force to overwrite)", dest)
				}
			}
			if err := cfg.SaveTo(dest); err != nil {
				return err
			}
			if jsonEnabled(cmd) {
				return emitJSON(cmd, map[string]string{"copied": dest})
			}
			return nil
		},
	}
	cmd.Flags().BoolP("force", "F", false, "overwrite an existing destination")
	addTargetFlags(cmd)
	return cmd
}
