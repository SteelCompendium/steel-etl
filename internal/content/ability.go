package content

import (
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// AbilityParser handles @type: ability sections.
type AbilityParser struct{}

func (p *AbilityParser) Type() string { return "ability" }

func (p *AbilityParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	// Clean the heading: strip cost suffix like "(11 Piety)"
	cleanName := CleanHeading(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(cleanName)
	}

	// Strip blockquote prefix ("> ") from body lines
	body := stripBlockquotePrefix(section.FullBodySource())

	fm := map[string]any{
		"name": cleanName,
		"type": "ability",
	}

	// Extract from annotation (explicit overrides)
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["cost"]; ok {
			fm["cost"] = v
		}
		if v, ok := ann["subtype"]; ok {
			fm["subtype"] = v
		}
		if v, ok := ann["action"]; ok {
			fm["action_type"] = v
		}
		if v, ok := ann["distance"]; ok {
			fm["distance"] = v
		}
		if v, ok := ann["target"]; ok {
			fm["target"] = v
		}
		if v, ok := ann["keywords"]; ok {
			fm["keywords"] = parseKeywords(v)
		}
		if v, ok := ann["trigger"]; ok {
			fm["trigger"] = v
		}
		// Subclass (e.g. beastheart Wild Nature) is reference metadata only — it is
		// surfaced as a frontmatter field and never alters the SCC path.
		if v, ok := ann["subclass"]; ok && v != "" {
			fm["subclass"] = parseSubclass(v)
		}
	}

	// Auto-extract from body content (only fill in what annotations didn't provide)
	extractAbilityFields(body, fm)

	// Look up parent class/kit/ancestry/treasure from context
	parentID := ""
	parentType := ""
	for level := section.HeadingLevel - 1; level >= 1; level-- {
		cur := ctx.Current(level)
		if cur == nil {
			continue
		}
		switch cur["type"] {
		case "class", "kit", "ancestry", "treasure":
			parentID = cur["id"]
			parentType = cur["type"]
		}
		if parentID != "" {
			break
		}
	}

	if parentID != "" {
		fm[parentType] = parentID
	}

	// Look up level from context
	levelStr := ""
	if level, ok := ctx.Lookup(section.HeadingLevel, "level"); ok {
		fm["level"] = level
		levelStr = level
	}

	// Look up companion species from context (beastheart book).
	companionID, _ := ctx.Lookup(section.HeadingLevel, "companion")
	if companionID != "" {
		fm["companion"] = companionID
	}

	if fs := featureSource(ctx, section); fs != "" {
		fm["feature_source"] = fs
	}

	// Build type path: feature.ability.{parent}.level-{N}
	// Companion abilities use feature.ability.companion.beastheart.{species}.level-{N}
	// (the class segment mirrors FeatureParser's companion path; empty-class guard prevents
	// a double-dot path when no class ancestor is present).
	// `ability` is the marked rigorous specialization in the hub-and-spoke feature
	// taxonomy (see docs/superpowers/specs/2026-06-07-feature-taxonomy-design.md);
	// plain features (feature.go) carry no kind segment.
	classID := findAncestorID(ctx, section.HeadingLevel, "class")
	typePath := []string{"feature", "ability"}
	if companionID != "" {
		// feature.ability.companion.beastheart.wolf.level-N/<id> — mirror the
		// FeatureParser companion path (empty-class guard).
		typePath = append(typePath, "companion")
		if classID != "" {
			typePath = append(typePath, classID)
		}
		typePath = append(typePath, companionID)
	} else if parentID != "" {
		typePath = append(typePath, parentID)
	} else {
		// Common abilities are flat under `feature.ability.common` regardless of
		// any feature-group ancestor (the Combat chapter's "Maneuvers" /
		// "Free Strikes" groups): we don't sub-group common abilities the way
		// class trees do, so a maneuver/free-strike ability lives directly under
		// common, not feature.ability.common.<group> (FOLLOWUPS #17).
		typePath = append(typePath, "common")
	}
	// Named feature-group grouping under a class (mirrors FeatureParser): an
	// ability sitting directly inside a named feature-group — class-scoped, no
	// level — takes the group id as a path segment (e.g. the fury's "Aspect of
	// the Wild" under "Stormwight Kits"). Kit-scoped signature abilities use
	// parentType=="kit" and are unaffected.
	if parentType == "class" && levelStr == "" {
		if groupID := findAncestorID(ctx, section.HeadingLevel, "feature-group"); groupID != "" {
			typePath = append(typePath, groupID)
		}
	}
	if levelStr != "" {
		typePath = append(typePath, "level-"+levelStr)
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}

// stripBlockquotePrefix removes "> " from the start of each line.
func stripBlockquotePrefix(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "> ") {
			lines[i] = line[2:]
		} else if line == ">" {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

var (
	// Matches the 2x2 ability table (keywords/action on row 1, distance/target on row 2)
	abilityTableKeywordsRe = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	// The SCC linking sweep wraps the header as "**[Power Roll](scc:…) + <chars>:**",
	// where <chars> may be multi-characteristic and link-wrapped
	// ("[Might](scc:…) or [Agility](scc:…)"). Accept bare or link-wrapped "Power
	// Roll", and capture the full characteristics expression verbatim (links kept,
	// like the sibling effect/distance fields). Non-greedy up to the closing ":**"
	// — scc: URLs contain ":" but never ":**", so the first ":**" is the real end.
	powerRollHeaderRe = regexp.MustCompile(`\*\*(?:\[Power Roll\]\([^)]*\)|Power Roll)\s*\+\s*(.+?):\*\*`)
	tierRe            = regexp.MustCompile(`\*\*([^*]+):\*\*\s*(.+)`)
	effectRe          = regexp.MustCompile(`\*\*Effect:\*\*\s*(.+)`)
	spendRe           = regexp.MustCompile(`\*\*Spend\s+(.+?):\*\*\s*(.+)`)
	triggerRe         = regexp.MustCompile(`\*\*Trigger:\*\*\s*(.+)`)
)

// extractAbilityFields parses the body text to extract structured fields.
// Only fills in fields that aren't already set (annotation overrides take precedence).
func extractAbilityFields(body string, fm map[string]any) {
	lines := strings.Split(body, "\n")

	// Extract flavor text (first italic paragraph)
	if _, exists := fm["flavor"]; !exists {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "*") && strings.HasSuffix(trimmed, "*") && !strings.HasPrefix(trimmed, "**") {
				fm["flavor"] = strings.Trim(trimmed, "*")
				break
			}
		}
	}

	// Extract from ability table (2x2 pattern)
	extractAbilityTable(lines, fm)

	// Extract power roll
	extractPowerRoll(lines, fm)

	// Extract effect
	if _, exists := fm["effect"]; !exists {
		for _, line := range lines {
			matches := effectRe.FindStringSubmatch(strings.TrimSpace(line))
			if matches != nil {
				fm["effect"] = strings.TrimSpace(matches[1])
				break
			}
		}
	}

	// Extract spend
	if _, exists := fm["spend"]; !exists {
		for _, line := range lines {
			matches := spendRe.FindStringSubmatch(strings.TrimSpace(line))
			if matches != nil {
				fm["spend"] = strings.TrimSpace(matches[1]) + ": " + strings.TrimSpace(matches[2])
				break
			}
		}
	}

	// Extract trigger
	if _, exists := fm["trigger"]; !exists {
		for _, line := range lines {
			matches := triggerRe.FindStringSubmatch(strings.TrimSpace(line))
			if matches != nil {
				fm["trigger"] = strings.TrimSpace(matches[1])
				break
			}
		}
	}
}

// extractAbilityTable parses the 2x2 keyword/action/distance/target table.
func extractAbilityTable(lines []string, fm map[string]any) {
	// Find table rows (lines starting with "|")
	var tableRows []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && !strings.Contains(trimmed, "---") {
			tableRows = append(tableRows, trimmed)
		}
	}

	if len(tableRows) < 2 {
		return
	}

	// Row 1: keywords | action type
	row1Cells := splitTableRow(tableRows[0])
	if len(row1Cells) >= 2 {
		if _, exists := fm["keywords"]; !exists {
			kw := extractBoldText(row1Cells[0])
			fm["keywords"] = parseKeywords(kw)
		}
		if _, exists := fm["action_type"]; !exists {
			fm["action_type"] = extractBoldText(row1Cells[1])
		}
	}

	// Row 2: distance | target
	row2Cells := splitTableRow(tableRows[1])
	if len(row2Cells) >= 2 {
		if _, exists := fm["distance"]; !exists {
			d := extractBoldText(row2Cells[0])
			// Strip emoji prefixes (📏)
			d = stripEmoji(d)
			fm["distance"] = strings.TrimSpace(d)
		}
		if _, exists := fm["target"]; !exists {
			t := extractBoldText(row2Cells[1])
			// Strip emoji prefixes (🎯)
			t = stripEmoji(t)
			fm["target"] = strings.TrimSpace(t)
		}
	}
}

// extractPowerRoll parses the power roll section.
func extractPowerRoll(lines []string, fm map[string]any) {
	inPowerRoll := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inPowerRoll {
			matches := powerRollHeaderRe.FindStringSubmatch(trimmed)
			if matches != nil {
				if _, exists := fm["power_roll_characteristic"]; !exists {
					fm["power_roll_characteristic"] = matches[1]
				}
				inPowerRoll = true
			}
			continue
		}

		// Parse tier lines
		matches := tierRe.FindStringSubmatch(trimmed)
		if matches == nil {
			if trimmed == "" {
				continue
			}
			break // end of power roll section
		}

		tierKey := matches[1]
		tierVal := strings.TrimSpace(matches[2])

		if strings.Contains(tierKey, "≤11") || strings.Contains(tierKey, "11 or lower") {
			if _, exists := fm["tier1"]; !exists {
				fm["tier1"] = tierVal
			}
		} else if strings.Contains(tierKey, "12-16") || strings.Contains(tierKey, "12–16") {
			if _, exists := fm["tier2"]; !exists {
				fm["tier2"] = tierVal
			}
		} else if strings.Contains(tierKey, "17+") || strings.Contains(tierKey, "17 or higher") {
			if _, exists := fm["tier3"]; !exists {
				fm["tier3"] = tierVal
			}
		}
	}
}

// splitTableRow splits a markdown table row by "|" and returns trimmed cells.
func splitTableRow(row string) []string {
	row = strings.Trim(row, "|")
	parts := strings.Split(row, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// extractBoldText extracts text from **bold** markers.
func extractBoldText(s string) string {
	matches := abilityTableKeywordsRe.FindAllStringSubmatch(s, -1)
	if len(matches) > 0 {
		var parts []string
		for _, m := range matches {
			parts = append(parts, m[1])
		}
		return strings.Join(parts, ", ")
	}
	return strings.TrimSpace(s)
}

// parseSubclass converts a @subclass annotation value into frontmatter form:
// a single value stays a string; comma-separated values become a []string so
// that features/abilities shared by multiple subclasses are represented cleanly.
func parseSubclass(s string) any {
	parts := parseKeywords(s)
	if len(parts) == 1 {
		return parts[0]
	}
	return parts
}

// parseKeywords splits a comma-separated keyword string into a list.
func parseKeywords(s string) []string {
	parts := strings.Split(s, ",")
	var keywords []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			keywords = append(keywords, p)
		}
	}
	return keywords
}

var emojiRe = regexp.MustCompile(`[\x{1F300}-\x{1F9FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}]\s*`)

// stripEmoji removes common emoji characters from a string.
func stripEmoji(s string) string {
	return emojiRe.ReplaceAllString(s, "")
}
