package cli

import (
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/kldzj/pzmod/pkg/service"
	"github.com/kldzj/pzmod/pkg/steam"
	"github.com/kldzj/pzmod/pkg/store"
	"github.com/spf13/cobra"
)

func newSearchCmd(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <text...>",
		Short: "Search the Steam Workshop for Project Zomboid mods",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, _ := cmd.Flags().GetString("profile")
			if !st.HasAPIKey(profile) {
				return errNoKey
			}
			key, _ := st.APIKey(profile)
			svc := service.New(steamFactory(key), st)

			limit, _ := cmd.Flags().GetInt("limit")
			tags, _ := cmd.Flags().GetStringArray("tag")
			page, err := svc.Search(cmd.Context(), steam.Query{
				SearchText: strings.Join(args, " "),
				Tags:       tags,
				PerPage:    limit,
			})
			if err != nil {
				return err
			}

			if jsonEnabled(cmd) {
				out := searchJSON{Total: page.Total, Items: make([]searchItemJSON, 0, len(page.Items))}
				for _, it := range page.Items {
					out.Items = append(out.Items, searchItemJSON{
						ID:       it.PublishedFileID,
						Title:    it.Title,
						FileSize: int64(it.FileSize),
					})
				}
				return emitJSON(cmd, out)
			}

			cmd.Printf("%s\n", styleMuted.Render(humanize.Comma(int64(page.Total))+" results"))
			for _, it := range page.Items {
				cmd.Printf("%s  %s  %s\n",
					styleInfo.Render(it.PublishedFileID),
					it.Title,
					styleMuted.Render(humanize.Bytes(uint64(it.FileSize))))
			}
			return nil
		},
	}
	cmd.Flags().IntP("limit", "l", 20, "max results")
	cmd.Flags().StringArray("tag", nil, "required Workshop tag (repeatable)")
	cmd.Flags().StringP("profile", "p", "", "use a profile's API key")
	return cmd
}
