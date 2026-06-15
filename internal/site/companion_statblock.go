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

// buildCompanionStatblockIsland maps a companion feature-group page (frontmatter +
// base-region body) onto the shared sbIsland. Stats come from the body grid
// (companions carry no stat frontmatter); abilities from the ## sections. Role is
// the literal "Companion" (grey "leader" accent — not a knownRoleKey); EV is empty
// (omitted by renderStatblockHead). The Skills grid cell rides in the meta block's
// Captain slot, relabeled "Skills"; Weakness is "—".
func buildCompanionStatblockIsland(fm, baseBody string) sbIsland {
	g := parseCompanionGrid(baseBody)
	name := strings.TrimSpace(parseFrontmatterField(fm, "name"))
	level := g.level
	if level == "" {
		level = strings.TrimSpace(parseFrontmatterField(fm, "level"))
	}
	metaVal := func(label string) string {
		if v := strings.TrimSpace(g.cells[label]); v != "" {
			return resolveSbLinks(v)
		}
		return "—"
	}
	return sbIsland{
		ID:       slugify(name),
		Name:     name,
		Ancestry: g.keywords,
		Level:    level,
		Role:     "Companion",
		RoleKey:  "leader",
		EV:       "",
		Defenses: []sbLV{
			{L: "Size", V: orDash(g.cells["Size"])},
			{L: "Speed", V: orDash(g.cells["Speed"])},
			{L: "Stamina", V: orDash(g.cells["Stamina"])},
			{L: "Stability", V: orDash(g.cells["Stability"])},
			{L: "Free Strike", V: orDash(g.cells["Free Strike"])},
		},
		Meta: sbMeta{
			Immunity: metaVal("Immunity"),
			Weakness: "—",
			Movement: metaVal("Movement"),
			Captain:  sbCaptain{Label: "Skills", Value: metaVal("Skills")},
		},
		Characteristics: []sbChar{
			{L: "Might", K: "M", V: signValue(g.cells["Might"])},
			{L: "Agility", K: "A", V: signValue(g.cells["Agility"])},
			{L: "Reason", K: "R", V: signValue(g.cells["Reason"])},
			{L: "Intuition", K: "I", V: signValue(g.cells["Intuition"])},
			{L: "Presence", K: "P", V: signValue(g.cells["Presence"])},
		},
		Features: companionFeatures(baseBody),
	}
}

// companionFeatures splits the base region into its ability ## sections (the caller
// passes a body already trimmed at the advancement-features boundary) and parses
// each into an sbFeature. It reuses parseStatblockIslandFeature by synthesizing the
// title line it expects ("• **Name**"), so all the spec-table / Effect / Spend /
// passive logic is shared with monster statblocks — no duplicate feature parsing.
func companionFeatures(baseBody string) []sbFeature {
	var out []sbFeature
	for _, sec := range companionAbilitySections(baseBody) {
		block := "• **" + sec.name + "**\n\n" + sec.body
		if f, ok := parseStatblockIslandFeature(block); ok {
			out = append(out, f)
		}
	}
	return out
}

type companionSection struct {
	name string
	body string
}

// companionAbilitySections returns each "## Heading … body" section in document
// order, heading text stripped of any {attr_list} suffix. Content before the first
// ## (the stat table) is ignored.
func companionAbilitySections(body string) []companionSection {
	var secs []companionSection
	var cur *companionSection
	var buf []string
	flush := func() {
		if cur != nil {
			cur.body = strings.TrimSpace(strings.Join(buf, "\n"))
			secs = append(secs, *cur)
		}
		buf = nil
	}
	for _, line := range strings.Split(body, "\n") {
		if h, ok := strings.CutPrefix(strings.TrimSpace(line), "## "); ok {
			flush()
			name := strings.TrimSpace(companionAttrRe.ReplaceAllString(h, ""))
			cur = &companionSection{name: name}
			continue
		}
		if cur != nil {
			buf = append(buf, line)
		}
	}
	flush()
	return secs
}
