package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/parser"
	"github.com/SteelCompendium/steel-etl/internal/pipeline"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate annotations and content structure",
	Long: `Check annotation syntax, coverage, and content structure without generating output.

Reports:
  - Annotation syntax errors (malformed comments, unknown fields)
  - Sections missing annotations (coverage gaps)
  - Unknown @type values (no registered parser)
  - SCC stability violations (when --scc-stable is set)
  - Duplicate SCC codes`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().Bool("scc-stable", false, "verify no existing SCC codes have changed or been removed")
	validateCmd.Flags().StringP("config", "c", "pipeline.yaml", "path to pipeline config file")
}

func runValidate(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	sccStable, _ := cmd.Flags().GetBool("scc-stable")

	cfg, err := pipeline.LoadConfig(configPath)
	if err != nil {
		return err
	}

	// Determine input path
	inputPath := cfg.ResolveInputPath()
	if len(args) > 0 {
		inputPath = args[0]
	}

	source, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	fmt.Printf("Validating: %s\n\n", inputPath)

	// --- 1. Parse document ---
	doc, err := parser.ParseDocument(source)
	if err != nil {
		return fmt.Errorf("parse document: %w", err)
	}

	// --- 2. Validate annotations ---
	registry := content.NewRegistry()
	var issues []validationIssue

	totalSections := 0
	annotatedSections := 0
	unannotatedSections := 0
	unknownTypes := 0
	parsedOK := 0

	var walkSections func(sections []*parser.Section, depth int)
	walkSections = func(sections []*parser.Section, depth int) {
		for _, sec := range sections {
			totalSections++

			if sec.Annotation == nil {
				unannotatedSections++
				// Only warn for H1-H4 sections (lower headings are typically sub-content)
				if sec.HeadingLevel <= 4 {
					issues = append(issues, validationIssue{
						level:   "info",
						heading: sec.Heading,
						hlevel:  sec.HeadingLevel,
						msg:     "section has no annotation",
					})
				}
			} else {
				annotatedSections++
				typeName := sec.Type()

				if typeName == "" {
					issues = append(issues, validationIssue{
						level:   "warn",
						heading: sec.Heading,
						hlevel:  sec.HeadingLevel,
						msg:     "annotation present but no @type field",
					})
				} else if !registry.Has(typeName) {
					unknownTypes++
					issues = append(issues, validationIssue{
						level:   "error",
						heading: sec.Heading,
						hlevel:  sec.HeadingLevel,
						msg:     fmt.Sprintf("unknown @type: %q (no registered parser)", typeName),
					})
				} else {
					parsedOK++
				}
			}

			walkSections(sec.Children, depth+1)
		}
	}
	walkSections(doc.Sections, 0)

	// --- 3. Run the pipeline to check SCC stability ---
	if sccStable {
		registryPath := ""
		if cfg.Classification.Registry != "" {
			registryPath = cfg.ResolvePath(cfg.Classification.Registry)
		}

		if registryPath == "" {
			issues = append(issues, validationIssue{
				level: "error",
				msg:   "--scc-stable: no classification registry configured",
			})
		} else {
			frozenRegistry, err := scc.LoadRegistry(registryPath)
			if err != nil {
				issues = append(issues, validationIssue{
					level: "error",
					msg:   fmt.Sprintf("--scc-stable: cannot load registry: %v", err),
				})
			} else {
				// Run the pipeline in dry mode to collect SCC codes
				mdOutputDir := cfg.ResolvePath(cfg.Output.BaseDir) + "/" + cfg.Locale + "/md"
				result, err := pipeline.RunWithConfig(cfg, inputPath, mdOutputDir, "")
				if err != nil {
					issues = append(issues, validationIssue{
						level: "error",
						msg:   fmt.Sprintf("--scc-stable: pipeline run failed: %v", err),
					})
				} else {
					// Check for duplicate SCC codes from pipeline
					for _, e := range result.Errors {
						if len(e) > 14 && e[:14] == "duplicate SCC " {
							issues = append(issues, validationIssue{
								level: "error",
								msg:   e,
							})
						}
					}

					// Build a registry from the pipeline run to compare
					// The pipeline already wrote its SCC codes, but we need to check
					// against the frozen registry by re-running classification
					newRegistry := scc.NewRegistry()
					// Re-parse to collect SCC codes without writing
					collectResult, err := pipeline.CollectSCCCodes(cfg, inputPath)
					if err != nil {
						issues = append(issues, validationIssue{
							level: "error",
							msg:   fmt.Sprintf("--scc-stable: SCC collection failed: %v", err),
						})
					} else {
						for _, code := range collectResult.Codes {
							newRegistry.Add(code)
						}

						// Check: every code in the frozen registry must still exist
						if err := newRegistry.ValidateAgainstFrozen(frozenRegistry); err != nil {
							issues = append(issues, validationIssue{
								level: "error",
								msg:   fmt.Sprintf("--scc-stable: %v", err),
							})
						}

						// Check for duplicates
						for _, dup := range collectResult.Duplicates {
							issues = append(issues, validationIssue{
								level: "error",
								msg:   fmt.Sprintf("duplicate SCC: %s", dup),
							})
						}
					}
				}
			}
		}
	}

	// --- 4. Report results ---
	fmt.Println("=== Coverage ===")
	fmt.Printf("  Total sections:     %d\n", totalSections)
	fmt.Printf("  Annotated:          %d\n", annotatedSections)
	fmt.Printf("  Unannotated:        %d\n", unannotatedSections)
	fmt.Printf("  Valid @type:        %d\n", parsedOK)
	fmt.Printf("  Unknown @type:      %d\n", unknownTypes)
	if totalSections > 0 {
		coverage := float64(annotatedSections) / float64(totalSections) * 100
		fmt.Printf("  Coverage:           %.1f%%\n", coverage)
	}
	fmt.Println()

	// Group and display issues
	errors := filterIssues(issues, "error")
	warns := filterIssues(issues, "warn")
	infos := filterIssues(issues, "info")

	if len(errors) > 0 {
		fmt.Printf("=== Errors (%d) ===\n", len(errors))
		for _, issue := range errors {
			fmt.Printf("  ERROR: %s\n", issue.String())
		}
		fmt.Println()
	}

	if len(warns) > 0 {
		fmt.Printf("=== Warnings (%d) ===\n", len(warns))
		for _, issue := range warns {
			fmt.Printf("  WARN:  %s\n", issue.String())
		}
		fmt.Println()
	}

	if len(infos) > 0 {
		fmt.Printf("=== Info (%d unannotated sections) ===\n", len(infos))
		// Summarize by heading level instead of listing all
		levelCounts := make(map[int]int)
		for _, issue := range infos {
			levelCounts[issue.hlevel]++
		}
		for level := 1; level <= 6; level++ {
			if count, ok := levelCounts[level]; ok {
				fmt.Printf("  H%d: %d unannotated\n", level, count)
			}
		}
		fmt.Println()
	}

	// Exit with error if there are any errors
	if len(errors) > 0 {
		return fmt.Errorf("validation failed with %d error(s)", len(errors))
	}

	fmt.Println("Validation passed.")
	return nil
}

type validationIssue struct {
	level   string // "error", "warn", "info"
	heading string
	hlevel  int
	msg     string
}

func (v validationIssue) String() string {
	if v.heading != "" {
		return fmt.Sprintf("[H%d %q] %s", v.hlevel, v.heading, v.msg)
	}
	return v.msg
}

func filterIssues(issues []validationIssue, level string) []validationIssue {
	var filtered []validationIssue
	for _, issue := range issues {
		if issue.level == level {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}
