package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ComplicationParser handles @type: complication sections.
type ComplicationParser struct{}

func (p *ComplicationParser) Type() string { return "complication" }

func (p *ComplicationParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "complication",
	}

	body := section.FullBodySource()

	// Extract structured fields
	if v := extractField(body, "Benefit"); v != "" {
		fm["benefit"] = v
	}
	if v := extractField(body, "Drawback"); v != "" {
		fm["drawback"] = v
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"complication"},
		ItemID:      id,
	}, nil
}
