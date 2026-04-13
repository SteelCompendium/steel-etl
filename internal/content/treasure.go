package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// TreasureParser handles @type: treasure sections.
type TreasureParser struct{}

func (p *TreasureParser) Type() string { return "treasure" }

func (p *TreasureParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "treasure",
	}

	// Determine treasure_type from annotation or body
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["treasure-type"]; ok {
			fm["treasure_type"] = v
		}
		if v, ok := ann["treasure_type"]; ok {
			fm["treasure_type"] = v
		}
	}

	// Auto-detect treasure type from parent context or body
	if _, ok := fm["treasure_type"]; !ok {
		if tt, ok := ctx.Lookup(section.HeadingLevel, "treasure-type"); ok {
			fm["treasure_type"] = tt
		}
	}

	// Extract properties
	body := section.BodySource
	if v := extractField(body, "Level"); v != "" {
		fm["level"] = v
	}
	if v := extractField(body, "Rarity"); v != "" {
		fm["rarity"] = v
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"treasure"},
		ItemID:      id,
	}, nil
}
