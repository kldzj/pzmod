package commands

import (
	"github.com/spf13/cobra"
)

func Init(rootCmd *cobra.Command) {
	rootCmd.AddCommand(cmdGet())
	rootCmd.AddCommand(cmdSet())
	rootCmd.AddCommand(cmdSetApiKey())
	rootCmd.AddCommand(cmdCopyConfig())
	rootCmd.AddCommand(cmdUpdate())
}
