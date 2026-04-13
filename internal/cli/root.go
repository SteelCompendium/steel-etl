package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "steel-etl",
	Short: "Process annotated Draw Steel markdown into structured output",
	Long: `steel-etl parses annotated Draw Steel TTRPG markdown and produces
per-section output files in multiple formats (markdown, JSON, YAML)
with full YAML frontmatter and SCC classification.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(genCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(classifyCmd)
	rootCmd.AddCommand(stripCmd)
}
