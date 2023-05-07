package commands

import (
	"fmt"

	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func cmdSetApiKey() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "api-key <key>",
		Short: "Set the API key",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			clear, _ := cmd.Flags().GetBool("clear")
			if clear {
				err := util.DeleteCredentials()
				cobra.CheckErr(err)
				return
			}

			if len(args) == 0 || len(args[0]) != 32 {
				cobra.CheckErr(util.ErrInvalidKey)
			}

			err := util.StoreCredentials(args[0])
			cobra.CheckErr(err)

			fmt.Println("API key set successfully")
		},
	}

	cmd.Flags().BoolP("clear", "c", false, "clear the API key")

	return cmd
}
