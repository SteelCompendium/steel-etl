package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// SkillParser handles @type: skill sections.
type SkillParser struct{}

func (p *SkillParser) Type() string { return "skill" }

func (p *SkillParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	typePath := []string{"skill"}
	if group, ok := section.Annotation["group"]; ok && group != "" {
		typePath = []string{"skill", group}
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": section.Heading,
			"type": "skill",
		},
		Body:     section.FullBodySource(),
		TypePath: typePath,
		ItemID:   id,
	}, nil
}
