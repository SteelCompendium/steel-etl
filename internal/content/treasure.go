package content

import (
	"strings"

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
	body := section.FullBodySource()
	if v := extractField(body, "Level"); v != "" {
		fm["level"] = v
	}
	if v := extractField(body, "Rarity"); v != "" {
		fm["rarity"] = v
	}

	// Extract keywords as array
	if v := extractField(body, "Keywords"); v != "" {
		fm["keywords"] = splitCommaList(v)
	} else if v := extractField(body, "Keyword"); v != "" {
		fm["keywords"] = splitCommaList(v)
	}

	// Extract project-related fields
	if v := extractField(body, "Prerequisite"); v != "" {
		fm["item_prerequisite"] = v
	}
	if v := extractField(body, "Source"); v != "" {
		fm["project_source"] = v
	}
	if v := extractField(body, "Effect"); v != "" {
		fm["effect"] = v
	}

	// Annotation overrides for fields not easily extracted from body
	if ann := section.Annotation; ann != nil {
		for _, key := range []string{"keywords", "item_prerequisite", "project_source", "effect"} {
			if v, ok := ann[key]; ok {
				if key == "keywords" {
					fm[key] = strings.Split(v, ",")
					for i, s := range fm[key].([]string) {
						fm[key].([]string)[i] = strings.TrimSpace(s)
					}
				} else {
					fm[key] = v
				}
			}
		}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"treasure"},
		ItemID:      id,
	}, nil
}
