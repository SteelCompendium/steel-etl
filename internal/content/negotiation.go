package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// NegotiationParser handles @type: negotiation sections.
type NegotiationParser struct{}

func (p *NegotiationParser) Type() string { return "negotiation" }

func (p *NegotiationParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": section.Heading,
			"type": "negotiation",
		},
		Body:     section.FullBodySource(),
		TypePath: []string{"negotiation"},
		ItemID:   id,
	}, nil
}
