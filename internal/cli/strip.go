package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/SteelCompendium/steel-etl/internal/output"
)

var stripCmd = &cobra.Command{
	Use:   "strip [file]",
	Short: "Remove annotations from markdown",
	Long:  `Strip all <!-- @... --> annotations and YAML frontmatter, producing clean markdown.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runStrip,
}

func init() {
	stripCmd.Flags().StringP("output", "o", "", "output file path (default: stdout)")
	stripCmd.Flags().Bool("for-translation", false, "produce translation-ready template")
}

func runStrip(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	clean := output.StripAnnotations(string(data))

	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath == "" {
		fmt.Print(clean)
		return nil
	}

	if err := os.WriteFile(outputPath, []byte(clean), 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("Stripped output written to %s\n", outputPath)
	return nil
}
