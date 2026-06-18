package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// SaintParser handles @type: saint sections — the legendary heroes/saints in the
// Gods and Religion chapter. Saints are flat (religion.saint/<id>): a saint's
// patron god is an explicit @patron annotation, never path nesting, because the
// book places several saints (Pentalion, Eseld, the Saints of Hell) as document
// siblings of their god rather than inside its subtree. Mirrors GodParser.
type SaintParser struct{}

func (p *SaintParser) Type() string { return "saint" }

func (p *SaintParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := headingName(section)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	fm := map[string]any{
		"name": name,
		"type": "saint",
	}
	if v, ok := section.Annotation["patron"]; ok && v != "" {
		fm["patron"] = v
	}
	if d := extractDomains(section.FullBodySource()); d != nil {
		fm["domains"] = d
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    []string{"religion", "saint"},
		ItemID:      id,
	}, nil
}
