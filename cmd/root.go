package cmd

import (
	"os"

	"bunnyshell.com/dev/cmd/remote"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "bunnyshell-dev",
	Short:        "Bunnyshell Development CLI",
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
func init() {
	rootCmd.AddCommand(remote.GetMainCommand())
}
