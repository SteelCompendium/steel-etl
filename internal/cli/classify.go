package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/SteelCompendium/steel-etl/internal/pipeline"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

var classifyCmd = &cobra.Command{
	Use:   "classify [file]",
	Short: "Show or export SCC classifications",
	Long: `Display SCC classifications for all sections in a document, or export the SCC-to-path mapping.

Without flags, prints all SCC codes grouped by type component.
With --export-map, writes the SCC-to-path mapping to a JSON file.
With --diff, shows changes against the existing classification registry.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runClassify,
}

func init() {
	classifyCmd.Flags().String("export-map", "", "export scc-to-path mapping to file")
	classifyCmd.Flags().Bool("diff", false, "show diff against existing registry (new, removed codes)")
	classifyCmd.Flags().StringP("config", "c", "pipeline.yaml", "path to pipeline config file")
}

func runClassify(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	exportMap, _ := cmd.Flags().GetString("export-map")
	showDiff, _ := cmd.Flags().GetBool("diff")

	cfg, err := pipeline.LoadConfig(configPath)
	if err != nil {
		return err
	}

	inputPath := cfg.ResolveInputPath()
	if len(args) > 0 {
		inputPath = args[0]
	}

	fmt.Printf("Classifying: %s\n", inputPath)
	fmt.Printf("Book:        %s\n\n", cfg.Book)

	// Collect SCC codes
	result, err := pipeline.CollectSCCCodes(cfg, inputPath)
	if err != nil {
		return err
	}

	// Group codes by type component
	byType := groupByType(result.Codes)
	types := sortedKeys(byType)

	// Display all codes grouped by type
	fmt.Printf("=== SCC Codes (%d total) ===\n\n", len(result.Codes))
	for _, typeName := range types {
		codes := byType[typeName]
		sort.Strings(codes)
		fmt.Printf("  %s (%d)\n", typeName, len(codes))
		for _, code := range codes {
			fmt.Printf("    %s\n", code)
		}
		fmt.Println()
	}

	// Report duplicates
	if len(result.Duplicates) > 0 {
		fmt.Printf("=== Duplicates (%d) ===\n", len(result.Duplicates))
		for _, dup := range result.Duplicates {
			fmt.Printf("  %s\n", dup)
		}
		fmt.Println()
	}

	// Diff against existing registry
	if showDiff {
		registryPath := ""
		if cfg.Classification.Registry != "" {
			registryPath = cfg.ResolvePath(cfg.Classification.Registry)
		}
		if registryPath == "" {
			return fmt.Errorf("--diff: no classification registry configured")
		}

		existing, err := scc.LoadRegistry(registryPath)
		if err != nil {
			return fmt.Errorf("--diff: load registry: %w", err)
		}

		existingCodes := existing.Codes()
		newCodes := result.Codes

		existingSet := toSet(existingCodes)
		newSet := toSet(newCodes)

		var added, removed []string
		for _, code := range newCodes {
			if !existingSet[code] {
				added = append(added, code)
			}
		}
		for _, code := range existingCodes {
			if !newSet[code] {
				removed = append(removed, code)
			}
		}
		sort.Strings(added)
		sort.Strings(removed)

		fmt.Printf("=== Registry Diff ===\n")
		fmt.Printf("  Existing: %d codes\n", len(existingCodes))
		fmt.Printf("  Current:  %d codes\n", len(newCodes))
		fmt.Println()

		if len(added) > 0 {
			fmt.Printf("  Added (%d):\n", len(added))
			for _, code := range added {
				fmt.Printf("    + %s\n", code)
			}
			fmt.Println()
		}

		if len(removed) > 0 {
			fmt.Printf("  Removed (%d):\n", len(removed))
			for _, code := range removed {
				fmt.Printf("    - %s\n", code)
			}
			fmt.Println()
		}

		if len(added) == 0 && len(removed) == 0 {
			fmt.Println("  No changes.")
		}
		fmt.Println()
	}

	// Export SCC-to-path mapping
	if exportMap != "" {
		type mapEntry struct {
			SCC  string `json:"scc"`
			Type string `json:"type"`
		}

		var entries []mapEntry
		for _, code := range result.Codes {
			parts := strings.SplitN(code, "/", 3)
			typePart := ""
			if len(parts) >= 2 {
				typePart = parts[1]
			}
			entries = append(entries, mapEntry{
				SCC:  code,
				Type: typePart,
			})
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].SCC < entries[j].SCC
		})

		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal export map: %w", err)
		}
		data = append(data, '\n')

		if err := os.WriteFile(exportMap, data, 0644); err != nil {
			return fmt.Errorf("write export map: %w", err)
		}
		fmt.Printf("Exported %d entries to %s\n", len(entries), exportMap)
	}

	return nil
}

// groupByType groups SCC codes by their type component (second part of source/type/item).
func groupByType(codes []string) map[string][]string {
	grouped := make(map[string][]string)
	for _, code := range codes {
		parts := strings.SplitN(code, "/", 3)
		typePart := "(unknown)"
		if len(parts) >= 2 {
			typePart = parts[1]
		}
		// Group by top-level type (e.g., "feature.ability.fury.level-1" -> "feature.ability")
		topType := typePart
		dotParts := strings.SplitN(typePart, ".", 3)
		if len(dotParts) >= 2 {
			topType = dotParts[0] + "." + dotParts[1]
		}
		grouped[topType] = append(grouped[topType], code)
	}
	return grouped
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func toSet(slice []string) map[string]bool {
	s := make(map[string]bool, len(slice))
	for _, v := range slice {
		s[v] = true
	}
	return s
}
