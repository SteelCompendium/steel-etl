package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ProjectParser handles @type: project sections (downtime projects).
type ProjectParser struct{}

func (p *ProjectParser) Type() string { return "project" }

func (p *ProjectParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	fm := map[string]any{
		"name": name,
		"type": "project",
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    []string{"project"},
		ItemID:      id,
	}, nil
}
