package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ComplicationParser handles @type: complication sections.
type ComplicationParser struct{}

func (p *ComplicationParser) Type() string { return "complication" }

func (p *ComplicationParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": section.Heading,
			"type": "complication",
		},
		Body:     section.BodySource,
		TypePath: []string{"complication"},
		ItemID:   id,
	}, nil
}
