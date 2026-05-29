package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// MovementParser handles @type: movement sections.
type MovementParser struct{}

func (p *MovementParser) Type() string { return "movement" }

func (p *MovementParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": section.Heading,
			"type": "movement",
		},
		Body:     section.FullBodySource(),
		TypePath: []string{"movement"},
		ItemID:   id,
	}, nil
}
