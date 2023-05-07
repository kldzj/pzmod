package commands

import (
	"github.com/spf13/cobra"
)

func Init(rootCmd *cobra.Command) {
	rootCmd.AddCommand(cmdGet())
	rootCmd.AddCommand(cmdSet())
	rootCmd.AddCommand(cmdCopyConfig())
	rootCmd.AddCommand(cmdSetApiKey())
	rootCmd.AddCommand(cmdUpdate())
}

func SetFileFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("file", "f", "", "server config file path")
	cmd.MarkFlagFilename("file", "ini")
	cmd.MarkFlagRequired("file")
}
