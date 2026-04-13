package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stripCmd = &cobra.Command{
	Use:   "strip [file]",
	Short: "Remove annotations from markdown",
	Long:  `Strip all <!-- @... --> annotations and YAML frontmatter, producing clean markdown.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("strip: not yet implemented")
		return nil
	},
}

func init() {
	stripCmd.Flags().StringP("output", "o", "", "output file path")
	stripCmd.Flags().Bool("for-translation", false, "produce translation-ready template")
}
