package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate annotations and content structure",
	Long:  `Check annotation syntax, coverage, and content structure without generating output.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("validate: not yet implemented")
		return nil
	},
}

func init() {
	validateCmd.Flags().Bool("scc-stable", false, "verify no existing SCC codes have changed")
}
