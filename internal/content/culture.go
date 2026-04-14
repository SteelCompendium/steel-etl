package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// CultureParser handles @type: culture sections.
type CultureParser struct{}

func (p *CultureParser) Type() string { return "culture" }

func (p *CultureParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "culture",
	}

	body := section.FullBodySource()

	// Extract structured fields
	if v := extractField(body, "Environment"); v != "" {
		fm["environment"] = v
	}
	if v := extractField(body, "Organization"); v != "" {
		fm["organization"] = v
	}
	if v := extractField(body, "Upbringing"); v != "" {
		fm["upbringing"] = v
	}
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

	// Annotation overrides
	if ann := section.Annotation; ann != nil {
		for _, key := range []string{"environment", "organization", "upbringing", "skill", "language"} {
			if v, ok := ann[key]; ok {
				fm[key] = v
			}
		}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"culture"},
		ItemID:      id,
	}, nil
}
