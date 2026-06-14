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

	if domain == "fixture" {
		// Fixture statblocks become featureblock entities in the
		// monster.fixture.<element>.featureblock family (Plan 5c).
		fm["type"] = "featureblock"
		// Parse role/terrain_type and clear spurious keywords.
		applyFixtureGrid(fm, body)
		// Build stats[] from the 2-col grid (Stamina/Size).
		fm["stats"] = fixtureStats(fm)
		// Features: base (Level-0) features from blockquotes.
		if feats := ParseRichFeatures(body); len(feats) > 0 {
			fm["features"] = RichFeatureMaps(feats)
		}
		return &ParsedContent{
			Frontmatter: fm,
			Body:        body,
			TypePath:    compactPath("monster", "fixture", category, "featureblock"),
			ItemID:      id,
		}, nil
	}

	typePath := compactPath(domain, category, subcategory, "statblock")

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}

// fixtureStats builds the stats[] list from frontmatter fields already populated
// by applyFixtureGrid (stamina and size). The list preserves Stamina-first order
// and omits any absent fields.
func fixtureStats(fm map[string]any) []map[string]any {
	var stats []map[string]any
	if v, ok := fm["stamina"].(string); ok && v != "" {
		stats = append(stats, map[string]any{"name": "Stamina", "value": v})
		delete(fm, "stamina")
	}
	if v, ok := fm["size"].(string); ok && v != "" {
		stats = append(stats, map[string]any{"name": "Size", "value": v})
		delete(fm, "size")
	}
	return stats
}

// MonsterParser handles @type: monster sections — a monster group (e.g.
// "Goblins"). It produces a lore landing page at monster.group/{category}
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
		TypePath:    []string{"monster", "group"},
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

	body := section.FullBodySource()
	heading := CleanHeading(section.Heading)

	fm := map[string]any{
		"name": name,
		"type": "featureblock",
	}

	// Companion advancement-features container (beastheart). When companion
	// context is present, this is the per-species "<C> Advancement Features"
	// block: classify as monster.companion.<class>.advancement-features/<species>
	// and embed the child @type:feature sections (the Level-3/6/10 advancement
	// features, which keep their own feature.* codes) as features[] for the card.
	if companionID, _ := ctx.Lookup(section.HeadingLevel, "companion"); companionID != "" {
		classID := findAncestorID(ctx, section.HeadingLevel, "class")
		if feats := collectChildFeatures(section); len(feats) > 0 {
			fm["features"] = RichFeatureMaps(feats)
		}
		return &ParsedContent{
			Frontmatter: fm,
			Body:        body,
			TypePath:    compactPath("monster", "companion", classID, "advancement-features"),
			ItemID:      companionID,
		}, nil
	}

	// Fixture advancement-features (summoner). When the enclosing monster-group
	// has domain=fixture, this sibling featureblock carries the Level-5/9
	// advancement features for the fixture entity (Plan 5c).
	if domain, category, _ := statblockDomain(ctx, section.HeadingLevel); domain == "fixture" {
		if feats := ParseRichFeatures(body); len(feats) > 0 {
			fm["features"] = RichFeatureMaps(feats)
		}
		return &ParsedContent{
			Frontmatter: fm,
			Body:        body,
			TypePath:    compactPath("monster", "fixture", category, "advancement-features"),
			ItemID:      id,
		}, nil
	}

	// kind: any "malice" mention in the heading marks a malice block ("Basilisk
	// Malice (Malice Features)", "Basic Malice"); everything else is a named
	// feature block ("Tactical Stance (Ajax Feature)").
	if strings.Contains(strings.ToLower(heading), "malice") {
		fm["kind"] = "malice"
	} else {
		fm["kind"] = "feature"
	}

	// level: from level-qualified headings ("… (Level 4+ Malice Features)").
	if m := levelRe.FindStringSubmatch(heading); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			fm["level"] = n
		}
	}

	if flavor := firstFlavorParagraph(body); flavor != "" {
		fm["flavor"] = flavor
	}
	if feats := ParseRichFeatures(body); len(feats) > 0 {
		fm["features"] = RichFeatureMaps(feats)
	}

	domain, category, subcategory := statblockDomain(ctx, section.HeadingLevel)
	typePath := compactPath(domain, category, subcategory)

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}

var (
	// fixture 2-col grid cell: "**Stamina:** 20 + your level"
	fixtureCellRe = regexp.MustCompile(`\*\*([A-Za-z ]+):\*\*\s*([^|]*)`)
	// the fixture's italic classifier line: "*Hazard Support*"
	fixtureRoleRe = regexp.MustCompile(`(?m)^\*([A-Za-z ]+)\*\s*$`)
)

// applyFixtureGrid parses the summoner-fixture statblock header — a 2-column
// "| **Stamina:** … | **Size:** … |" grid plus an italic "*Hazard Support*"
// role line — which the standard parseStatGrid does not understand
// (workspace FOLLOWUPS #6). It also removes the garbage keywords the standard
// header parse derives from the first grid cell.
func applyFixtureGrid(fm map[string]any, body string) {
	delete(fm, "keywords")

	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "|") || strings.Contains(t, "---") {
			continue
		}
		for _, m := range fixtureCellRe.FindAllStringSubmatch(t, -1) {
			key := strings.ToLower(strings.TrimSpace(m[1]))
			val := linkDisplay(strings.TrimSpace(m[2]))
			if (key == "stamina" || key == "size") && val != "" {
				fm[key] = val
			}
		}
	}

	if m := fixtureRoleRe.FindStringSubmatch(body); m != nil {
		words := strings.Fields(strings.TrimSpace(m[1]))
		if len(words) >= 2 {
			role := words[len(words)-1]
			if knownRoles[role] {
				fm["role"] = role
				fm["terrain_type"] = strings.Join(words[:len(words)-1], " ")
			}
		}
	}
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

// collectChildFeatures returns the @type:feature descendants of a section as
// RichFeatures (name + prose body + level), in document order, for embedding in a
// featureblock's features[]. Used by the companion advancement-features block,
// whose Level-3/6/10 members are plain prose features. Each member keeps its own
// SCC code; this embed is render-only.
func collectChildFeatures(section *parser.Section) []RichFeature {
	var out []RichFeature
	for _, child := range section.Children {
		switch child.Type() {
		case "feature":
			rf := RichFeature{
				Name: CleanHeading(child.Heading),
				Body: strings.TrimSpace(child.FullBodySource()),
			}
			if lv, ok := child.Annotation["level"]; ok {
				if n, err := strconv.Atoi(lv); err == nil {
					rf.Level = n
				}
			}
			out = append(out, rf)
		case "":
			out = append(out, collectChildFeatures(child)...)
		}
	}
	return out
}
