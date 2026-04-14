package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ConditionParser handles @type: condition sections.
type ConditionParser struct{}

func (p *ConditionParser) Type() string { return "condition" }

func (p *ConditionParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": section.Heading,
			"type": "condition",
		},
		Body:     section.FullBodySource(),
		TypePath: []string{"condition"},
		ItemID:   id,
	}, nil
}
