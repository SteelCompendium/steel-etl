package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// SkillGroupParser handles @type: skill-group sections — the five skill-group
// overview sections (Crafting, Exploration, Interpersonal, Intrigue, Lore).
// Each emits a group-landing page skill.group/<group> (e.g. skill.group/crafting)
// so prose can link to "the <group> skill group". The container pushes NO path
// context: child skills derive their group from their own @group annotation (see
// SkillParser), so the landing page and the leaf skills stay decoupled.
// FullBodySource carries the intro prose + the unannotated skills table; the
// annotated per-skill children are skipped (they become their own pages).
type SkillGroupParser struct{}

func (p *SkillGroupParser) Type() string { return "skill-group" }

func (p *SkillGroupParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": section.Heading,
			"type": "skill-group",
		},
		Body:     section.FullBodySource(),
		TypePath: []string{"skill", "group"},
		ItemID:   id,
	}, nil
}
