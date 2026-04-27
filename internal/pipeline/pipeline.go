package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	ctx "github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/output"
	"github.com/SteelCompendium/steel-etl/internal/parser"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

// Result holds the outcome of a pipeline run.
type Result struct {
	TotalSections      int
	ParsedSections     int
	SkippedSections    int // no parser for the type
	ClassifiedSections int
	WrittenFiles       int
	Errors             []string
}

// Run executes the full pipeline: parse → classify → output.
// This is the legacy entrypoint that produces markdown-only output.
func Run(inputPath string, outputDir string, registryPath string) (*Result, error) {
	cfg := &Config{
		Input:  inputPath,
		Locale: "en",
		Output: OutputConfig{
			BaseDir: filepath.Dir(outputDir), // strip the locale/md suffix
			Formats: []string{"md"},
		},
		Classification: ClassificationConfig{
			Registry: registryPath,
		},
	}
	// The legacy function expected outputDir = base_dir/en/md already, so override
	return RunWithConfig(cfg, inputPath, outputDir, registryPath)
}

// RunWithConfig executes the full pipeline using a Config.
func RunWithConfig(cfg *Config, inputPath, mdOutputDir, registryPath string) (*Result, error) {
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}

	doc, err := parser.ParseDocument(source)
	if err != nil {
		return nil, fmt.Errorf("parse document: %w", err)
	}

	bookSource := ""
	if book, ok := doc.Frontmatter["book"]; ok {
		if bookStr, ok := book.(string); ok {
			bookSource = bookStr
		}
	}

	// Initialize components
	registry := content.NewRegistry()

	// Load existing registry to preserve frozen state and validate stability
	var frozenRegistry *scc.Registry
	sccRegistry := scc.NewRegistry()
	if registryPath != "" {
		if existing, err := scc.LoadRegistry(registryPath); err == nil {
			if existing.IsFrozen() {
				frozenRegistry = existing
				sccRegistry.Freeze()
			}
		}
	}

	contextStack := ctx.NewContextStack(frontmatterToMetadata(doc.Frontmatter))

	// Build the set of output generators
	generators := buildGenerators(cfg, mdOutputDir, registryPath, sccRegistry, source)

	result := &Result{}
	seenSCC := make(map[string]string)

	var walk func(sections []*parser.Section)
	walk = func(sections []*parser.Section) {
		for _, section := range sections {
			result.TotalSections++

			if section.Annotation != nil {
				contextStack.Push(section.HeadingLevel, ctx.Metadata(section.Annotation))
			}

			typeName := section.Type()
			if typeName == "" || !registry.Has(typeName) {
				result.SkippedSections++
				walk(section.Children)
				continue
			}

			p, _ := registry.Get(typeName)
			parsed, err := p.Parse(contextStack, section)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", section.Heading, err))
				walk(section.Children)
				continue
			}
			result.ParsedSections++

			if parsed.TypePath != nil && parsed.ItemID != "" {
				sccCode := scc.Classify(bookSource, parsed.TypePath, parsed.ItemID)
				parsed.Frontmatter["scc"] = sccCode
				sccRegistry.Add(sccCode)
				result.ClassifiedSections++

				// Handle SCC overrides from annotations
				if section.Annotation != nil {
					if override, ok := section.Annotation["scc"]; ok {
						sccCode = override
						parsed.Frontmatter["scc"] = sccCode
					}
					if alias, ok := section.Annotation["scc-alias"]; ok {
						sccRegistry.AddAlias(alias, sccCode)
					}
				}

				// Detect duplicate SCC codes
				if prevHeading, exists := seenSCC[sccCode]; exists {
					result.Errors = append(result.Errors, fmt.Sprintf("duplicate SCC %s: %q overwrites %q", sccCode, section.Heading, prevHeading))
				}
				seenSCC[sccCode] = section.Heading

				// Write to all generators
				for _, gen := range generators {
					if err := gen.WriteSection(sccCode, parsed); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("write %s [%s]: %v", sccCode, gen.Format(), err))
					} else {
						result.WrittenFiles++
					}
				}
			}

			walk(section.Children)
		}
	}
	walk(doc.Sections)

	// Finalize generators that implement BulkGenerator
	for _, gen := range generators {
		if bulk, ok := gen.(output.BulkGenerator); ok {
			if err := bulk.Finalize(); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("finalize [%s]: %v", gen.Format(), err))
			}
		}
	}

	// Validate against frozen registry if applicable
	if frozenRegistry != nil && cfg.Classification.Freeze {
		if err := sccRegistry.ValidateAgainstFrozen(frozenRegistry); err != nil {
			return result, fmt.Errorf("frozen registry violation: %w", err)
		}
	}

	// Save classification registry
	if registryPath != "" {
		if err := sccRegistry.Save(registryPath); err != nil {
			return result, fmt.Errorf("save registry: %w", err)
		}
	}

	return result, nil
}

// buildGenerators creates all configured output generators.
func buildGenerators(cfg *Config, mdOutputDir, registryPath string, sccRegistry *scc.Registry, rawInput []byte) []output.Generator {
	var generators []output.Generator
	locale := cfg.Locale
	if locale == "" {
		locale = "en"
	}

	resolver := scc.NewResolver(sccRegistry, ".md")

	// Base output directory
	baseDir := mdOutputDir
	if cfg.Output.BaseDir != "" && cfg.ConfigDir != "" {
		baseDir = filepath.Join(cfg.ResolvePath(cfg.Output.BaseDir), locale)
	} else if baseDir == "" && cfg.Output.BaseDir != "" {
		baseDir = filepath.Join(cfg.Output.BaseDir, locale)
	}

	// Standard format generators
	for _, format := range cfg.Output.Formats {
		switch format {
		case "md":
			dir := mdOutputDir
			if dir == "" {
				dir = filepath.Join(baseDir, "md")
			}
			generators = append(generators, &output.MarkdownGenerator{BaseDir: dir})
		case "json":
			generators = append(generators, &output.JSONGenerator{
				BaseDir: filepath.Join(baseDir, "json"),
			})
		case "yaml":
			generators = append(generators, &output.YAMLGenerator{
				BaseDir: filepath.Join(baseDir, "yaml"),
			})
		}
	}

	// Variant generators
	if cfg.Output.Variants.Linked {
		generators = append(generators, &output.LinkedGenerator{
			BaseDir:  filepath.Join(baseDir, "md-linked"),
			Resolver: resolver,
		})
	}
	if cfg.Output.Variants.DSE {
		generators = append(generators, &output.DSEGenerator{
			BaseDir: filepath.Join(baseDir, "md-dse"),
		})
	}
	if cfg.Output.Variants.DSELinked {
		generators = append(generators, &output.DSELinkedGenerator{
			BaseDir:  filepath.Join(baseDir, "md-dse-linked"),
			Resolver: resolver,
		})
	}

	// Stripped markdown
	if cfg.Output.Stripped.Enabled && cfg.Output.Stripped.OutputDir != "" {
		outputPath := cfg.ResolvePath(cfg.Output.Stripped.OutputDir)
		// Use the input filename for the stripped output
		inputBase := filepath.Base(cfg.Input)
		if inputBase == "" || inputBase == "." {
			inputBase = "output.md"
		}
		generators = append(generators, &output.StrippedGenerator{
			OutputPath: filepath.Join(outputPath, inputBase),
			RawInput:   rawInput,
		})
	}

	// Aggregation
	if cfg.Output.Aggregate.Enabled && cfg.Output.Aggregate.OutputDir != "" {
		generators = append(generators, &output.AggregateGenerator{
			BaseDir: filepath.Join(cfg.ResolvePath(cfg.Output.Aggregate.OutputDir), locale, "md"),
		})
	}

	// SCC-to-path mapping
	if cfg.Output.SCCMap.Enabled && cfg.Output.SCCMap.OutputFile != "" {
		generators = append(generators, &output.SCCMapGenerator{
			OutputPath: cfg.ResolvePath(cfg.Output.SCCMap.OutputFile),
		})
	}

	return generators
}

func frontmatterToMetadata(fm map[string]any) ctx.Metadata {
	m := make(ctx.Metadata, len(fm))
	for k, v := range fm {
		if s, ok := v.(string); ok {
			m[k] = s
		}
	}
	return m
}
