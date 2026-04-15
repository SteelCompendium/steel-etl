package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SteelCompendium/steel-etl/internal/site"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Build the MkDocs site from steel-etl output",
	Long: `Map steel-etl output (md-linked variant) into the MkDocs directory structure.

Reads a site config file that defines how SCC-based content maps to
website sections (tabs). Replaces the v2 bash justfile build pipeline.`,
	RunE: runSite,
}

func init() {
	siteCmd.Flags().StringP("config", "c", "site.yaml", "path to site config file")
}

func runSite(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")

	cfg, err := site.LoadSiteConfig(configPath)
	if err != nil {
		return err
	}

	fmt.Printf("Source:  %s\n", cfg.SourceDir)
	fmt.Printf("Output:  %s\n", cfg.DocsDir)
	fmt.Printf("Sections: %d\n", len(cfg.Sections))
	fmt.Println()

	result, err := site.Build(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("Sections built: %d\n", result.Sections)
	fmt.Printf("Files copied:   %d\n", result.CopiedFiles)
	fmt.Printf("Nav files:      %d\n", result.NavFiles)
	fmt.Printf("Search exclude: %d files\n", result.SearchExclude)

	if len(result.Errors) > 0 {
		fmt.Printf("Errors: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("  %s\n", e)
		}
	}

	return nil
}
