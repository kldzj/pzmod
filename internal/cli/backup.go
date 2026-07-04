package cli

import (
	"github.com/dustin/go-humanize"
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

func newBackupCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage config backups",
	}
	cmd.AddCommand(newBackupListCmd(st), newBackupSnapshotCmd(st), newBackupRestoreCmd(st))
	return cmd
}

func newBackupListCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backups for the target config",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := resolveTarget(cmd, st)
			if err != nil {
				return err
			}
			entries, err := st.Backups(t.profileID())
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				cmd.Println(styleMuted.Render("no backups yet"))
				return nil
			}
			for _, e := range entries {
				note := e.Note
				if note != "" {
					note = "  " + styleMuted.Render(note)
				}
				cmd.Printf("%s  %s  %s%s\n", e.ID, styleMuted.Render("["+e.Kind+"]"), humanize.Bytes(uint64(e.Size)), note)
			}
			return nil
		},
	}
	addTargetFlags(cmd)
	return cmd
}

func newBackupSnapshotCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Take a manual backup of the target config",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := resolveTarget(cmd, st)
			if err != nil {
				return err
			}
			note, _ := cmd.Flags().GetString("note")
			entry, err := t.services(st).SnapshotProfile(t.profile, note, "manual")
			if err != nil {
				return err
			}
			cmd.Println(styleOK.Render("snapshot created"), entry.ID)
			return nil
		},
	}
	cmd.Flags().String("note", "", "optional note")
	addTargetFlags(cmd)
	return cmd
}

func newBackupRestoreCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <backup-id>",
		Short: "Restore a backup (a safety snapshot is taken first)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := resolveTarget(cmd, st)
			if err != nil {
				return err
			}
			if err := st.Restore(t.profileID(), args[0], t.iniPath()); err != nil {
				return err
			}
			cmd.Println(styleOK.Render("restored"), args[0])
			return nil
		},
	}
	addTargetFlags(cmd)
	return cmd
}
