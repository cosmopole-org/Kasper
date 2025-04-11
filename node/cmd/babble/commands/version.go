package commands

import (
	"fmt"

	"kasper/src/version"

	"github.com/spf13/cobra"
)

// VersionCmd displays the version of babble being used
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Version)
	},
}
