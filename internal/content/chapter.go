package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ChapterParser handles @type: chapter sections. Passthrough -- captures content as-is.
type ChapterParser struct{}

func (p *ChapterParser) Type() string { return "chapter" }

func (p *ChapterParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": section.Heading,
			"type": "chapter",
		},
		Body:     section.FullBodySource(),
		TypePath: []string{"chapter"},
		ItemID:   id,
	}, nil
}
