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

	// Extract heroic resource from body
	if hr := extractHeroicResource(section.BodySource); hr != "" {
		fm["heroic_resource"] = hr
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.BodySource,
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
