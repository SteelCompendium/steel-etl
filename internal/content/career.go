package content

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// CareerParser handles @type: career sections.
type CareerParser struct{}

func (p *CareerParser) Type() string { return "career" }

func (p *CareerParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "career",
	}

	body := section.FullBodySource()

	// Extract structured fields from body
	if v := extractField(body, "Skill"); v != "" {
		fm["skill"] = v
	} else if v := extractField(body, "Skills"); v != "" {
		fm["skill"] = v
	}
	if v := extractField(body, "Language"); v != "" {
		fm["language"] = v
	} else if v := extractField(body, "Languages"); v != "" {
		fm["language"] = v
	}
	if v := extractField(body, "Renown"); v != "" {
		fm["renown"] = v
	}
	if v := extractField(body, "Wealth"); v != "" {
		fm["wealth"] = v
	}
	if v := extractField(body, "Project Points"); v != "" {
		fm["project_points"] = v
	}
	if v := extractField(body, "Perk"); v != "" {
		fm["perk"] = v
	}

	// Check annotations for explicit overrides
	if ann := section.Annotation; ann != nil {
		for _, key := range []string{"skill", "language", "renown", "wealth", "perk"} {
			if v, ok := ann[key]; ok {
				fm[key] = v
			}
		}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"career"},
		ItemID:      id,
	}, nil
}

// extractListField looks for lines starting with "- " after a field header.
func extractListField(body, fieldName string) []string {
	lines := strings.Split(body, "\n")
	var result []string
	inField := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		clean := strings.ReplaceAll(trimmed, "**", "")

		if strings.HasPrefix(clean, fieldName+":") {
			inField = true
			// Check for inline value
			val := strings.TrimSpace(strings.TrimPrefix(clean, fieldName+":"))
			if val != "" {
				result = append(result, val)
				inField = false
			}
			continue
		}

		if inField {
			if strings.HasPrefix(trimmed, "- ") {
				result = append(result, strings.TrimPrefix(trimmed, "- "))
			} else if trimmed == "" {
				continue
			} else {
				break
			}
		}
	}
	return result
}
