package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/SteelCompendium/steel-etl/internal/pipeline"
	"github.com/SteelCompendium/steel-etl/internal/scc"
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
	genCmd.Flags().String("link-mode", "", "link density mode: all, first, none (default: all)")
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
	if linkMode, _ := cmd.Flags().GetString("link-mode"); linkMode != "" {
		cfg.Output.LinkMode = linkMode
	}

	// Select which book(s) to generate. By default only the primary book is
	// generated; --all generates the primary plus every secondary book, and
	// --book <id> generates a single named book.
	bookFilter, _ := cmd.Flags().GetString("book")
	all, _ := cmd.Flags().GetBool("all")

	configs, err := selectBookConfigs(cfg, bookFilter, all)
	if err != nil {
		return err
	}

	// --all is an authoritative full rebuild of every book, so the SCC registry
	// should reflect exactly the codes this run emits. The per-book pipeline
	// merges into the existing registry (so a single-book run preserves other
	// books' codes), which means codes removed/renamed since the last run would
	// otherwise linger as orphans. Reset the shared registry up front so the
	// per-book accumulation rebuilds a clean set.
	if all {
		if err := resetRegistryForRebuild(cfg); err != nil {
			return err
		}
	}

	var allItems []pipeline.ClassifiedItem
	includesPrimary := false
	for _, bookCfg := range configs {
		result, err := generateBook(bookCfg)
		if err != nil {
			return err
		}
		allItems = append(allItems, result.Classified...)
		if bookCfg.Book == cfg.Book {
			includesPrimary = true
		}
	}

	// Cross-book shared outputs (aggregate / scc_api / scc_map) span every book.
	// Only regenerate them when the run covers the primary plus at least one
	// secondary book (i.e. `--all`); a lone primary run already wrote them per-book,
	// and a lone secondary run must not clobber the shared targets.
	if includesPrimary && len(configs) > 1 {
		fmt.Println("\nRegenerating cross-book shared outputs (aggregate / scc_api / scc_map)...")
		if err := pipeline.RunSharedOutputs(cfg, allItems); err != nil {
			return err
		}
		fmt.Printf("Shared outputs regenerated over %d classified items from %d books.\n", len(allItems), len(configs))
	}
	return nil
}

// resetRegistryForRebuild deletes the SCC registry file before an --all run so
// the rebuild drops codes no longer emitted (orphans from renames/removals); the
// per-book pipeline then accumulates a clean set from empty. A frozen registry is
// left intact — freeze means codes are permanent, and the pipeline's
// ValidateAgainstFrozen enforces stability during the run instead. A missing or
// unreadable registry is a no-op (the run starts fresh anyway).
func resetRegistryForRebuild(cfg *pipeline.Config) error {
	if cfg.Classification.Registry == "" {
		return nil
	}
	registryPath := cfg.ResolvePath(cfg.Classification.Registry)

	existing, err := scc.LoadRegistry(registryPath)
	if err != nil {
		return nil // no registry yet (or unreadable) — nothing to prune
	}
	if existing.IsFrozen() {
		fmt.Printf("Registry: frozen — preserving baseline (orphaned codes not pruned)\n")
		return nil
	}
	if err := os.Remove(registryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reset registry %s: %w", registryPath, err)
	}
	fmt.Printf("Registry: rebuilding from scratch for --all (pruning orphaned codes)\n")
	return nil
}

// selectBookConfigs resolves the gen flags into the list of book configs to run.
func selectBookConfigs(cfg *pipeline.Config, bookFilter string, all bool) ([]*pipeline.Config, error) {
	if all {
		configs := []*pipeline.Config{cfg}
		for _, b := range cfg.Books {
			configs = append(configs, cfg.EffectiveBookConfig(b))
		}
		return configs, nil
	}

	if bookFilter != "" && bookFilter != cfg.Book {
		for _, b := range cfg.Books {
			if b.Book == bookFilter {
				return []*pipeline.Config{cfg.EffectiveBookConfig(b)}, nil
			}
		}
		return nil, fmt.Errorf("book %q not found in config", bookFilter)
	}

	return []*pipeline.Config{cfg}, nil
}

func generateBook(cfg *pipeline.Config) (*pipeline.Result, error) {
	// Resolve paths
	inputPath := cfg.ResolveInputPath()
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
	if cfg.Output.LinkMode != "" {
		fmt.Printf("LinkMode: %s\n", cfg.Output.LinkMode)
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
		return nil, err
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

	return result, nil
}
