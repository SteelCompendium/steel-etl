package pipeline

import (
	"fmt"
	"os"

	"github.com/SteelCompendium/steel-etl/internal/content"
	ctx "github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

// CollectResult holds the SCC codes collected from a pipeline run without writing output.
type CollectResult struct {
	Codes      []string // all SCC codes found
	Duplicates []string // SCC codes that appear more than once
}

// CollectSCCCodes runs the pipeline parse+classify steps without generating output.
// Returns all SCC codes that would be produced.
func CollectSCCCodes(cfg *Config, inputPath string) (*CollectResult, error) {
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

	registry := content.NewRegistry()
	contextStack := ctx.NewContextStack(frontmatterToMetadata(doc.Frontmatter))

	result := &CollectResult{}
	seen := make(map[string]string) // scc -> heading

	var walk func(sections []*parser.Section)
	walk = func(sections []*parser.Section) {
		for _, section := range sections {
			if section.Annotation != nil {
				contextStack.Push(section.HeadingLevel, ctx.Metadata(section.Annotation))
			}

			typeName := section.Type()
			if typeName == "" || !registry.Has(typeName) {
				walk(section.Children)
				continue
			}

			p, _ := registry.Get(typeName)
			parsed, err := p.Parse(contextStack, section)
			if err != nil {
				walk(section.Children)
				continue
			}

			if parsed.TypePath != nil && parsed.ItemID != "" {
				sccCode := scc.Classify(bookSource, parsed.TypePath, parsed.ItemID)

				// Handle SCC overrides
				if section.Annotation != nil {
					if override, ok := section.Annotation["scc"]; ok {
						sccCode = override
					}
				}

				if prevHeading, exists := seen[sccCode]; exists {
					result.Duplicates = append(result.Duplicates,
						fmt.Sprintf("%s: %q and %q", sccCode, section.Heading, prevHeading))
				}
				seen[sccCode] = section.Heading
				result.Codes = append(result.Codes, sccCode)
			}

			// Parser-emitted coded children (e.g. fixture advancement members) get
			// their own codes too — mirror the main pipeline walk so --scc-stable
			// and other collect-based callers see them.
			for _, child := range parsed.CodedChildren {
				if child.TypePath == nil || child.ItemID == "" {
					continue
				}
				childCode := scc.Classify(bookSource, child.TypePath, child.ItemID)
				if prevHeading, exists := seen[childCode]; exists {
					result.Duplicates = append(result.Duplicates,
						fmt.Sprintf("%s: %q and %q", childCode, fmt.Sprint(child.Frontmatter["name"]), prevHeading))
				}
				seen[childCode] = fmt.Sprint(child.Frontmatter["name"])
				result.Codes = append(result.Codes, childCode)
			}

			walk(section.Children)
		}
	}
	walk(doc.Sections)

	return result, nil
}
