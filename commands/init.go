package commands

import (
	"github.com/spf13/cobra"
)

func Init(rootCmd *cobra.Command) {
	rootCmd.AddCommand(cmdGet())
	rootCmd.AddCommand(cmdSet())
	rootCmd.AddCommand(cmdCopyConfig())
}
