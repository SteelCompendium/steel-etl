package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// AncestryParser handles @type: ancestry sections.
type AncestryParser struct{}

func (p *AncestryParser) Type() string { return "ancestry" }

func (p *AncestryParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "ancestry",
	}

	body := section.FullBodySource()

	// Extract signature trait
	if v := extractField(body, "Signature Trait"); v != "" {
		fm["signature_trait"] = v
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"ancestry"},
		ItemID:      id,
	}, nil
}
