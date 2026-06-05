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

	// Resolve echelon (item annotation → ancestor context) and record it.
	echelon := ""
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["echelon"]; ok {
			echelon = v
		}
	}
	if echelon == "" {
		if v, ok := ctx.Lookup(section.HeadingLevel, "echelon"); ok {
			echelon = v
		}
	}
	if echelon != "" {
		fm["echelon"] = echelon
	}

	// Category (consumable/trinket/armor/implement/weapon/other) was resolved
	// into fm["treasure_type"] above from annotation or ancestor context.
	category, _ := fm["treasure_type"].(string)

	// Nested type path: treasure/<tier>/<category>. tier is the echelon slug
	// (1st-echelon…4th-echelon) or "leveled" when the treasure has no echelon.
	typePath := []string{"treasure"}
	tier := echelonSlug(echelon)
	if tier == "" {
		tier = "leveled"
	}
	typePath = append(typePath, tier)
	if category != "" {
		typePath = append(typePath, category)
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}

// echelonSlug converts an echelon number ("1".."4") into its tier slug
// ("1st-echelon".."4th-echelon"). Any other value returns "".
func echelonSlug(echelon string) string {
	switch strings.TrimSpace(echelon) {
	case "1":
		return "1st-echelon"
	case "2":
		return "2nd-echelon"
	case "3":
		return "3rd-echelon"
	case "4":
		return "4th-echelon"
	default:
		return ""
	}
}

// TreasureGroupParser handles @type: treasure-group sections — structural
// category containers (e.g. "1st-Echelon Consumables", "Leveled Weapon
// Treasures") that provide echelon + treasure-type context to child treasures.
// Like FeatureGroupParser, it produces no standalone output file; the pipeline
// pushes its annotation into the context stack regardless.
type TreasureGroupParser struct{}

func (p *TreasureGroupParser) Type() string { return "treasure-group" }

func (p *TreasureGroupParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	fm := map[string]any{
		"name": section.Heading,
		"type": "treasure-group",
	}
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["echelon"]; ok {
			fm["echelon"] = v
		}
		if v, ok := ann["treasure-type"]; ok {
			fm["treasure_type"] = v
		}
	}
	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
	}, nil
}
