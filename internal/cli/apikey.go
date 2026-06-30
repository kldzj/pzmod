package cli

import (
	"errors"

	"github.com/kldzj/pzmod/internal/store"
	"github.com/spf13/cobra"
)

func newAPIKeyCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api-key <key>",
		Short: "Set or clear the Steam Web API key",
		Long: "Stores the Steam Web API key in the pzmod config dir. With --profile, sets a\n" +
			"per-profile override instead of the global key. Get a key at\n" +
			"https://steamcommunity.com/dev/apikey",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, _ := cmd.Flags().GetString("profile")

			if clear, _ := cmd.Flags().GetBool("clear"); clear {
				return st.ClearKey(profile)
			}
			if len(args) == 0 || len(args[0]) != 32 {
				return errors.New("a 32-character Steam Web API key is required")
			}
			if profile != "" {
				if err := st.SetProfileKey(profile, args[0]); err != nil {
					return err
				}
			} else if err := st.SetGlobalKey(args[0]); err != nil {
				return err
			}
			cmd.Println(styleOK.Render("API key saved"))
			return nil
		},
	}
	cmd.Flags().BoolP("clear", "c", false, "clear the key instead of setting it")
	cmd.Flags().StringP("profile", "p", "", "set a per-profile key override")
	return cmd
}
