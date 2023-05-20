package commands

import (
	"os"
	"path/filepath"

	"github.com/kldzj/pzmod/config"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func cmdCopyConfig() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "copy <path>",
		Short: "Copy the server config to another path",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			config := config.UnsafeLoadConfig(cmd)
			saveTo := filepath.Clean(args[0])
			if !filepath.IsAbs(saveTo) {
				cwd, err := os.Getwd()
				cobra.CheckErr(err)
				saveTo = filepath.Join(cwd, saveTo)
			}

			if util.FileExists(saveTo) {
				force, _ := cmd.Flags().GetBool("force")
				if !force {
					cobra.CheckErr(util.ErrFileExists)
				}
			}

			config.SaveTo(saveTo)
		},
	}

	cmd.Flags().BoolP("force", "F", false, "overwrite existing file")
	SetFileFlag(cmd)

	return cmd
}
