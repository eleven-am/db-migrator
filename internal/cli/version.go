package cli

import (
	"fmt"

	"github.com/eleven-am/storm/pkg/storm"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display Storm version and build information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(storm.FullVersionInfo())
	},
}
