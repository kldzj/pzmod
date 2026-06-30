package cli

import (
	"errors"

	"github.com/kldzj/pzmod/internal/version"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update pzmod to the latest release",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !version.IsSet() {
				return errors.New("this build has no embedded version; install a released binary to self-update")
			}
			updater, err := version.NewUpdater()
			if err != nil {
				return err
			}
			ver := version.Get()
			latest, err := version.GetLatestRelease(updater)
			if err != nil {
				return err
			}
			if version.IsLatest(ver, latest) {
				cmd.Println("pzmod is already up to date")
				return nil
			}
			if check, _ := cmd.Flags().GetBool("check"); check {
				cmd.Println("A new version is available:", latest.Version())
				return nil
			}
			return version.Update(ver, latest, updater)
		},
	}
	cmd.Flags().BoolP("check", "c", false, "only check for updates")
	return cmd
}
