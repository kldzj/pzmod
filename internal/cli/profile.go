package cli

import (
	"fmt"

	"github.com/kldzj/pzmod/internal/pathutil"
	"github.com/kldzj/pzmod/internal/store"
	"github.com/spf13/cobra"
)

func newProfileCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage server profiles",
	}
	cmd.AddCommand(
		newProfileListCmd(st),
		newProfileAddCmd(st),
		newProfileRemoveCmd(st),
		newProfileUseCmd(st),
		newProfileShowCmd(st),
	)
	return cmd
}

func newProfileListCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := st.Profiles()
			if err != nil {
				return err
			}
			if len(profiles) == 0 {
				cmd.Println(styleMuted.Render("no profiles yet - add one with `pzmod profile add`"))
				return nil
			}
			def, _ := st.DefaultProfile()
			for _, p := range profiles {
				marker := "  "
				if p.ID == def.ID {
					marker = styleOK.Render("* ")
				}
				cmd.Printf("%s%s  %s  %s\n", marker, p.ID, styleMuted.Render(pathutil.Abbreviate(p.IniPath)), buildBadge(p.Build))
			}
			return nil
		},
	}
}

func newProfileAddCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			file, _ := cmd.Flags().GetString("file")
			buildStr, _ := cmd.Flags().GetString("build")
			workshop, _ := cmd.Flags().GetString("workshop-path")
			if name == "" || file == "" {
				return fmt.Errorf("--name and --file are required")
			}
			file = pathutil.Expand(file)
			if !pathutil.FileExists(file) {
				return fmt.Errorf("config file not found: %s", file)
			}
			if workshop != "" {
				workshop = pathutil.Expand(workshop)
			}
			p, err := st.AddProfile(store.Profile{
				Name:                name,
				IniPath:             file,
				Build:               buildStr,
				WorkshopContentPath: workshop,
			})
			if err != nil {
				return err
			}
			cmd.Println(styleOK.Render("added profile"), p.ID)
			return nil
		},
	}
	cmd.Flags().String("name", "", "display name")
	cmd.Flags().StringP("file", "f", "", "path to servertest.ini")
	cmd.Flags().String("build", "", "game build: b41 or b42")
	cmd.Flags().String("workshop-path", "", "optional Workshop content dir for mod.info enrichment")
	return cmd
}

func newProfileRemoveCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := st.RemoveProfile(args[0]); err != nil {
				return err
			}
			cmd.Println(styleOK.Render("removed profile"), args[0])
			return nil
		},
	}
}

func newProfileUseCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "use <id>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := st.SetDefaultProfile(args[0]); err != nil {
				return err
			}
			cmd.Println(styleOK.Render("default profile is now"), args[0])
			return nil
		},
	}
}

func newProfileShowCmd(st *store.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "show [id]",
		Short: "Show profile details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var p store.Profile
			var err error
			if len(args) == 1 {
				p, err = st.Profile(args[0])
			} else {
				p, err = st.DefaultProfile()
			}
			if err != nil {
				return err
			}
			cmd.Printf("ID:            %s\n", p.ID)
			cmd.Printf("Name:          %s\n", p.Name)
			cmd.Printf("Config:        %s\n", pathutil.Abbreviate(p.IniPath))
			cmd.Printf("Build:         %s\n", buildBadge(p.Build))
			if p.WorkshopContentPath != "" {
				cmd.Printf("Workshop path: %s\n", pathutil.Abbreviate(p.WorkshopContentPath))
			}
			return nil
		},
	}
}

func buildBadge(b string) string {
	switch b {
	case "b41":
		return styleInfo.Render("[B41]")
	case "b42":
		return styleWarn.Render("[B42]")
	default:
		return styleMuted.Render("[build?]")
	}
}
