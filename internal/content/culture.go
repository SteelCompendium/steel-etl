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

	// Skill options: wrap as single-element array (source text is natural language)
	if v := extractField(body, "Skill"); v != "" {
		fm["skill_options"] = []string{v}
	} else if v := extractField(body, "Skills"); v != "" {
		fm["skill_options"] = []string{v}
	}

	if v := extractField(body, "Language"); v != "" {
		fm["language"] = v
	} else if v := extractField(body, "Languages"); v != "" {
		fm["language"] = v
	}

	// Annotation overrides
	if ann := section.Annotation; ann != nil {
		for _, key := range []string{"environment", "organization", "upbringing", "language"} {
			if v, ok := ann[key]; ok {
				fm[key] = v
			}
		}
		// skill annotation → skill_options array
		if v, ok := ann["skill"]; ok {
			fm["skill_options"] = []string{v}
		}
		if v, ok := ann["skill_options"]; ok {
			fm["skill_options"] = splitCommaList(v)
		}
		if v, ok := ann["quick_build_skill"]; ok {
			fm["quick_build_skill"] = v
		}
		if v, ok := ann["culture_benefit_type"]; ok {
			fm["culture_benefit_type"] = v
		}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"culture"},
		ItemID:      id,
	}, nil
}
