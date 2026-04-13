package pipeline

import (
	"fmt"
	"os"

	ctx "github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/output"
	"github.com/SteelCompendium/steel-etl/internal/parser"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

// Result holds the outcome of a pipeline run.
type Result struct {
	TotalSections     int
	ParsedSections    int
	SkippedSections   int // no parser for the type
	ClassifiedSections int
	WrittenFiles      int
	Errors            []string
}

// Run executes the full pipeline: parse → classify → output.
func Run(inputPath string, outputDir string, registryPath string) (*Result, error) {
	// Read input
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}

	// Parse document
	doc, err := parser.ParseDocument(source)
	if err != nil {
		return nil, fmt.Errorf("parse document: %w", err)
	}

	// Get book source from frontmatter
	bookSource := ""
	if book, ok := doc.Frontmatter["book"]; ok {
		if bookStr, ok := book.(string); ok {
			bookSource = bookStr
		}
	}

	// Initialize components
	registry := content.NewRegistry()
	sccRegistry := scc.NewRegistry()
	generator := &output.MarkdownGenerator{BaseDir: outputDir}
	contextStack := ctx.NewContextStack(frontmatterToMetadata(doc.Frontmatter))

	result := &Result{}
	seenSCC := make(map[string]string) // sccCode → first heading that used it

	// Walk all sections
	var walk func(sections []*parser.Section)
	walk = func(sections []*parser.Section) {
		for _, section := range sections {
			result.TotalSections++

			// Update context stack
			if section.Annotation != nil {
				contextStack.Push(section.HeadingLevel, ctx.Metadata(section.Annotation))
			}

			typeName := section.Type()
			if typeName == "" || !registry.Has(typeName) {
				result.SkippedSections++
				walk(section.Children)
				continue
			}

			// Parse content
			p, _ := registry.Get(typeName)
			parsed, err := p.Parse(contextStack, section)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", section.Heading, err))
				walk(section.Children)
				continue
			}
			result.ParsedSections++

			// Classify (skip containers like feature-group)
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

				// Write output
				if err := generator.WriteSection(sccCode, parsed); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("write %s: %v", sccCode, err))
				} else {
					result.WrittenFiles++
				}
			}

			walk(section.Children)
		}
	}
	walk(doc.Sections)

	// Save classification registry
	if registryPath != "" {
		if err := sccRegistry.Save(registryPath); err != nil {
			return result, fmt.Errorf("save registry: %w", err)
		}
	}

	return result, nil
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
