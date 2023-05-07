package main

import (
	"github.com/kldzj/pzmod/commands"
	"github.com/kldzj/pzmod/interactive"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "pzmod --file <server config file>",
		Short: "pzmod is a tool for managing Project Zomboid server mods.",
		Example: `pzmod --file server.ini
pzmod --file server.ini get list
pzmod --file server.ini get name
pzmod --file server.ini set name "My Server"`,
		Run: interactive.Execute,
	}

	rootCmd.PersistentFlags().StringP("file", "f", "", "server config file path")
	rootCmd.MarkPersistentFlagFilename("file", "ini")
	rootCmd.MarkPersistentFlagRequired("file")

	commands.Init(rootCmd)
	rootCmd.Execute()
}
