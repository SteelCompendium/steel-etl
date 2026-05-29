package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// GodParser handles @type: god sections (deities in the Gods and Religion chapter).
type GodParser struct{}

func (p *GodParser) Type() string { return "god" }

func (p *GodParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	fm := map[string]any{
		"name": name,
		"type": "god",
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    []string{"god"},
		ItemID:      id,
	}, nil
}
