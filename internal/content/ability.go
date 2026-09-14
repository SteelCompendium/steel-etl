package content

import (
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
	"github.com/SteelCompendium/steel-etl/internal/scc"
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

	// Look up parent class/kit/ancestry/treasure from context. When none of
	// those is the nearest recognised ancestor, an ability may instead sit
	// directly under a `rule` section tagged `@group: treasure` (e.g. an
	// armor/weapon/implement enhancement granting a bonus ability, like Imbue
	// Armor's Dragon Soul II granting Dragon's Fire) — that's a
	// treasure-granted ability (SC-323). A rule ancestor with any OTHER group
	// is not "recognised" here and is skipped over, same as before.
	parentID := ""
	parentType := ""
	treasureRuleGroup := ""
	treasureRuleID := ""
	for level := section.HeadingLevel - 1; level >= 1; level-- {
		cur := ctx.Current(level)
		if cur == nil {
			continue
		}
		switch cur["type"] {
		case "class", "kit", "ancestry", "treasure":
			parentID = cur["id"]
			parentType = cur["type"]
		case "rule":
			if cur["group"] == "treasure" && cur["id"] != "" {
				treasureRuleGroup = cur["group"]
				treasureRuleID = cur["id"]
			}
		}
		if parentID != "" || treasureRuleID != "" {
			break
		}
	}

	if parentID != "" {
		fm[parentType] = parentID
	} else if treasureRuleID != "" {
		// Relationship is a frontmatter link, never path nesting (scc-reference
		// "relationships are frontmatter links, never path nesting").
		if book, ok := ctx.Lookup(section.HeadingLevel, "book"); ok && book != "" {
			fm["granted_by"] = scc.Classify(book, []string{"rule", treasureRuleGroup}, treasureRuleID)
		}
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
	} else if treasureRuleID != "" {
		// Treasure-granted abilities are flat under `feature.ability.treasure`,
		// mirroring the flatness of the common bucket below — the granting rule
		// page is carried as the `granted_by` frontmatter link above, not nested
		// into the path (SC-323).
		typePath = append(typePath, "treasure")
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
	triggerRe         = regexp.MustCompile(`\*\*Trigger:\*\*\s*(.+)`)
	// A standalone named-effect line: "**<Label>:** <text>" at the start of a
	// body line. <Label> is any bold label (Effect, Spend N <Resource>,
	// Persistent N, Strained, Special, …) — the label is non-greedy up to the
	// first ":**", so a colon inside the effect text doesn't truncate it.
	namedEffectRe = regexp.MustCompile(`^\*\*([^*]+?):\*\*\s*(.+)$`)
	// A tier-outcome bullet line ("- **≤11:** …"), for extractOrderedEffects
	// (SC-310). Requires the leading bullet marker — unlike tierRe (used only
	// once already inside extractPowerRoll's recognized power-roll block) —
	// so it never misfires on an ordinary "**Label:** text" paragraph.
	orderedTierLineRe = regexp.MustCompile(`^[-*]\s*\*\*([^*]+):\*\*\s*(.+)$`)
)

// classifyOrderedTier maps a tier bullet's bold key text to its effects[]
// field name, mirroring extractPowerRoll's own key matching (≤11/12-16/17+
// and their "11 or lower"/"12–16"/"17 or higher" spellings).
func classifyOrderedTier(key string) (string, bool) {
	switch {
	case strings.Contains(key, "≤11") || strings.Contains(key, "11 or lower"):
		return "tier1", true
	case strings.Contains(key, "12-16") || strings.Contains(key, "12–16"):
		return "tier2", true
	case strings.Contains(key, "17+") || strings.Contains(key, "17 or higher"):
		return "tier3", true
	default:
		return "", false
	}
}

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

	// Extract the ordered effects list — the power-roll entry plus every named
	// effect (Effect, Spend N, Persistent N, Strained, any ability-specific
	// rider), all in document order. Must run AFTER extractPowerRoll so the
	// roll's characteristic/tiers are available. The Trigger line is handled
	// separately as a top-level field, so it is excluded here.
	if _, exists := fm["effects"]; !exists {
		if ordered := extractOrderedEffects(lines, fm); len(ordered) > 0 {
			fm["effects"] = ordered
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

// extractOrderedEffects builds the effects[] list for an ability body in exact
// document order (SC-310, mirroring SC-308's statblock/featureblock model —
// see docs/statblocks.md's "The effects[] list is the authoritative body
// order"): one entry per labeled paragraph (**Effect:**, **Special:**, a
// "Spend N <Resource>:" cost, …) and per bare prose paragraph, plus the
// attachment rule — a tier list (headered "**Power Roll + <Char>:**" or bare)
// attaches to the entry created by the paragraph immediately before it,
// becoming that entry's roll/tier1..3; a tier list with nothing eligible
// before it (right after the spec table, or after an entry that already
// consumed a roll) becomes its own {roll, tier1..3} entry — the pre-existing
// shape for the common single-roll-right-after-the-table case.
//
// The `roll` value is the header text verbatim ("Power Roll + Reason") when
// the source had one, omitted entirely for a header-less list — never a
// derived label (the site derives a header-less panel's display head at
// render time from this SAME entry's own effect text,
// content.DeriveTestLabel).
//
// The Trigger line and the ability's single flavor paragraph (the first
// bare-italic line, already captured separately by extractAbilityFields as
// fm["flavor"]) are excluded here — both are top-level feature.schema.json
// fields, not effects[] entries.
//
// curIdx tracks the entry eligible for the next attach BY INDEX, not a
// pointer, because effects keeps growing via append (SC-308 review round 2,
// finding L-4: a pointer into a growing slice goes stale across a
// reallocation). rollKeysSet tracks which tier keys curIdx has already
// received so a tier list's three bullet lines all land on the SAME entry,
// while a second list immediately following (nothing new to attach to) still
// starts its own standalone entry — mirrors statblock_parse.go's
// parseStatblockEffects exactly (the closest existing model: also a per-line,
// not per-paragraph, walk). lastWasPlainProse folds a run of consecutive
// non-blank, non-special lines with no intervening blank line into ONE bare
// entry — the same mechanism that lets an ordinary (non-tier) bulleted list
// join its preceding lead-in sentence into a single entry (e.g. Judgment's
// "Additionally, you can spend 1 wrath…" + its four-item list) rather than
// spawning one entry per physical line.
func extractOrderedEffects(lines []string, fm map[string]any) []map[string]any {
	var effects []map[string]any
	curIdx := -1
	rollKeysSet := map[string]bool{}
	pendingRoll := ""
	flavorSkipped := false
	lastWasPlainProse := false

	newEntry := func(name, cost, text string) {
		e := map[string]any{}
		if name != "" {
			e["name"] = name
		}
		if cost != "" {
			e["cost"] = cost
		}
		if text != "" {
			e["effect"] = text
		}
		effects = append(effects, e)
		curIdx = len(effects) - 1
		rollKeysSet = map[string]bool{}
	}
	attach := func(roll, key, val string) {
		var target map[string]any
		if curIdx >= 0 && !rollKeysSet[key] {
			target = effects[curIdx]
		} else {
			target = map[string]any{}
			effects = append(effects, target)
			curIdx = len(effects) - 1
			rollKeysSet = map[string]bool{}
		}
		if roll != "" {
			target["roll"] = roll
		}
		target[key] = val
		rollKeysSet[key] = true
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "|") {
			lastWasPlainProse = false
			continue
		}

		// The ability's single flavor paragraph (the first bare-italic line) is
		// a top-level field (fm["flavor"]), not an effects entry — skip exactly
		// the first occurrence, matching extractAbilityFields' own flavor scan.
		if !flavorSkipped && strings.HasPrefix(trimmed, "*") && strings.HasSuffix(trimmed, "*") && !strings.HasPrefix(trimmed, "**") {
			flavorSkipped = true
			lastWasPlainProse = false
			continue
		}

		// Power-roll header → formula pending for the next tier list; not
		// itself a block.
		if m := powerRollHeaderRe.FindStringSubmatch(trimmed); m != nil {
			pendingRoll = "Power Roll + " + strings.TrimSpace(m[1])
			lastWasPlainProse = false
			continue
		}

		// A tier bullet line ("- **≤11:** …") attaches to curIdx per the rule
		// above, or starts a standalone roll entry.
		if tm := orderedTierLineRe.FindStringSubmatch(trimmed); tm != nil {
			if key, ok := classifyOrderedTier(tm[1]); ok {
				attach(pendingRoll, key, strings.TrimSpace(tm[2]))
				pendingRoll = ""
				lastWasPlainProse = false
				continue
			}
		}

		// Labeled paragraph ("**Effect:**", "**Spend N X:**", …) → its own
		// entry, eligible for a following tier list to attach to. Trigger is
		// routed to the top-level `trigger` field (extractAbilityFields), not
		// here — and, like the statblock data path, resets curIdx so a roll
		// that follows a Trigger paragraph doesn't reach back to whatever
		// entry preceded it.
		if m := namedEffectRe.FindStringSubmatch(trimmed); m != nil {
			label := strings.TrimSpace(m[1])
			if label == "Trigger" {
				curIdx = -1
				lastWasPlainProse = false
				continue
			}
			text := strings.TrimSpace(m[2])
			if strings.HasPrefix(label, "Spend ") {
				newEntry("", label, text)
			} else {
				newEntry(label, "", text)
			}
			lastWasPlainProse = false
			continue
		}

		// Bare prose → its own entry too (Divine Dragon's rolls sit under bare
		// prose, not a labeled section), joining a run of consecutive plain
		// lines (no intervening blank line) into the SAME entry.
		if lastWasPlainProse && curIdx >= 0 {
			if s, ok := effects[curIdx]["effect"].(string); ok {
				effects[curIdx]["effect"] = s + " " + trimmed
				continue
			}
		}
		newEntry("", "", trimmed)
		lastWasPlainProse = true
	}
	return effects
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
		if p != "" && !isDashPlaceholder(p) {
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
