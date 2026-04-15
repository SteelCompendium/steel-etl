package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/SteelCompendium/steel-etl/internal/output"
)

var stripCmd = &cobra.Command{
	Use:   "strip [file]",
	Short: "Remove annotations from markdown",
	Long: `Strip all <!-- @... --> annotations and YAML frontmatter, producing clean markdown.

With --for-translation, keeps annotations intact but prepends a translation guide
header, producing a template ready for translators to fill in.`,
	Args: cobra.ExactArgs(1),
	RunE: runStrip,
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

	forTranslation, _ := cmd.Flags().GetBool("for-translation")

	var result string
	if forTranslation {
		result = output.TranslationTemplate(string(data), filepath.Base(inputPath))
	} else {
		result = output.StripAnnotations(string(data))
	}

	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath == "" {
		fmt.Print(result)
		return nil
	}

	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	if forTranslation {
		fmt.Printf("Translation template written to %s\n", outputPath)
	} else {
		fmt.Printf("Stripped output written to %s\n", outputPath)
	}
	return nil
}
