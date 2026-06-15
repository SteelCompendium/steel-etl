package site

// High-Fantasy Steel COMPANION statblock adapter. Beastheart companion pages are
// type: feature-group (SCC monster.companion.beastheart.statblock/<species>), not
// type: statblock — their stats live in a body table and their abilities are ##
// sections. This file parses that shape into the shared sbIsland model
// (statblock_page.go) so the companion renders as the .sb-wrap card on its own
// page (replacing the raw table) and as the .sb-prev preview on the index. The
// advancement-features section (## … {data-scc=…advancement-features…}) is left
// verbatim — its card quality is a separate task. SITE-ONLY: shared data repos
// untouched. Reuses parseAbilityTable / parseStatblockIslandFeature / resolveSbLinks.

import (
	"regexp"
	"strings"
)

// companionGrid is the parsed companion stat table: header keywords + level, and
// a label→value map across the three data rows (Size…Presence; values keep any
// markdown link, resolved later by resolveSbLinks).
type companionGrid struct {
	keywords string
	level    string
	cells    map[string]string
}

var (
	// "Level 1" inside the header row.
	companionLevelRe = regexp.MustCompile(`(?i)\bLevel\s+(\S+)`)
	// strip an attr_list suffix from a heading: "Pounce {data-scc=…}" → "Pounce".
	companionAttrRe = regexp.MustCompile(`\s*\{[^}]*\}\s*$`)
)

// parseCompanionGrid reads the first markdown table in body. The first row
// (before the :---: separator) is the header (keywords in col 0, "Level N"
// somewhere); each later data-row cell is "**value**<br>Label" → cells[Label]=value.
func parseCompanionGrid(body string) companionGrid {
	g := companionGrid{cells: map[string]string{}}
	var rows [][]string
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "|") {
			if len(rows) > 0 {
				break // table ended
			}
			continue
		}
		if strings.Contains(t, "---") {
			continue // separator row
		}
		rows = append(rows, splitRow(t))
	}
	if len(rows) == 0 {
		return g
	}
	// Header: keywords + level.
	header := strings.Join(rows[0], " ")
	if m := companionLevelRe.FindStringSubmatch(header); m != nil {
		g.level = strings.TrimSpace(m[1])
	}
	if len(rows[0]) > 0 {
		g.keywords = cellText(rows[0][0])
	}
	// Data rows: "**value**<br>Label".
	for _, row := range rows[1:] {
		for _, cell := range row {
			val, label, ok := splitCompanionCell(cell)
			if ok {
				g.cells[label] = val
			}
		}
	}
	return g
}

// splitCompanionCell splits "**value**<br>Label" → (value, label). value has its
// **bold** wrapper stripped but keeps any inner markdown link. ok=false for an
// empty/padding cell (no <br>).
func splitCompanionCell(cell string) (val, label string, ok bool) {
	parts := strings.SplitN(cell, "<br>", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	label = strings.TrimSpace(parts[1])
	val = cellText(strings.TrimSpace(parts[0]))
	if label == "" {
		return "", "", false
	}
	return val, label, true
}
