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

	if f := firstFlavorParagraph(body); f != "" {
		fm["flavor"] = f
	}

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
//
// The label itself may be SCC link-swept — e.g. a kit's "**[Speed](…) [Bonus](…):**
// +1" or a class's "**Average [Potency](…):** [Presence](…) − 1" — so the label is
// matched by stripping its markdown links, not by a literal compare. Without this,
// link-swept labels silently stopped matching and the field dropped out (every
// Browse/kit card showed 0/—; class potency / career renown went missing too).
// The label↔value boundary is the first colon that is *not* inside a markdown link
// target — link targets contain their own colons (`scc.v1:…`) — so the returned
// value keeps any links it carries (class potency, treasure effect, perk
// prerequisite all reference SCC codes that must survive into the data output).
func extractField(body, fieldName string) string {
	for _, line := range strings.Split(body, "\n") {
		clean := strings.TrimSpace(strings.ReplaceAll(line, "**", ""))
		colon := labelColonIndex(clean)
		if colon < 0 {
			continue
		}
		if stripInlineMarkdown(clean[:colon]) == fieldName {
			return strings.TrimSpace(clean[colon+1:])
		}
	}
	return ""
}

// labelColonIndex returns the index of the first ':' in s that sits outside a
// markdown link target '(…)', i.e. the colon that terminates a "Label:" field
// prefix. Colons inside link targets (e.g. the "scc.v1:" in "[Speed](scc.v1:…)")
// are skipped. Returns -1 when there is no such colon.
func labelColonIndex(s string) int {
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
