package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SteelCompendium/steel-etl/internal/pipeline"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Run the pipeline to generate output files",
	Long: `Parse annotated markdown and generate per-section output files
in the configured formats (md, json, yaml) with YAML frontmatter.

Supports multiple output formats, variants (linked, DSE), stripped
markdown, aggregation, and SCC-to-path mapping.`,
	RunE: runGen,
}

func init() {
	genCmd.Flags().StringP("config", "c", "pipeline.yaml", "path to pipeline config file")
	genCmd.Flags().String("format", "", "output format filter (md, json, yaml)")
	genCmd.Flags().String("locale", "", "locale override")
	genCmd.Flags().String("book", "", "book filter (e.g., mcdm.heroes.v1)")
	genCmd.Flags().Bool("all", false, "generate all books")
}

func runGen(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")

	cfg, err := pipeline.LoadConfig(configPath)
	if err != nil {
		return err
	}

	// Apply CLI overrides
	if locale, _ := cmd.Flags().GetString("locale"); locale != "" {
		cfg.Locale = locale
	}
	if format, _ := cmd.Flags().GetString("format"); format != "" {
		cfg.Output.Formats = []string{format}
		// Disable variants when filtering to a single format
		cfg.Output.Variants.Linked = false
		cfg.Output.Variants.DSE = false
		cfg.Output.Variants.DSELinked = false
	}

	// Resolve paths
	inputPath := cfg.ResolvePath(cfg.Input)
	registryPath := ""
	if cfg.Classification.Registry != "" {
		registryPath = cfg.ResolvePath(cfg.Classification.Registry)
	}

	locale := cfg.Locale
	mdOutputDir := cfg.ResolvePath(cfg.Output.BaseDir)
	mdOutputDir = mdOutputDir + "/" + locale + "/md"

	fmt.Printf("Book:     %s\n", cfg.Book)
	fmt.Printf("Input:    %s\n", inputPath)
	fmt.Printf("Output:   %s\n", mdOutputDir)
	fmt.Printf("Formats:  %v\n", cfg.Output.Formats)
	fmt.Printf("Locale:   %s\n", locale)
	if registryPath != "" {
		fmt.Printf("Registry: %s\n", registryPath)
	}
	if cfg.Output.Variants.Linked {
		fmt.Println("Variant:  linked")
	}
	if cfg.Output.Variants.DSE {
		fmt.Println("Variant:  dse")
	}
	if cfg.Output.Variants.DSELinked {
		fmt.Println("Variant:  dse-linked")
	}
	if cfg.Output.Stripped.Enabled {
		fmt.Printf("Stripped: %s\n", cfg.ResolvePath(cfg.Output.Stripped.OutputDir))
	}
	if cfg.Output.Aggregate.Enabled {
		fmt.Printf("Aggregate: %s\n", cfg.ResolvePath(cfg.Output.Aggregate.OutputDir))
	}
	if cfg.Output.SCCMap.Enabled {
		fmt.Printf("SCC Map: %s\n", cfg.ResolvePath(cfg.Output.SCCMap.OutputFile))
	}
	fmt.Println()

	result, err := pipeline.RunWithConfig(cfg, inputPath, mdOutputDir, registryPath)
	if err != nil {
		return err
	}

	fmt.Printf("Sections: %d total, %d parsed, %d skipped\n",
		result.TotalSections, result.ParsedSections, result.SkippedSections)
	fmt.Printf("Classified: %d, Written: %d files\n",
		result.ClassifiedSections, result.WrittenFiles)

	if len(result.Errors) > 0 {
		fmt.Printf("Errors: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("  %s\n", e)
		}
	}

	return nil
}
