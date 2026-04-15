package content

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// PerkParser handles @type: perk sections.
type PerkParser struct{}

func (p *PerkParser) Type() string { return "perk" }

func (p *PerkParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "perk",
	}

	body := section.FullBodySource()

	// Extract prerequisites from body
	if prereq := extractField(body, "Prerequisite"); prereq != "" {
		fm["prerequisites"] = prereq
	} else if prereq := extractField(body, "Prerequisites"); prereq != "" {
		fm["prerequisites"] = prereq
	}

	// Extract perk_group from annotation or context
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["perk-group"]; ok {
			fm["perk_group"] = v
		}
		if v, ok := ann["perk_group"]; ok {
			fm["perk_group"] = v
		}
	}
	if _, ok := fm["perk_group"]; !ok {
		if pg, ok := ctx.Lookup(section.HeadingLevel, "perk-group"); ok {
			fm["perk_group"] = pg
		}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"perk"},
		ItemID:      id,
	}, nil
}

// extractField looks for a **FieldName:** pattern and returns the value.
func extractField(body, fieldName string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		clean := strings.ReplaceAll(trimmed, "**", "")
		if strings.HasPrefix(clean, fieldName+":") {
			return strings.TrimSpace(strings.TrimPrefix(clean, fieldName+":"))
		}
	}
	return ""
}
