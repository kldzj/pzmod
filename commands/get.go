package commands

import (
	"github.com/kldzj/pzmod/config"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

func cmdGet() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "get <name>",
		Short: "Get server config values",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			config := config.UnsafeLoadConfig(cmd)

			if args[0] == "list" {
				listAvailableConfigKeys(cmd)
				return
			}

			var key string
			if mapped, ok := configKeyMap[args[0]]; ok {
				key = mapped
			} else {
				cobra.CheckErr(util.ErrInvalidKey)
			}

			value := config.GetOrDefault(key, "")
			cmd.Println(value)
		},
	}

	SetFileFlag(cmd)

	return cmd
}
