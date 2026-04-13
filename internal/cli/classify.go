package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var classifyCmd = &cobra.Command{
	Use:   "classify [file]",
	Short: "Show or export SCC classifications",
	Long:  `Display SCC classifications for all sections in a document, or export the SCC-to-path mapping.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("classify: not yet implemented")
		return nil
	},
}

func init() {
	classifyCmd.Flags().String("export-map", "", "export scc-to-path mapping to file")
}
