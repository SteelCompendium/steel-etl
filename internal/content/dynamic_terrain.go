package content

import (
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

var (
	// "- **EV:** 2" style list fields.
	terrainFieldRe = regexp.MustCompile(`(?m)^-\s*\*\*([A-Za-z ]+):\*\*\s*(.+)$`)
	// "(Level 2 Hazard Hexer)" trailing classifier in the heading.
	terrainLevelRe = regexp.MustCompile(`Level\s+(\d+)`)
)

// DynamicTerrainParser handles @type: dynamic-terrain sections — terrain objects
// (hazards, fieldworks, mechanisms, fixtures). Classifies as {domain}.{category}/{id}
// where domain defaults to "dynamic-terrain".
type DynamicTerrainParser struct{}

func (p *DynamicTerrainParser) Type() string { return "dynamic-terrain" }

func (p *DynamicTerrainParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)
	name = strings.TrimSpace(trailingParenRe.ReplaceAllString(name, ""))

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	body := section.FullBodySource()
	fm := map[string]any{
		"name": name,
		"type": "dynamic-terrain",
	}

	if m := terrainLevelRe.FindStringSubmatch(section.Heading); m != nil {
		fm["level"] = m[1]
	}
	for _, m := range terrainFieldRe.FindAllStringSubmatch(body, -1) {
		key := strings.ToLower(strings.TrimSpace(m[1]))
		key = strings.ReplaceAll(key, " ", "_")
		fm[key] = strings.TrimSpace(m[2])
	}

	domain := "dynamic-terrain"
	if d, ok := ctx.Lookup(section.HeadingLevel, "domain"); ok && d != "" {
		domain = d
	}
	category, _ := ctx.Lookup(section.HeadingLevel, "category")

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    compactPath(domain, category),
		ItemID:      id,
	}, nil
}
