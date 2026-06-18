package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// GodParser handles @type: god sections (deities in the Gods and Religion chapter).
type GodParser struct{}

func (p *GodParser) Type() string { return "god" }

func (p *GodParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := headingName(section)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	fm := map[string]any{
		"name": name,
		"type": "god",
	}
	if d := extractDomains(section.FullBodySource()); d != nil {
		fm["domains"] = d
	}
	for _, key := range []string{"pantheon", "alignment", "god_class"} {
		if v, ok := section.Annotation[key]; ok && v != "" {
			fm[key] = v
		}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    []string{"religion", "god"},
		ItemID:      id,
	}, nil
}
