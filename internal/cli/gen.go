package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/SteelCompendium/steel-etl/internal/pipeline"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Run the pipeline to generate output files",
	Long: `Parse annotated markdown and generate per-section output files
in the configured formats (md, json, yaml) with YAML frontmatter.`,
	RunE: runGen,
}

func init() {
	genCmd.Flags().StringP("config", "c", "pipeline.yaml", "path to pipeline config file")
	genCmd.Flags().String("format", "", "output format filter (md, json, yaml)")
	genCmd.Flags().String("locale", "", "locale override")
	genCmd.Flags().Bool("all", false, "generate all books")
}

type pipelineConfig struct {
	Book           string `yaml:"book"`
	Input          string `yaml:"input"`
	Locale         string `yaml:"locale"`
	Classification struct {
		Registry string `yaml:"registry"`
		Freeze   bool   `yaml:"freeze"`
	} `yaml:"classification"`
	Output struct {
		BaseDir string `yaml:"base_dir"`
	} `yaml:"output"`
}

func runGen(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")

	// Read config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", configPath, err)
	}

	var cfg pipelineConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	// Resolve paths relative to config file
	configDir := filepath.Dir(configPath)
	inputPath := resolvePath(configDir, cfg.Input)
	outputDir := resolvePath(configDir, cfg.Output.BaseDir)
	registryPath := ""
	if cfg.Classification.Registry != "" {
		registryPath = resolvePath(configDir, cfg.Classification.Registry)
	}

	locale := cfg.Locale
	if l, _ := cmd.Flags().GetString("locale"); l != "" {
		locale = l
	}
	if locale == "" {
		locale = "en"
	}

	// Output to locale subdirectory: base_dir/locale/md/
	mdOutputDir := filepath.Join(outputDir, locale, "md")

	fmt.Printf("Input:    %s\n", inputPath)
	fmt.Printf("Output:   %s\n", mdOutputDir)
	if registryPath != "" {
		fmt.Printf("Registry: %s\n", registryPath)
	}
	fmt.Println()

	result, err := pipeline.Run(inputPath, mdOutputDir, registryPath)
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

func resolvePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}
