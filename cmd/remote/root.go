package remote

import (
	"github.com/spf13/cobra"
)

var mainCmd = &cobra.Command{
	Use:   "remote",
	Short: "Remote Development",
}

func GetMainCommand() *cobra.Command {
	return mainCmd
}
