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
	body := stripBlockquotePrefix(section.BodySource)

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
	}

	// Auto-extract from body content (only fill in what annotations didn't provide)
	extractAbilityFields(body, fm)

	// Look up parent class/kit from context
	classID := ""
	for level := section.HeadingLevel - 1; level >= 1; level-- {
		cur := ctx.Current(level)
		if cur == nil {
			continue
		}
		if cur["type"] == "class" || cur["type"] == "kit" {
			classID = cur["id"]
			break
		}
	}

	if classID != "" {
		fm["class"] = classID
	}

	// Look up level from context
	levelStr := ""
	if level, ok := ctx.Lookup(section.HeadingLevel, "level"); ok {
		fm["level"] = level
		levelStr = level
	}

	// Build type path: feature.ability.{class}.level-{N}
	typePath := []string{"feature", "ability"}
	if classID != "" {
		typePath = append(typePath, classID)
	} else {
		typePath = append(typePath, "common")
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
	powerRollHeaderRe      = regexp.MustCompile(`\*\*Power Roll \+ (\w+):\*\*`)
	tierRe                 = regexp.MustCompile(`\*\*([^*]+):\*\*\s*(.+)`)
	effectRe               = regexp.MustCompile(`\*\*Effect:\*\*\s*(.+)`)
	spendRe                = regexp.MustCompile(`\*\*Spend\s+(.+?):\*\*\s*(.+)`)
	triggerRe              = regexp.MustCompile(`\*\*Trigger:\*\*\s*(.+)`)
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
