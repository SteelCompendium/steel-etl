package content

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// AncestryParser handles @type: ancestry sections.
type AncestryParser struct{}

func (p *AncestryParser) Type() string { return "ancestry" }

func (p *AncestryParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "ancestry",
	}

	body := section.FullBodySource()

	// Extract signature trait name. Source uses two formats:
	//   1. Inline: **Signature Trait:** Name
	//   2. Heading: #### Signature Trait: Name (inside child "Traits" section)
	if v := extractField(body, "Signature Trait"); v != "" {
		fm["signature_trait_name"] = v
	} else if name := findSignatureTraitHeading(section); name != "" {
		fm["signature_trait_name"] = name
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"ancestry"},
		ItemID:      id,
	}, nil
}

// findSignatureTraitHeading searches all descendant sections for a heading
// matching "Signature Trait: <Name>" and returns the name portion.
func findSignatureTraitHeading(section *parser.Section) string {
	for _, s := range section.AllSections() {
		if strings.HasPrefix(s.Heading, "Signature Trait:") {
			name := strings.TrimSpace(strings.TrimPrefix(s.Heading, "Signature Trait:"))
			if name != "" {
				return name
			}
		}
	}
	return ""
}
