package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kldzj/pzmod/pkg/serverconfig"
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

// aliasMap maps friendly CLI names to servertest.ini keys (v2-compatible).
var aliasMap = map[string]string{
	"name":     serverconfig.KeyName,
	"desc":     serverconfig.KeyDescription,
	"public":   serverconfig.KeyPublic,
	"password": serverconfig.KeyPassword,
	"slots":    serverconfig.KeyMaxPlayers,
}

func resolveKey(name string) (string, bool) {
	if k, ok := aliasMap[name]; ok {
		return k, true
	}
	return name, false
}

func newGetCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Print a server config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "list" {
				if jsonEnabled(cmd) {
					return emitJSON(cmd, map[string][]string{"keys": sortedAliases()})
				}
				cmd.Println("Available keys:", strings.Join(sortedAliases(), ", "))
				return nil
			}
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
			value := cfg.GetOr(key, "")
			if jsonEnabled(cmd) {
				return emitJSON(cmd, getJSON{Key: args[0], Value: value})
			}
			cmd.Println(value)
			return nil
		},
	}
	cmd.ValidArgsFunction = completeConfigKeys
	addTargetFlags(cmd)
	return cmd
}

func sortedAliases() []string {
	keys := make([]string, 0, len(aliasMap))
	for k := range aliasMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
