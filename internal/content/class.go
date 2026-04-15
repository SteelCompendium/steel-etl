package content

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ClassParser handles @type: class sections.
type ClassParser struct{}

func (p *ClassParser) Type() string { return "class" }

func (p *ClassParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "class",
	}

	body := section.FullBodySource()

	// Extract heroic resource from body
	if hr := extractHeroicResource(body); hr != "" {
		fm["heroic_resource"] = hr
	}

	// Extract primary characteristics
	if v := extractField(body, "Primary Characteristics"); v != "" {
		fm["primary_characteristics"] = splitCommaList(v)
	}

	// Extract potency fields
	if v := extractField(body, "Weak Potency"); v != "" {
		fm["weak_potency"] = v
	}
	if v := extractField(body, "Average Potency"); v != "" {
		fm["average_potency"] = v
	}
	if v := extractField(body, "Strong Potency"); v != "" {
		fm["strong_potency"] = v
	}

	// Extract skills
	if v := extractField(body, "Skill"); v != "" {
		fm["skills"] = splitCommaList(v)
	} else if v := extractField(body, "Skills"); v != "" {
		fm["skills"] = splitCommaList(v)
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"class"},
		ItemID:      id,
	}, nil
}

// extractHeroicResource looks for "Heroic Resource: X" in the body.
func extractHeroicResource(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		// Match **Heroic Resource: Ferocity** or similar
		line = strings.ReplaceAll(line, "**", "")
		if strings.HasPrefix(line, "Heroic Resource:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Heroic Resource:"))
		}
	}
	return ""
}
