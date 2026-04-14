package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// TitleParser handles @type: title sections.
type TitleParser struct{}

func (p *TitleParser) Type() string { return "title" }

func (p *TitleParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "title",
	}

	// Extract echelon from annotation or body
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["echelon"]; ok {
			fm["echelon"] = v
		}
	}
	if _, ok := fm["echelon"]; !ok {
		if v := extractField(section.FullBodySource(), "Echelon"); v != "" {
			fm["echelon"] = v
		}
	}

	// Look up echelon from parent context if not set
	if _, ok := fm["echelon"]; !ok {
		if echelon, ok := ctx.Lookup(section.HeadingLevel, "echelon"); ok {
			fm["echelon"] = echelon
		}
	}

	// Extract benefits as list
	benefits := extractListField(section.FullBodySource(), "Benefits")
	if len(benefits) == 0 {
		benefits = extractListField(section.FullBodySource(), "Benefit")
	}
	if len(benefits) > 0 {
		fm["benefits"] = benefits
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    []string{"title"},
		ItemID:      id,
	}, nil
}
