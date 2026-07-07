package cli

import (
	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

// completeProfiles suggests stored profile IDs. Read-only; degrades to no
// suggestions on any error.
func completeProfiles(st *store.Store) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		profs, err := st.Profiles()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ids := make([]string, 0, len(profs))
		for _, p := range profs {
			ids = append(ids, p.ID)
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeInstalledIDs suggests the workshop IDs, logical mod IDs, and map names
// from the resolved target config. No network. Degrades to nothing on error.
func completeInstalledIDs(st *store.Store) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		t, err := resolveTarget(cmd, st)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cfg, err := t.config()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		sm := cfg.ServerMods()
		out := make([]string, 0, len(sm.WorkshopItems)+len(sm.Mods)+len(sm.Maps))
		out = append(out, sm.WorkshopItems...)
		for _, m := range sm.Mods {
			out = append(out, domain.ParseModRef(m).ID)
		}
		out = append(out, sm.Maps...)
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeConfigKeys suggests the known config alias keys, for the key argument
// of get/set (only at the first positional position).
func completeConfigKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return sortedAliases(), cobra.ShellCompDirectiveNoFileComp
}

// registerFlagCompletions walks the command tree and registers profile-ID
// completion for every command that has a --profile flag.
func registerFlagCompletions(root *cobra.Command, st *store.Store) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Flags().Lookup("profile") != nil {
			_ = c.RegisterFlagCompletionFunc("profile", completeProfiles(st))
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}
