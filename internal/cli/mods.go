package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/kldzj/pzmod/internal/build"
	"github.com/kldzj/pzmod/internal/domain"
	"github.com/kldzj/pzmod/internal/service"
	"github.com/kldzj/pzmod/internal/steam"
	"github.com/kldzj/pzmod/internal/store"
	"github.com/spf13/cobra"
)

func newModsCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mods",
		Short: "List, add, and remove mods",
	}
	cmd.AddCommand(newModsListCmd(st), newModsAddCmd(st), newModsRemoveCmd(st))
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

			resolveDeps, _ := cmd.Flags().GetBool("resolve-deps")
			var projected domain.ServerMods
			if resolveDeps {
				plan, err := svc.Resolve(cmd.Context(), args, sm)
				if err != nil {
					return err
				}
				projected = plan.Apply(sm, t.build() == build.B42)
				printPlan(cmd, plan)
			} else {
				updated, missing, added, err := shallowAdd(cmd.Context(), svc, sm, args, t.build() == build.B42)
				if err != nil {
					return err
				}
				projected = updated
				cmd.Printf("added %d item(s)\n", len(added))
				if len(missing) > 0 {
					cmd.Println(styleWarn.Render("could not fetch:"), strings.Join(missing, ", "))
				}
			}

			cfg.ApplyServerMods(projected)
			if noBackup, _ := cmd.Flags().GetBool("no-backup"); !noBackup {
				if _, err := svc.SnapshotProfile(t.profile, "before mods add", "auto"); err != nil {
					return err
				}
			}
			return cfg.Save()
		},
	}
	cmd.Flags().Bool("resolve-deps", false, "also add transitive dependencies")
	cmd.Flags().Bool("no-backup", false, "do not snapshot before saving")
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
			sm := cfg.ServerMods()
			for _, id := range args {
				sm = sm.RemoveItem(id).RemoveMod(id).RemoveMap(id)
			}
			cfg.ApplyServerMods(sm)
			if noBackup, _ := cmd.Flags().GetBool("no-backup"); !noBackup {
				if _, err := t.services(st).SnapshotProfile(t.profile, "before mods remove", "auto"); err != nil {
					return err
				}
			}
			return cfg.Save()
		},
	}
	cmd.Flags().Bool("no-backup", false, "do not snapshot before saving")
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
