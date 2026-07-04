package cli

import (
	"fmt"
	"strings"

	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

// setTransforms applies value normalization for certain aliases, matching v2.
var setTransforms = map[string]func(string) string{
	"desc":   func(v string) string { return strings.ReplaceAll(v, "\n", "<LINE>") },
	"public": func(v string) string { return boolString(v == "true" || v == "1") },
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func newSetCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a server config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := resolveTarget(cmd, st)
			if err != nil {
				return err
			}
			cfg, err := t.config()
			if err != nil {
				return err
			}

			key, isAlias := resolveKey(args[0])
			if !isAlias && !cfg.Document().Has(key) {
				return fmt.Errorf("unknown key %q (try `get list`)", args[0])
			}
			value := args[1]
			if tf, ok := setTransforms[args[0]]; ok {
				value = tf(value)
			}
			cfg.Set(key, value)

			if noSave, _ := cmd.Flags().GetBool("no-save"); noSave {
				cmd.Print(cfg.String())
				return nil
			}
			return cfg.Save()
		},
	}
	cmd.Flags().BoolP("no-save", "n", false, "print the result instead of writing the file")
	addTargetFlags(cmd)
	return cmd
}
