package content

import (
	"regexp"
	"strconv"
	"strings"
)

// statHeader holds the values from a statblock grid's header row.
type statHeader struct {
	keywords     []string
	level        int
	organization string
	role         string
	ev           string
}

// statGrid is the fully parsed stat grid: header row + label→value map.
type statGrid struct {
	header statHeader
	labels map[string]string
}

var (
	// "**VALUE**<br>Label" or "**VALUE**<br/>Label" — value may be empty bold.
	cellRe  = regexp.MustCompile(`\*\*(.*?)\*\*\s*<br\s*/?>\s*([A-Za-z][A-Za-z ]*)`)
	levelRe = regexp.MustCompile(`Level\s+(\d+)`)
	evRe    = regexp.MustCompile(`EV\s+([0-9A-Za-z+ /x-]+)`)
)

var knownOrganizations = map[string]bool{
	"Minion": true, "Horde": true, "Platoon": true,
	"Elite": true, "Solo": true, "Leader": true, "Retainer": true,
}

var knownRoles = map[string]bool{
	"Ambusher": true, "Artillery": true, "Brute": true, "Controller": true,
	"Defender": true, "Harrier": true, "Hexer": true, "Support": true, "Mount": true,
}

// splitRoleCell separates an "Org Role" cell (e.g. "Horde Hexer") into
// organization and role by matching each word against the known vocabularies.
// Organization-only cells ("Leader", "Solo") return an empty role.
func splitRoleCell(cell string) (organization, role string) {
	for _, w := range strings.Fields(cell) {
		switch {
		case knownOrganizations[w]:
			organization = w
		case knownRoles[w]:
			role = w
		}
	}
	return organization, role
}

// gridRows returns the non-separator table rows split into trimmed cells.
func gridRows(grid string) [][]string {
	var rows [][]string
	for _, line := range strings.Split(grid, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if strings.Contains(line, "---") { // separator row
			continue
		}
		raw := strings.Split(strings.Trim(line, "|"), "|")
		cells := make([]string, len(raw))
		for i, c := range raw {
			cells[i] = strings.TrimSpace(c)
		}
		rows = append(rows, cells)
	}
	return rows
}

var (
	sbTitleRe     = regexp.MustCompile(`^([^\sA-Za-z*][^*]*?)\s*\*\*(.+?)\*\*\s*$`)
	sbParenRe     = regexp.MustCompile(`^(.*?)\s*\(([^)]+)\)\s*$`)
	sbTierRe      = regexp.MustCompile(`^-\s*\*\*(≤?\d+(?:-\d+)?\+?):\*\*\s*(.*)$`)
	sbPowerRollRe = regexp.MustCompile(`\*\*(Power Roll[^*]*)\*\*`)
)

// splitBlockquoteBlocks breaks a body into individual blockquote blocks, one per
// feature. Lines that begin with ">" form a block; a non-quote line (typically a
// blank line) ends the current block. As a safety net for blocks not separated
// by a blank line, each block is further split on title boundaries.
func splitBlockquoteBlocks(body string) []string {
	var blocks []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			blocks = append(blocks, strings.Join(cur, "\n"))
			cur = nil
		}
	}
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, ">") {
			content := strings.TrimPrefix(t, ">")
			content = strings.TrimPrefix(content, " ")
			cur = append(cur, content)
		} else {
			flush()
		}
	}
	flush()

	var out []string
	for _, b := range blocks {
		out = append(out, splitOnTitles(b)...)
	}
	return out
}

// splitOnTitles splits a block whenever a new "EMOJI **Title**" line appears.
func splitOnTitles(block string) []string {
	var blocks []string
	var cur []string
	for _, line := range strings.Split(block, "\n") {
		if sbTitleRe.MatchString(strings.TrimSpace(line)) && len(cur) > 0 {
			blocks = append(blocks, strings.Join(cur, "\n"))
			cur = nil
		}
		cur = append(cur, line)
	}
	if len(cur) > 0 {
		blocks = append(blocks, strings.Join(cur, "\n"))
	}
	return blocks
}

// ParseStatblockFeatures parses the feature blockquotes of a statblock body into
// SDK-feature maps (matching feature.schema.json shape).
func ParseStatblockFeatures(body string) []map[string]any {
	var features []map[string]any
	for _, block := range splitBlockquoteBlocks(body) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if f := parseOneFeature(block); f != nil {
			features = append(features, f)
		}
	}
	return features
}

func parseOneFeature(block string) map[string]any {
	lines := strings.Split(block, "\n")
	m := sbTitleRe.FindStringSubmatch(strings.TrimSpace(lines[0]))
	if m == nil {
		return nil
	}
	icon := strings.TrimSpace(m[1])
	name := strings.TrimSpace(m[2])

	f := map[string]any{
		"type":         "feature",
		"feature_type": "ability",
		"name":         name,
	}
	if icon != "" {
		f["icon"] = icon
	}

	// Parenthetical: "(Signature Ability)" → ability_type; "(N Malice)" → cost.
	if pm := sbParenRe.FindStringSubmatch(name); pm != nil {
		f["name"] = strings.TrimSpace(pm[1])
		paren := strings.TrimSpace(pm[2])
		if strings.EqualFold(paren, "Signature Ability") {
			f["ability_type"] = paren
		} else {
			f["cost"] = paren
		}
	}

	rest := lines[1:]
	rows := featureTableRows(rest)
	switch {
	case len(rows) >= 2:
		f["keywords"] = splitCommaList(stripBold(rows[0][0]))
		f["usage"] = stripBold(rows[0][1])
		f["distance"] = cleanIconCell(rows[1][0])
		f["target"] = cleanIconCell(rows[1][1])
	case len(rows) == 1:
		f["keywords"] = splitCommaList(stripBold(rows[0][0]))
		f["usage"] = stripBold(rows[0][1])
	}

	// Effects: power-roll tiers or plain trait text.
	tiers := map[string]string{}
	var prose []string
	var roll string
	for _, line := range rest {
		t := strings.TrimSpace(line)
		if pr := sbPowerRollRe.FindStringSubmatch(t); pr != nil {
			roll = strings.TrimSuffix(strings.TrimSpace(pr[1]), ":")
			continue
		}
		if tm := sbTierRe.FindStringSubmatch(t); tm != nil {
			switch {
			case strings.HasPrefix(tm[1], "≤"):
				tiers["tier1"] = strings.TrimSpace(tm[2])
			case strings.Contains(tm[1], "-"):
				tiers["tier2"] = strings.TrimSpace(tm[2])
			case strings.HasSuffix(tm[1], "+"):
				tiers["tier3"] = strings.TrimSpace(tm[2])
			}
			continue
		}
		if t == "" || strings.HasPrefix(t, "|") {
			continue
		}
		prose = append(prose, t)
	}

	if len(tiers) > 0 {
		eff := map[string]any{"roll": roll}
		for k, v := range tiers {
			eff[k] = v
		}
		f["effects"] = []map[string]any{eff}
	} else if len(prose) > 0 {
		// No power roll and no keyword/usage table → a trait.
		if _, hasUsage := f["usage"]; !hasUsage {
			f["feature_type"] = "trait"
		}
		f["effects"] = []map[string]any{{"effect": strings.Join(prose, "\n")}}
	}

	return f
}

// featureTableRows extracts non-separator markdown table rows (2 cells each).
func featureTableRows(lines []string) [][2]string {
	var rows [][2]string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "|") || strings.Contains(t, "---") {
			continue
		}
		cells := strings.Split(strings.Trim(t, "|"), "|")
		if len(cells) >= 2 {
			rows = append(rows, [2]string{strings.TrimSpace(cells[0]), strings.TrimSpace(cells[1])})
		}
	}
	return rows
}

func stripBold(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "**", ""))
}

// cleanIconCell removes bold markers and a leading 📏/🎯 icon from a cell.
func cleanIconCell(s string) string {
	s = stripBold(s)
	for _, icon := range []string{"📏", "🎯", "🔅"} {
		s = strings.TrimSpace(strings.TrimPrefix(s, icon))
	}
	return strings.TrimSpace(s)
}

// parseStatGrid parses a statblock's 4-row markdown grid.
func parseStatGrid(grid string) statGrid {
	out := statGrid{labels: map[string]string{}}
	rows := gridRows(grid)
	if len(rows) == 0 {
		return out
	}

	// Header row.
	header := rows[0]
	if len(header) > 0 {
		out.header.keywords = splitCommaList(header[0])
	}
	joined := strings.Join(header, " | ")
	if m := levelRe.FindStringSubmatch(joined); m != nil {
		out.header.level, _ = strconv.Atoi(m[1])
	}
	if m := evRe.FindStringSubmatch(joined); m != nil {
		out.header.ev = strings.TrimSpace(m[1])
	}
	// Role cell is the one (besides the EV cell) containing an org/role word.
	for _, cell := range header {
		if strings.Contains(cell, "EV ") {
			continue
		}
		if org, role := splitRoleCell(cell); org != "" || role != "" {
			out.header.organization = org
			out.header.role = role
			break
		}
	}

	// Label/value rows (rows[1:]).
	for _, row := range rows[1:] {
		for _, cell := range row {
			if m := cellRe.FindStringSubmatch(cell); m != nil {
				value := strings.TrimSpace(m[1])
				if value == "" {
					value = "-"
				}
				out.labels[strings.TrimSpace(m[2])] = value
			}
		}
	}
	return out
}
