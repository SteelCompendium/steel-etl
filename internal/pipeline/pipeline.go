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
	"github.com/SteelCompendium/steel-etl/internal/site"
)

// Result holds the outcome of a pipeline run.
type Result struct {
	TotalSections      int
	ParsedSections     int
	SkippedSections    int // no parser for the type
	ClassifiedSections int
	WrittenFiles       int
	Errors             []string
	// Classified holds every classified (sccCode, parsed) pair in document order,
	// so a multi-book orchestrator can feed the cross-book shared outputs
	// (aggregate, scc_api, scc_map) over the union of all books.
	Classified []ClassifiedItem
}

// ClassifiedItem is a single classified section: its SCC code and parsed content.
type ClassifiedItem struct {
	SCCCode string
	Parsed  *content.ParsedContent
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
			for _, code := range existing.Codes() {
				sccRegistry.Add(code)
			}
			for alias, canonical := range existing.Aliases() {
				sccRegistry.AddAlias(alias, canonical)
			}
		}
	}

	contextStack := ctx.NewContextStack(frontmatterToMetadata(doc.Frontmatter))

	// Build the set of output generators
	generators := buildGenerators(cfg, mdOutputDir, registryPath, sccRegistry, source)

	// Clean output directories to remove stale files from previous runs
	if err := output.CleanGeneratorDirs(generators); err != nil {
		return nil, fmt.Errorf("clean output dirs: %w", err)
	}

	result := &Result{}
	seenSCC := make(map[string]string)
	chapterOrder := 0

	// sccBySection maps each classified section to its final (post-override) SCC
	// code so RenderSubtree can mark coded descendant headings. PageBody render +
	// generator writes are deferred until after the walk so the map is complete
	// (a parent is visited before its children, so its descendants' codes are not
	// yet known at parent-render time).
	sccBySection := make(map[*parser.Section]string)
	type pendingWrite struct {
		section *parser.Section
		parsed  *content.ParsedContent
		sccCode string
	}
	var pending []pendingWrite

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

			// Chapters get a per-book document-order index so the site builder can
			// present them in book order rather than alphabetically.
			if typeName == "chapter" {
				parsed.Frontmatter["order"] = chapterOrder
				chapterOrder++
			}

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

				// Record the final code so coded descendant headings can be marked
				// in PageBody, and defer the render + writes until the walk is done.
				sccBySection[section] = sccCode
				pending = append(pending, pendingWrite{section: section, parsed: parsed, sccCode: sccCode})
			}

			walk(section.Children)
		}
	}
	walk(doc.Sections)

	// Now that every section's SCC code is known, render each page's book-order
	// PageBody (marking coded descendant headings with data-scc) and write to all
	// generators. Deferred from the walk so the sccBySection map is complete.
	//
	// Monster *group* pages (@type: monster) are an exception: they are modular
	// Browse landing pages that show only the group's lore, NOT its statblocks and
	// malice. Leaving their PageBody empty makes the reading-format generators fall
	// back to the lore-only Body. The book-faithful, everything-inline view still
	// exists on the Read tab via the chapter page's PageBody (rendered separately).
	for _, pw := range pending {
		if t, _ := pw.parsed.Frontmatter["type"].(string); t != "monster" {
			pw.parsed.PageBody = content.RenderSubtree(pw.section, sccBySection)
		}
		for _, gen := range generators {
			if err := gen.WriteSection(pw.sccCode, pw.parsed); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("write %s [%s]: %v", pw.sccCode, gen.Format(), err))
			} else {
				result.WrittenFiles++
			}
		}
		result.Classified = append(result.Classified, ClassifiedItem{SCCCode: pw.sccCode, Parsed: pw.parsed})
	}

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

// RunSharedOutputs regenerates the cross-book shared outputs (aggregate / scc_map /
// scc_api) over the union of classified items from every book, so secondary books
// (e.g. monsters, beastheart) join the same SCC API and data-unified aggregate as
// the primary book. It is invoked by the orchestrator after all books have been
// generated (which leaves the shared registry fully populated on disk).
//
// Per-book generation still writes these shared outputs for the primary book; this
// pass cleans and rewrites them from the complete set, so it is a strict superset.
func RunSharedOutputs(cfg *Config, items []ClassifiedItem) error {
	// Specialize the config to emit ONLY the shared cross-book outputs: no per-book
	// formats/variants/stripped (those were already written per book).
	shared := *cfg
	out := cfg.Output
	out.Formats = nil
	out.Variants = VariantsConfig{}
	out.Stripped.Enabled = false
	shared.Output = out

	// Load the full registry (written by every book run) so aliases are complete.
	sccRegistry := scc.NewRegistry()
	registryPath := ""
	if cfg.Classification.Registry != "" {
		registryPath = cfg.ResolvePath(cfg.Classification.Registry)
		if existing, err := scc.LoadRegistry(registryPath); err == nil {
			for _, code := range existing.Codes() {
				sccRegistry.Add(code)
			}
			for alias, canonical := range existing.Aliases() {
				sccRegistry.AddAlias(alias, canonical)
			}
		}
	}

	generators := buildGenerators(&shared, "", registryPath, sccRegistry, nil)
	if len(generators) == 0 {
		return nil
	}

	if err := output.CleanGeneratorDirs(generators); err != nil {
		return fmt.Errorf("clean shared output dirs: %w", err)
	}

	for _, item := range items {
		for _, gen := range generators {
			if err := gen.WriteSection(item.SCCCode, item.Parsed); err != nil {
				return fmt.Errorf("shared write %s [%s]: %w", item.SCCCode, gen.Format(), err)
			}
		}
	}

	for _, gen := range generators {
		if bulk, ok := gen.(output.BulkGenerator); ok {
			if err := bulk.Finalize(); err != nil {
				return fmt.Errorf("shared finalize [%s]: %w", gen.Format(), err)
			}
		}
	}

	return nil
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
			LinkMode: cfg.Output.ParseLinkMode(),
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
			LinkMode: cfg.Output.ParseLinkMode(),
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

	// SCC resolution API
	if cfg.Output.SCCAPI.Enabled && cfg.Output.SCCAPI.OutputDir != "" {
		apiGen := &output.SCCAPIGenerator{
			OutputDir: cfg.ResolvePath(cfg.Output.SCCAPI.OutputDir),
			BaseURL:   cfg.Output.SCCAPI.BaseURL,
			Aliases:   sccRegistry.Aliases(),
		}
		if cfg.Output.SCCAPI.SiteConfig != "" {
			siteCfg, err := site.LoadSiteConfig(cfg.ResolvePath(cfg.Output.SCCAPI.SiteConfig))
			if err == nil {
				apiGen.Sections = siteCfg.Sections
			}
		}
		generators = append(generators, apiGen)
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
