package commands

import (
	"fmt"
	"strings"

	"github.com/kldzj/pzmod/config"
	"github.com/kldzj/pzmod/util"
	"github.com/spf13/cobra"
)

var configKeyMap = map[string]string{
	"name":     util.CfgKeyName,
	"desc":     util.CfgKeyDesc,
	"public":   util.CfgKeyPub,
	"password": util.CfgKeyPass,
	"slots":    util.CfgKeyMax,
}

var setTransformMap = map[string]func(string) (string, error){
	"desc": func(v string) (string, error) {
		return strings.ReplaceAll(v, "\n", "<line>"), nil
	},
	"public": func(v string) (string, error) {
		return util.BoolString(v == "true" || v == "1"), nil
	},
}

func cmdSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <name> <value>",
		Short: "Set server config values",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			config := config.UnsafeLoadConfig(cmd)
			value := args[1]

			if value == "list" {
				listAvailableConfigKeys(cmd)
				return
			}

			var key string
			if mapped, ok := configKeyMap[args[0]]; ok {
				key = mapped
			} else {
				cobra.CheckErr(util.ErrInvalidKey)
			}

			if transform, ok := setTransformMap[args[0]]; ok {
				value, err := transform(value)
				cobra.CheckErr(err)
				args[1] = value
			}

			config.Set(key, value)

			noSave, _ := cmd.Flags().GetBool("no-save")
			if noSave {
				return
			}

			config.Save()
		},
	}

	cmd.Flags().BoolP("no-save", "n", false, "do not save to file")

	return cmd
}

func listAvailableConfigKeys(cmd *cobra.Command) {
	keys := make([]string, 0, len(configKeyMap))
	for k := range configKeyMap {
		keys = append(keys, k)
	}

	msg := "Available config keys: "
	msg += strings.Join(keys, ", ")
	fmt.Println(msg)
}
