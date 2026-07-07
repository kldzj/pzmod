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
			old := cfg.GetOr(key, "")
			value := args[1]
			if tf, ok := setTransforms[args[0]]; ok {
				value = tf(value)
			}

			if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
				if jsonEnabled(cmd) {
					return emitJSON(cmd, setPreviewJSON{Key: args[0], Old: old, New: value, DryRun: true})
				}
				cmd.Printf("%s: %q -> %q (dry run, nothing written)\n", args[0], old, value)
				return nil
			}

			cfg.Set(key, value)

			if noSave, _ := cmd.Flags().GetBool("no-save"); noSave {
				if jsonEnabled(cmd) {
					return emitJSON(cmd, map[string]string{"config": cfg.String()})
				}
				cmd.Print(cfg.String())
				return nil
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			if jsonEnabled(cmd) {
				return emitJSON(cmd, map[string]any{"key": args[0], "value": value, "saved": true})
			}
			return nil
		},
	}
	cmd.Flags().BoolP("no-save", "n", false, "print the result instead of writing the file")
	cmd.Flags().Bool("dry-run", false, "show the change without writing")
	cmd.ValidArgsFunction = completeConfigKeys
	addTargetFlags(cmd)
	return cmd
}
