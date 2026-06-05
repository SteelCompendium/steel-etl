package content

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

var trailingParenRe = regexp.MustCompile(`\s*\([^)]*\)\s*$`)

// featureblockName drops a trailing descriptor parenthetical from a featureblock
// heading UNLESS it contains a digit. So "Goblin Malice (Malice Features)" →
// "Goblin Malice" and "Tactical Stance (Ajax Feature)" → "Tactical Stance", while
// "Demon Malice (Level 1+ Malice Features)" keeps its level qualifier so the
// Level 1/4/7/10 blocks stay distinct.
func featureblockName(heading string) string {
	name := CleanHeading(heading)
	if paren := trailingParenRe.FindString(name); paren != "" && !strings.ContainsAny(paren, "0123456789") {
		name = strings.TrimSpace(trailingParenRe.ReplaceAllString(name, ""))
	}
	return name
}

// statblockDomain returns the SCC domain root ("monster" by default), the
// category slug, and an optional subcategory (e.g. echelon) from the surrounding
// context, set by an enclosing MonsterParser group or monster-group container.
func statblockDomain(ctx *context.ContextStack, level int) (domain, category, subcategory string) {
	domain = "monster"
	if d, ok := ctx.Lookup(level, "domain"); ok && d != "" {
		domain = d
	}
	category, _ = ctx.Lookup(level, "category")
	subcategory, _ = ctx.Lookup(level, "subcategory")
	return domain, category, subcategory
}

// compactPath drops empty segments from a type path.
func compactPath(parts ...string) []string {
	var out []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// intField parses a numeric stat value ("+2", "-2", "0") into an int.
func intField(s string) (int, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "+", ""))
	if s == "" || s == "-" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// StatblockParser handles @type: statblock sections — individual creature stat
// blocks. Classifies as {domain}.{category}.statblock/{id}.
type StatblockParser struct{}

func (p *StatblockParser) Type() string { return "statblock" }

func (p *StatblockParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)
	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	body := section.FullBodySource()
	grid := parseStatGrid(body)

	fm := map[string]any{
		"name": name,
		"type": "statblock",
	}
	if grid.header.level > 0 {
		fm["level"] = grid.header.level
	}
	if grid.header.role != "" {
		fm["role"] = grid.header.role
	}
	if grid.header.organization != "" {
		fm["organization"] = grid.header.organization
	}
	if len(grid.header.keywords) > 0 {
		fm["keywords"] = grid.header.keywords
	}
	if grid.header.ev != "" {
		fm["ev"] = grid.header.ev
	}

	// String labels.
	for label, key := range map[string]string{
		"Stamina": "stamina", "Size": "size", "Movement": "movement",
	} {
		if v, ok := grid.labels[label]; ok && v != "-" {
			fm[key] = v
		}
	}
	// Integer labels.
	for label, key := range map[string]string{
		"Speed": "speed", "Stability": "stability", "Free Strike": "free_strike",
		"Might": "might", "Agility": "agility", "Reason": "reason",
		"Intuition": "intuition", "Presence": "presence",
	} {
		if n, ok := intField(grid.labels[label]); ok {
			fm[key] = n
		}
	}
	// Immunity / Weakness become arrays (split on comma).
	if v, ok := grid.labels["Immunity"]; ok && v != "-" {
		fm["immunities"] = splitCommaList(v)
	}
	if v, ok := grid.labels["Weakness"]; ok && v != "-" {
		fm["weaknesses"] = splitCommaList(v)
	}
	if v, ok := grid.labels["With Captain"]; ok && v != "-" {
		fm["with_captain"] = v
	}

	domain, category, subcategory := statblockDomain(ctx, section.HeadingLevel)
	typePath := compactPath(domain, category, subcategory, "statblock")

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}

// MonsterParser handles @type: monster sections — a monster group (e.g.
// "Goblins"). It produces a lore landing page at monster/{category}/{category}
// AND seeds the `category` (and optional `domain`) context the pipeline pushes
// for its descendant statblocks and featureblocks.
type MonsterParser struct{}

func (p *MonsterParser) Type() string { return "monster" }

func (p *MonsterParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)

	category := ""
	if section.Annotation != nil {
		category = section.Annotation["category"]
	}
	if category == "" {
		category = Slugify(name)
	}

	fm := map[string]any{
		"name":     name,
		"type":     "monster",
		"category": category,
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    []string{"monster", category},
		ItemID:      category,
	}, nil
}

// FeatureblockParser handles @type: featureblock sections — a named block of
// malice/tactical features attached to a monster group (e.g. "Goblin Malice").
// Classifies as {domain}.{category}/{id}, a sibling of the statblock/ folder.
type FeatureblockParser struct{}

func (p *FeatureblockParser) Type() string { return "featureblock" }

func (p *FeatureblockParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := featureblockName(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	fm := map[string]any{
		"name": name,
		"type": "featureblock",
	}

	domain, category, subcategory := statblockDomain(ctx, section.HeadingLevel)
	typePath := compactPath(domain, category, subcategory)

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}

// MonsterGroupParser handles @type: monster-group — a non-code-producing
// container (like feature-group/treasure-group) that seeds `domain` and
// `category` context for descendant statblocks/terrain. Produces no file.
type MonsterGroupParser struct{}

func (p *MonsterGroupParser) Type() string { return "monster-group" }

func (p *MonsterGroupParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	fm := map[string]any{
		"name": CleanHeading(section.Heading),
		"type": "monster-group",
	}
	if section.Annotation != nil {
		for _, k := range []string{"domain", "category", "subcategory"} {
			if v, ok := section.Annotation[k]; ok {
				fm[k] = v
			}
		}
	}
	return &ParsedContent{Frontmatter: fm, Body: section.FullBodySource()}, nil
}
