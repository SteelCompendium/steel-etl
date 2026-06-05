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
