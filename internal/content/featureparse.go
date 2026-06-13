package content

import (
	"regexp"
	"strconv"
	"strings"
)

// RichFeature is the non-lossy feature shape shared by featureblocks, dynamic
// terrain, and fixture statblocks (spec:
// docs/superpowers/specs/2026-06-12-featureblock-cards-design.md §2). Unlike
// the SDK statblock feature shape (ParseStatblockFeatures), it keeps labeled
// Effect/Trigger sections, cost enhancements, trailing notes, and the source
// emoji icon. Link markdown in text fields is kept verbatim (the data-field
// convention); only the power-roll formula is link-stripped (cosmetic).
type RichFeature struct {
	Icon         string
	Name         string
	Cost         string // "7 Malice", "Signature", "Villain Action 1", …
	Usage        string // "Main action", "Maneuver", … (from the spec table)
	Keywords     []string
	Distance     string
	Target       string
	PowerRoll    *RichPowerRoll
	Sections     []RichSection     // labeled paragraphs: Effect / Trigger / Special …
	Enhancements []RichEnhancement // cost-labeled paragraphs: "2 Malice:" / "Spend …:"
	Body         string            // prose of a table-less (passive) feature
	Trailing     string            // prose after the structured parts of an ability
	Level        int               // advancement group level ("Level 5 Fixture Advancement Feature")
}

// RichPowerRoll is a power roll: Formula "+ 2" (labeled form) or "2d10 + R"
// (dice-in-title form); "" means a bare test result (renderer omits the head).
type RichPowerRoll struct {
	Formula string
	Tiers   map[string]string // keys: low / mid / high
}

type RichSection struct{ Label, Text string }
type RichEnhancement struct{ Cost, Text string }

var (
	fbParaSplitRe = regexp.MustCompile(`\n[ \t]*\n`)
	// power-roll header, tolerant of a link-wrapped "Power Roll" label
	// (mirrors internal/site prHeadRe).
	fbPRHeadRe = regexp.MustCompile(`(?s)^\*\*(?:\[Power Roll\]\([^)]*\)|Power Roll)\s*\+\s*(.+?):\*\*\s*$`)
	// a labeled paragraph: "**Effect:** text…" (mirrors internal/site labelRe).
	fbLabelRe = regexp.MustCompile(`(?s)^\*\*([^*:]+):\*\*\s*(.+)$`)
	// a label that is a cost ("2 Malice", "5+ Malice", "Spend …").
	fbCostLabelRe = regexp.MustCompile(`(?i)^(?:\d+\+?\s+\S+.*|spend\b.*)$`)
	// a standalone bold level-group label inside a blockquote:
	// "**Level 5 Fixture Advancement Feature**".
	fbLevelLabelRe = regexp.MustCompile(`^\*\*Level\s+(\d+)\b[^*]*\*\*$`)
	fbCollapseRe   = regexp.MustCompile(`\s*\n\s*`)
)

// fbCollapse joins a multi-line paragraph into one line.
func fbCollapse(s string) string {
	return strings.TrimSpace(fbCollapseRe.ReplaceAllString(s, " "))
}

// parenToCost maps a title parenthetical to its cost label: the canonical
// "Signature" for "Signature Ability", everything else verbatim ("7 Malice").
func parenToCost(paren string) string {
	if strings.EqualFold(paren, "Signature Ability") {
		return "Signature"
	}
	return paren
}

// ParseRichFeatures parses a body's feature blockquotes into RichFeatures.
// A standalone bold "Level N …" block sets the Level carried by all features
// that follow it (the fixture-advancement form).
func ParseRichFeatures(body string) []RichFeature {
	var out []RichFeature
	level := 0
	for _, block := range splitBlockquoteBlocks(body) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if m := fbLevelLabelRe.FindStringSubmatch(block); m != nil {
			level, _ = strconv.Atoi(m[1])
			continue
		}
		if f, ok := parseRichFeature(block); ok {
			f.Level = level
			out = append(out, f)
		}
	}
	return out
}

// parseRichFeature parses one feature blockquote. Ported from the statblock
// island parser (internal/site/statblock_page.go parseStatblockIslandFeature);
// Plan 2 of the featureblock effort swaps the island onto this shared copy.
func parseRichFeature(block string) (RichFeature, bool) {
	paras := fbParaSplitRe.Split(block, -1)
	if len(paras) == 0 {
		return RichFeature{}, false
	}
	tm := sbTitleRe.FindStringSubmatch(strings.TrimSpace(paras[0]))
	if tm == nil {
		return RichFeature{}, false
	}
	f := RichFeature{Icon: strings.TrimSpace(tm[1]), Name: strings.TrimSpace(tm[2])}

	// Dice-in-title power roll (summoner signatures) is checked BEFORE the
	// parenthetical-cost strip: a link-wrapped characteristic ([R](scc:…)) holds
	// "(...)" that sbParenRe would misread as a cost. linkDisplay collapses that
	// link to bare text first; any parenthetical THEN remaining on the formula is
	// a genuine cost — real titles read "Name 2d10 + R (Signature Ability)".
	diceFormula := ""
	if dm := sbDiceRe.FindStringSubmatch(f.Name); dm != nil {
		f.Name = strings.TrimSpace(dm[1])
		diceFormula = linkDisplay(strings.TrimSpace(dm[2]))
		if pm := sbParenRe.FindStringSubmatch(diceFormula); pm != nil {
			diceFormula = strings.TrimSpace(pm[1])
			f.Cost = parenToCost(strings.TrimSpace(pm[2]))
		}
	}

	// Parenthetical → Signature / cost / Villain Action N (non-dice titles).
	if diceFormula == "" {
		if pm := sbParenRe.FindStringSubmatch(f.Name); pm != nil {
			f.Name = strings.TrimSpace(pm[1])
			f.Cost = parenToCost(strings.TrimSpace(pm[2]))
		}
	}

	var (
		tableSeen bool
		formula   = diceFormula
		tiers     [3]string
		tiersSeen bool
		bareIdx   int
		prose     []string
	)

	for _, para := range paras[1:] {
		tp := strings.TrimSpace(para)
		if tp == "" {
			continue
		}

		// Spec table → keywords / usage (row 1), distance / target (row 2).
		if strings.HasPrefix(tp, "|") {
			rows := featureTableRows(strings.Split(para, "\n"))
			if len(rows) >= 1 {
				f.Keywords = splitCommaList(stripBold(rows[0][0]))
				f.Usage = stripBold(rows[0][1])
			}
			if len(rows) >= 2 {
				f.Distance = cleanIconCell(rows[1][0])
				f.Target = cleanIconCell(rows[1][1])
			}
			tableSeen = true
			continue
		}

		// Power-roll header → formula ("+ 2"); the next list holds the tiers.
		if m := fbPRHeadRe.FindStringSubmatch(tp); m != nil {
			formula = "+ " + linkDisplay(strings.TrimSpace(m[1]))
			continue
		}

		// Labeled tier list ("- **≤11:** …").
		if fbLooksLikeTiers(tp) {
			fbParseTiers(tp, &tiers)
			tiersSeen = true
			continue
		}

		// Dice-in-title abilities: bare digit-led lines are tiers by position.
		if diceFormula != "" && bareIdx < 3 && sbBareTierRe.MatchString(tp) {
			tiers[bareIdx] = fbCollapse(tp)
			bareIdx++
			tiersSeen = true
			continue
		}

		// Labeled paragraph → cost enhancement or titled section.
		if m := fbLabelRe.FindStringSubmatch(tp); m != nil {
			label := strings.TrimSpace(m[1])
			text := fbCollapse(m[2])
			if fbCostLabelRe.MatchString(label) {
				f.Enhancements = append(f.Enhancements, RichEnhancement{Cost: label, Text: text})
			} else {
				f.Sections = append(f.Sections, RichSection{Label: label, Text: text})
			}
			continue
		}

		prose = append(prose, fbCollapse(tp))
	}

	if tiersSeen {
		t := map[string]string{}
		for i, key := range []string{"low", "mid", "high"} {
			if tiers[i] != "" {
				t[key] = tiers[i]
			}
		}
		f.PowerRoll = &RichPowerRoll{Formula: formula, Tiers: t}
	}
	if tableSeen {
		f.Trailing = strings.Join(prose, " ")
	} else if len(prose) > 0 {
		f.Body = strings.Join(prose, "\n\n")
	}
	return f, true
}

// fbLooksLikeTiers reports whether a paragraph is a labeled tier list.
func fbLooksLikeTiers(para string) bool {
	return sbTierRe.MatchString(strings.TrimSpace(strings.Split(para, "\n")[0]))
}

// fbParseTiers fills tiers[0..2] (low/mid/high) from "- **≤11:** …" lines.
func fbParseTiers(para string, tiers *[3]string) {
	for _, line := range strings.Split(para, "\n") {
		m := sbTierRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		switch {
		case strings.HasPrefix(m[1], "≤"):
			tiers[0] = strings.TrimSpace(m[2])
		case strings.Contains(m[1], "-"):
			tiers[1] = strings.TrimSpace(m[2])
		case strings.HasSuffix(m[1], "+"):
			tiers[2] = strings.TrimSpace(m[2])
		}
	}
}

// ToMap converts a RichFeature to the featureblock.schema.json features[]
// shape (snake_case keys, empty fields omitted).
func (f RichFeature) ToMap() map[string]any {
	m := map[string]any{"name": f.Name}
	if f.Icon != "" {
		m["icon"] = f.Icon
	}
	if f.Cost != "" {
		m["cost"] = f.Cost
	}
	if f.Usage != "" {
		m["usage"] = f.Usage
	}
	if len(f.Keywords) > 0 {
		m["keywords"] = f.Keywords
	}
	if f.Distance != "" {
		m["distance"] = f.Distance
	}
	if f.Target != "" {
		m["target"] = f.Target
	}
	if f.PowerRoll != nil {
		pr := map[string]any{"tiers": f.PowerRoll.Tiers}
		if f.PowerRoll.Formula != "" {
			pr["formula"] = f.PowerRoll.Formula
		}
		m["power_roll"] = pr
	}
	if len(f.Sections) > 0 {
		ss := make([]map[string]any, 0, len(f.Sections))
		for _, s := range f.Sections {
			ss = append(ss, map[string]any{"label": s.Label, "text": s.Text})
		}
		m["sections"] = ss
	}
	if len(f.Enhancements) > 0 {
		es := make([]map[string]any, 0, len(f.Enhancements))
		for _, e := range f.Enhancements {
			es = append(es, map[string]any{"cost": e.Cost, "text": e.Text})
		}
		m["enhancements"] = es
	}
	if f.Body != "" {
		m["body"] = f.Body
	}
	if f.Trailing != "" {
		m["trailing"] = f.Trailing
	}
	if f.Level > 0 {
		m["level"] = f.Level
	}
	return m
}

// RichFeatureMaps converts a parsed feature list to schema-shaped maps.
func RichFeatureMaps(fs []RichFeature) []map[string]any {
	out := make([]map[string]any, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.ToMap())
	}
	return out
}
