package content

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

var (
	// "- **EV:** 2" style list fields (ordered, loose).
	terrainFieldRe = regexp.MustCompile(`(?m)^-\s*\*\*([A-Za-z ]+):\*\*\s*(.+)$`)
	// "(Level 2 Hazard Hexer)" trailing classifier: level, terrain type
	// (may be multi-word, e.g. "Siege Engine"), role.
	terrainClassifierRe = regexp.MustCompile(`\(Level\s+(\d+)\s+(.+?)\s+(\w+)\)\s*$`)
	// fallback for headings without the full classifier.
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

	// "(Level 2 Hazard Hexer)" → level / terrain_type / role. The role word is
	// validated against the statblock role vocabulary; an unrecognized
	// classifier falls back to level-only extraction.
	if m := terrainClassifierRe.FindStringSubmatch(section.Heading); m != nil && knownRoles[m[3]] {
		if n, err := strconv.Atoi(m[1]); err == nil {
			fm["level"] = n
		}
		fm["terrain_type"] = strings.TrimSpace(m[2])
		fm["role"] = m[3]
	} else if m := terrainLevelRe.FindStringSubmatch(section.Heading); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			fm["level"] = n
		}
	}

	if flavor := firstFlavorParagraph(body); flavor != "" {
		fm["flavor"] = flavor
	}

	// Ordered loose stat pairs ("EV: 2", "Stamina: 3 per square", …).
	var stats []map[string]any
	for _, m := range terrainFieldRe.FindAllStringSubmatch(body, -1) {
		stats = append(stats, map[string]any{
			"name":  strings.TrimSpace(m[1]),
			"value": strings.TrimSpace(m[2]),
		})
	}
	if len(stats) > 0 {
		fm["stats"] = stats
	}

	if feats := ParseRichFeatures(body); len(feats) > 0 {
		fm["features"] = RichFeatureMaps(feats)
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
