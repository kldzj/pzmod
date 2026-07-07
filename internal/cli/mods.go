package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/kldzj/pzmod/pkg/build"
	"github.com/kldzj/pzmod/pkg/domain"
	"github.com/kldzj/pzmod/pkg/service"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

func newModsCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mods",
		Short: "List, add, and remove mods",
	}
	cmd.AddCommand(newModsListCmd(st), newModsAddCmd(st), newModsRemoveCmd(st), newModsShowCmd(st))
	return cmd
}

func newModsListCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List enabled mods and workshop items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := resolveTarget(cmd, st)
			if err != nil {
				return err
			}
			cfg, err := t.config()
			if err != nil {
				return err
			}
			sm := cfg.ServerMods()
			if jsonEnabled(cmd) {
				return emitJSON(cmd, modsListJSON{
					Mods:          orEmpty(sm.Mods),
					WorkshopItems: orEmpty(sm.WorkshopItems),
					Maps:          orEmpty(sm.Maps),
				})
			}
			cmd.Printf("%s (%d)\n", styleInfo.Render("Mods"), len(sm.Mods))
			for i, m := range sm.Mods {
				cmd.Printf("  %2d. %s\n", i+1, m)
			}
			cmd.Printf("%s (%d)\n", styleInfo.Render("WorkshopItems"), len(sm.WorkshopItems))
			for _, id := range sm.WorkshopItems {
				cmd.Printf("  %s\n", id)
			}
			if len(sm.Maps) > 0 {
				cmd.Printf("%s: %s\n", styleInfo.Render("Map"), strings.Join(sm.Maps, "; "))
			}
			return nil
		},
	}
	addTargetFlags(cmd)
	return cmd
}

func newModsAddCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <workshop-id...>",
		Short: "Add workshop items (and optionally resolve dependencies)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := resolveTarget(cmd, st)
			if err != nil {
				return err
			}
			svc, err := t.servicesWithSteam(st)
			if err != nil {
				return err
			}
			cfg, err := t.config()
			if err != nil {
				return err
			}
			sm := cfg.ServerMods()

			asJSON := jsonEnabled(cmd)
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			resolveDeps, _ := cmd.Flags().GetBool("resolve-deps")
			var projected domain.ServerMods
			var result any // JSON payload for the chosen path
			if resolveDeps {
				plan, err := svc.Resolve(cmd.Context(), args, sm)
				if err != nil {
					return err
				}
				projected = plan.Apply(sm, t.build() == build.B42)
				if asJSON {
					rj := newResolveJSON(plan)
					rj.DryRun = dryRun
					result = rj
				} else {
					printPlan(cmd, plan)
				}
			} else {
				updated, missing, added, err := shallowAdd(cmd.Context(), svc, sm, args, t.build() == build.B42)
				if err != nil {
					return err
				}
				projected = updated
				if asJSON {
					addedIDs := make([]string, len(added))
					for i, it := range added {
						addedIDs[i] = it.PublishedFileID
					}
					result = shallowAddJSON{Added: addedIDs, Missing: orEmpty(missing), DryRun: dryRun}
				} else {
					verb := "added"
					if dryRun {
						verb = "would add"
					}
					cmd.Printf("%s %d item(s)\n", verb, len(added))
					if len(missing) > 0 {
						cmd.Println(styleWarn.Render("could not fetch:"), strings.Join(missing, ", "))
					}
				}
			}

			if dryRun {
				if asJSON {
					return emitJSON(cmd, result)
				}
				cmd.Println(styleMuted.Render("dry run: nothing written"))
				return nil
			}

			cfg.ApplyServerMods(projected)
			if noBackup, _ := cmd.Flags().GetBool("no-backup"); !noBackup {
				if _, err := svc.SnapshotProfile(t.profile, "before mods add", "auto"); err != nil {
					return err
				}
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			if asJSON {
				return emitJSON(cmd, result)
			}
			return nil
		},
	}
	cmd.Flags().Bool("resolve-deps", false, "also add transitive dependencies")
	cmd.Flags().Bool("no-backup", false, "do not snapshot before saving")
	cmd.Flags().Bool("dry-run", false, "resolve and show the plan without writing")
	addTargetFlags(cmd)
	return cmd
}

func newModsRemoveCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <id...>",
		Short: "Remove mod IDs and/or workshop items",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := resolveTarget(cmd, st)
			if err != nil {
				return err
			}
			cfg, err := t.config()
			if err != nil {
				return err
			}
			before := cfg.ServerMods().Clone()
			after := before
			for _, id := range args {
				after = after.RemoveItem(id).RemoveMod(id).RemoveMap(id)
			}

			if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
				prev := removePreviewJSON{
					RemovedMods:  orEmpty(removedFrom(before.Mods, after.Mods)),
					RemovedItems: orEmpty(removedFrom(before.WorkshopItems, after.WorkshopItems)),
					RemovedMaps:  orEmpty(removedFrom(before.Maps, after.Maps)),
					DryRun:       true,
				}
				if jsonEnabled(cmd) {
					return emitJSON(cmd, prev)
				}
				cmd.Printf("would remove %d mod(s), %d item(s), %d map(s) (dry run, nothing written)\n",
					len(prev.RemovedMods), len(prev.RemovedItems), len(prev.RemovedMaps))
				return nil
			}

			cfg.ApplyServerMods(after)
			if noBackup, _ := cmd.Flags().GetBool("no-backup"); !noBackup {
				if _, err := t.services(st).SnapshotProfile(t.profile, "before mods remove", "auto"); err != nil {
					return err
				}
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			if jsonEnabled(cmd) {
				return emitJSON(cmd, map[string][]string{"removed": args})
			}
			return nil
		},
	}
	cmd.Flags().Bool("no-backup", false, "do not snapshot before saving")
	cmd.Flags().Bool("dry-run", false, "show what would be removed without writing")
	cmd.ValidArgsFunction = completeInstalledIDs(st)
	addTargetFlags(cmd)
	return cmd
}

func shallowAdd(ctx context.Context, svc *service.Services, sm domain.ServerMods, ids []string, explicit bool) (domain.ServerMods, []string, []steam.WorkshopItem, error) {
	items, missing, err := svc.Details(ctx, ids)
	if err != nil {
		return sm, nil, nil, err
	}
	var content []steam.WorkshopItem
	var memberIDs []string
	for _, it := range items {
		if it.IsCollection() {
			memberIDs = append(memberIDs, it.GetChildIDs()...)
		} else {
			content = append(content, it)
		}
	}
	if len(memberIDs) > 0 {
		members, miss, err := svc.Details(ctx, memberIDs)
		if err != nil {
			return sm, nil, nil, err
		}
		missing = append(missing, miss...)
		for _, m := range members {
			if !m.IsCollection() {
				content = append(content, m)
			}
		}
	}
	for _, it := range content {
		sm = sm.AddItem(it.PublishedFileID)
		parsed := it.Parse()
		for _, mod := range parsed.Mods {
			sm = sm.AddMod(domain.FormatModRef(it.PublishedFileID, mod, explicit))
		}
		for _, mp := range parsed.Maps {
			sm = sm.AddMap(mp)
		}
	}
	return sm, missing, content, nil
}

func printPlan(cmd *cobra.Command, plan service.ResolvePlan) {
	cmd.Printf("resolved: +%d items, +%d mods, +%d maps\n",
		len(plan.AddWorkshopItems), len(plan.AddMods), len(plan.AddMaps))
	if len(plan.Missing) > 0 {
		cmd.Println(styleWarn.Render("missing:"), strings.Join(plan.Missing, ", "))
	}
	for _, mm := range plan.MultiMod {
		cmd.Printf("%s item %s declares multiple mods: %s\n",
			styleWarn.Render("note:"), mm.ItemID, strings.Join(mm.ModIDs, ", "))
	}
	if len(plan.Cycles) > 0 {
		cmd.Println(styleWarn.Render(fmt.Sprintf("%d dependency cycle(s) detected", len(plan.Cycles))))
	}
}

func newModsShowCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <id...>",
		Short: "Show resolved Workshop details for one or more items",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, _ := cmd.Flags().GetString("profile")
			if !st.HasAPIKey(profile) {
				return errNoKey
			}
			key, _ := st.APIKey(profile)
			svc := service.New(steamFactory(key), st)

			items, missing, err := svc.Details(cmd.Context(), args)
			if err != nil {
				return err
			}
			if jsonEnabled(cmd) {
				out := modShowResultJSON{
					Items:   make([]modShowJSON, 0, len(items)),
					Missing: orEmpty(missing),
				}
				for i := range items {
					out.Items = append(out.Items, newModShowJSON(&items[i]))
				}
				return emitJSON(cmd, out)
			}
			for i := range items {
				printModDetails(cmd, &items[i])
			}
			if len(missing) > 0 {
				cmd.Println(styleWarn.Render("could not fetch:"), strings.Join(missing, ", "))
			}
			return nil
		},
	}
	cmd.ValidArgsFunction = completeInstalledIDs(st)
	cmd.Flags().StringP("profile", "p", "", "use a profile's API key")
	return cmd
}

func printModDetails(cmd *cobra.Command, it *steam.WorkshopItem) {
	parsed := it.Parse()
	kind := "mod"
	if it.IsCollection() {
		kind = "collection"
	}
	cmd.Printf("%s  %s\n", styleInfo.Render(it.PublishedFileID), it.Title)
	cmd.Printf("  type: %s   size: %s\n", kind, humanize.Bytes(uint64(it.FileSize)))
	if len(parsed.Mods) > 0 {
		cmd.Printf("  mod ids: %s\n", strings.Join(parsed.Mods, ", "))
	}
	if len(parsed.Maps) > 0 {
		cmd.Printf("  maps: %s\n", strings.Join(parsed.Maps, ", "))
	}
	if it.IsCollection() && len(it.Children) > 0 {
		cmd.Printf("  items: %s\n", strings.Join(it.GetChildIDs(), ", "))
	}
	if it.ShortDesc != "" {
		cmd.Printf("  %s\n", styleMuted.Render(it.ShortDesc))
	}
	cmd.Println()
}

// removedFrom returns the entries in before that are absent from after.
func removedFrom(before, after []string) []string {
	inAfter := make(map[string]bool, len(after))
	for _, x := range after {
		inAfter[x] = true
	}
	var out []string
	for _, x := range before {
		if !inAfter[x] {
			out = append(out, x)
		}
	}
	return out
}
