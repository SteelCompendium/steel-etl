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
	cost         string
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

	// mdLinkRe matches any markdown link: [display](url). This is the
	// protocol-agnostic variant (matches any URL); the scc-protocol-specific form
	// lives in internal/scc/resolver.go (also named mdLinkRe). The two match
	// different input sets by design — but both rely on the same display-text class
	// `[^\]]+`, so if CommonMark link syntax changes (e.g. escaped brackets become
	// valid inside the display text), update that class in both.
	mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
)

// linkDisplay strips markdown links to their display text (e.g. "[R](scc:…)" → "R").
func linkDisplay(s string) string { return mdLinkRe.ReplaceAllString(s, "$1") }

var knownOrganizations = map[string]bool{
	"Minion": true, "Horde": true, "Platoon": true,
	"Elite": true, "Solo": true, "Leader": true, "Retainer": true,
	"Champion": true, // Summoner book Portfolio Champions
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

// splitTableCells splits one markdown table row into trimmed cells, honoring
// backslash-escaped pipes ("\|") as literal cell content rather than column
// separators. Escaped pipes are unescaped to a literal "|" in the result — this
// preserves multi-value stat cells like "4 \| 4 \| 4" (summoner minion Stamina).
func splitTableCells(line string) []string {
	line = strings.Trim(strings.TrimSpace(line), "|")
	var cells []string
	var cur strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] == '\\' && i+1 < len(line) && line[i+1] == '|' {
			cur.WriteByte('|') // escaped pipe → literal
			i++
			continue
		}
		if line[i] == '|' {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(line[i])
	}
	cells = append(cells, strings.TrimSpace(cur.String()))
	return cells
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
		rows = append(rows, splitTableCells(line))
	}
	return rows
}

var (
	sbTitleRe = regexp.MustCompile(`^([^\sA-Za-z*][^*]*?)\s*\*\*(.+?)\*\*\s*$`)
	sbParenRe = regexp.MustCompile(`^(.*?)\s*\(([^)]+)\)\s*$`)
	sbTierRe  = regexp.MustCompile(`^-\s*\*\*(≤?\d+(?:-\d+)?\+?):\*\*\s*(.*)$`)
	// sbPowerRollRe matches the Monsters labeled power-roll header "**Power Roll + N:**".
	// "Power Roll" may be link-wrapped ("**[Power Roll](scc:…) + N:**"). The whole header
	// stays in group 1; the consumer applies linkDisplay so the stored roll is link-free.
	sbPowerRollRe = regexp.MustCompile(`\*\*((?:\[Power Roll\]\([^)]*\)|Power Roll)[^*]*)\*\*`)
	// sbDiceRe splits a "Name Nd10 + <characteristic>" title (summoner statblock
	// signature abilities encode the power roll inline in the title) into the
	// clean name and the dice roll. The characteristic may be a bare token (e.g.
	// R) or a markdown link (e.g. [R](scc:…)); linkDisplay is applied to dm[2]
	// when storing the roll so it contains only the display text. sbBareTierRe
	// matches the three bare, digit-led tier outcome lines that follow (no
	// "≤11:" labels).
	sbDiceRe     = regexp.MustCompile(`^(.*?)\s+(\d+d\d+\s*\+\s*(?:\[[^\]]+\]\([^)]*\)|\S).*?)$`)
	sbBareTierRe = regexp.MustCompile(`^\d`)
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

// ParseStatblockFields parses a creature statblock's stat grid (header row +
// label cells) into the frontmatter fields shared by the data output and the
// inline-card renderer. It covers only the universal creature fields; domain/SCC
// classification and fixture handling stay in StatblockParser.Parse. The v2
// embed_cards post-pass reuses this to build an inline `.sb-wrap` card for an
// unclassified (`@classify: false`) statblock straight from its book markdown.
func ParseStatblockFields(name, body string) map[string]any {
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
	// Non-EV summon cost (Summoner book) — kept distinct from ev so consumers know
	// whether the value is an Encounter Value or a gametime cost. Cleared for
	// fixtures in applyFixtureGrid.
	if grid.header.cost != "" {
		fm["cost"] = grid.header.cost
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
	// Flavor: the prose paragraph under the heading, before the stat grid (e.g.
	// Summoner portfolio summons describe the creature there). Lifted into
	// frontmatter so it survives the v2 .sb-wrap card render, which rebuilds the
	// body from structured fields. Mirrors the featureblock parser (monster.go).
	if flavor := firstFlavorParagraph(body); flavor != "" {
		fm["flavor"] = flavor
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
	// Summoner-book statblocks carry no captain concept; their grid uses this
	// label in the same 4th-meta-cell position instead (every classified
	// Summoner statblock — minions, champions, rivals, the retainer conjurer
	// and its summons — carries it; site-only, not part of the SDK schema, so
	// it's deliberately absent from statblockScalarKeys). Consumed by
	// statblock_page.go's statblockMeta4 (FOLLOWUPS #7 piece 2).
	if v, ok := grid.labels["Free Strike Damage Type"]; ok && v != "-" && v != "—" {
		fm["free_strike_damage_type"] = v
	}

	return fm
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
	// Strip any scc links from the title before splitting: name/cost/ability_type are
	// structured fields stored link-free (the effect/tier values keep their links).
	// This also prevents a markdown link's own ")" from breaking the cost-paren split
	// (sbParenRe) — e.g. "(2 [Malice](scc:…))".
	name := linkDisplay(strings.TrimSpace(m[2]))

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

	// Dice-in-title power roll ("Molten Strike 2d10 + R"): lift the dice to the
	// effect's roll and clean the name. The tier outcomes are extracted below.
	// Apply linkDisplay to the roll so a linked characteristic (e.g.
	// [R](scc:…)) is stored as display text only ("2d10 + R").
	diceRoll := ""
	if dm := sbDiceRe.FindStringSubmatch(f["name"].(string)); dm != nil {
		f["name"] = strings.TrimSpace(dm[1])
		diceRoll = linkDisplay(strings.TrimSpace(dm[2]))
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

	effects, trigger := parseStatblockEffects(rest, diceRoll)
	if trigger != "" {
		// feature.schema.json has a dedicated top-level `trigger` field
		// (inherited by statblock.schema.json's features[] via its $ref); the
		// SDK's own MarkdownFeatureWriter always emits it first, ahead of
		// `effects` — matching every real Trigger paragraph's position (always
		// the first paragraph after the spec table). Routed here rather than
		// left as an ordinary effects entry.
		f["trigger"] = trigger
	}
	if len(effects) > 0 {
		f["effects"] = effects
		// No power roll anywhere and no keyword/usage table → a trait (the
		// narrowed taxonomy: docs/superpowers/specs/2026-06-07-feature-taxonomy-design.md).
		_, hasUsage := f["usage"]
		hasRoll := false
		for _, e := range effects {
			if _, ok := e["tier1"]; ok {
				hasRoll = true
				break
			}
		}
		if !hasUsage && !hasRoll {
			f["feature_type"] = "trait"
		}
	}

	return f
}

// parseStatblockEffects builds the SDK `effects[]` list for a statblock
// feature's body in document order (SC-308 round 3b, folds SC-309): every
// labeled paragraph (Effect/Special/…), cost enhancement ("3 Malice:"), and
// bare prose paragraph becomes its own entry, carrying the tier list that
// directly attaches to it. A `**Trigger:**` paragraph is routed to the
// separate return value instead — feature.schema.json's dedicated top-level
// `trigger` field, not an effects entry (see parseOneFeature). A tier list
// right after the spec table (nothing eligible before it) becomes its own
// {roll, tier1..3} entry, unchanged from before. curIdx tracks attachment by
// INDEX (not pointer) because effects keeps growing via append (SC-308 review
// round 2, finding L-4: a pointer into a growing slice goes stale across a
// reallocation).
func parseStatblockEffects(rest []string, diceRoll string) (effects []map[string]any, trigger string) {
	curIdx := -1
	pendingRoll := diceRoll // a header's text, seen but not yet attached to a list
	rollKeysSet := map[string]bool{}
	bareTierIdx := 0
	bareTiers := [3]string{}
	bareTiersDone := false

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
	attach := func(roll string, key, val string) {
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

	lastWasPlainProse := false // true only right after a bare-prose LINE with no
	// intervening blank line/table/roll/tier/label — lets a word-wrapped prose
	// paragraph that spans multiple physical lines join into ONE effects entry
	// instead of becoming one entry per line.
	for _, line := range rest {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "|") {
			lastWasPlainProse = false
			continue
		}
		if pr := sbPowerRollRe.FindStringSubmatch(t); pr != nil {
			pendingRoll = strings.TrimSuffix(strings.TrimSpace(linkDisplay(pr[1])), ":")
			lastWasPlainProse = false
			continue
		}
		if tm := sbTierRe.FindStringSubmatch(t); tm != nil {
			var key string
			switch {
			case strings.HasPrefix(tm[1], "≤"):
				key = "tier1"
			case strings.Contains(tm[1], "-"):
				key = "tier2"
			case strings.HasSuffix(tm[1], "+"):
				key = "tier3"
			default:
				continue
			}
			attach(pendingRoll, key, strings.TrimSpace(tm[2]))
			pendingRoll = ""
			lastWasPlainProse = false
			continue
		}
		// Dice-in-title abilities: the first run of up to three bare digit-led
		// lines below the table are the ≤11 / 12-16 / 17+ tiers, by position.
		if diceRoll != "" && !bareTiersDone && bareTierIdx < 3 && sbBareTierRe.MatchString(t) {
			bareTiers[bareTierIdx] = t
			bareTierIdx++
			if bareTierIdx == 3 {
				keys := []string{"tier1", "tier2", "tier3"}
				for i, v := range bareTiers {
					attach(diceRoll, keys[i], v)
					pendingRoll = ""
				}
			}
			lastWasPlainProse = false
			continue
		}
		if bareTierIdx > 0 {
			bareTiersDone = true // a non-tier line ends the tier run
		}
		if m := fbLabelRe.FindStringSubmatch(t); m != nil {
			// Link-free: name/cost are structured fields (SC-308 review r3, I-1).
			label := linkDisplay(strings.TrimSpace(m[1]))
			text := fbCollapse(m[2])
			if strings.EqualFold(label, "Trigger") {
				trigger = text
				curIdx = -1
				lastWasPlainProse = false
				continue
			}
			if fbCostLabelRe.MatchString(label) {
				newEntry("", label, text)
			} else {
				newEntry(label, "", text)
			}
			lastWasPlainProse = false
			continue
		}
		if lastWasPlainProse && curIdx >= 0 {
			if s, ok := effects[curIdx]["effect"].(string); ok {
				effects[curIdx]["effect"] = s + " " + t
				continue
			}
		}
		newEntry("", "", t)
		lastWasPlainProse = true
	}
	return effects, trigger
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
	// Also strip scc links: stripBold feeds the structured ability-table cells
	// (keywords / usage, and via cleanIconCell distance / target), which are stored
	// link-free even if the source links a term inside a cell (e.g. "Triggered Action").
	return linkDisplay(strings.TrimSpace(strings.ReplaceAll(s, "**", "")))
}

// cleanIconCell removes bold markers and a leading 📏/🎯 icon from a cell.
func cleanIconCell(s string) string {
	s = stripBold(s)
	for _, icon := range []string{"📏", "🎯", "🔅"} {
		s = strings.TrimSpace(strings.TrimPrefix(s, icon))
	}
	return strings.TrimSpace(s)
}

// parseCreatureKeywords parses a statblock header keyword cell into the data
// model's keyword list. Top-level commas (outside parentheses) separate distinct
// keywords; a trailing "(d1, d2, …)" parenthetical lists domain qualifiers on the
// preceding keyword and is DISTRIBUTED — one keyword per domain — so the
// Summoner-book elemental cell "Elemental (Air, Earth)" becomes
// ["Elemental (Air)", "Elemental (Earth)"] (each independently filterable) rather
// than the naive comma-split "Elemental (Air" / "Earth)". Cells with no
// parenthetical behave exactly like splitCommaList ("Abyssal, Demon" →
// ["Abyssal", "Demon"]). The faithful "Elemental (Air, Earth)" display form is
// reconstructed for the statblock head by collapseKeywords (internal/site).
func parseCreatureKeywords(s string) []string {
	var out []string
	for _, tok := range splitTopLevelCommas(s) {
		base, domains := splitKeywordDomains(tok)
		if len(domains) == 0 {
			out = append(out, base)
			continue
		}
		for _, d := range domains {
			out = append(out, base+" ("+d+")")
		}
	}
	return out
}

// splitTopLevelCommas splits on commas that sit outside any parentheses, trimming
// each part and dropping empties.
func splitTopLevelCommas(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				if p := strings.TrimSpace(s[start:i]); p != "" {
					out = append(out, p)
				}
				start = i + 1
			}
		}
	}
	if p := strings.TrimSpace(s[start:]); p != "" {
		out = append(out, p)
	}
	return out
}

// splitKeywordDomains splits a single keyword token "Base (d1, d2, …)" into its
// base and the trimmed domain list. A token with no trailing parenthetical (or an
// empty/baseless one) returns (token, nil).
func splitKeywordDomains(tok string) (string, []string) {
	tok = strings.TrimSpace(tok)
	open := strings.LastIndex(tok, "(")
	if open < 0 || !strings.HasSuffix(tok, ")") {
		return tok, nil
	}
	base := strings.TrimSpace(tok[:open])
	var domains []string
	for _, d := range strings.Split(tok[open+1:len(tok)-1], ",") {
		if d = strings.TrimSpace(d); d != "" {
			domains = append(domains, d)
		}
	}
	if base == "" || len(domains) == 0 {
		return tok, nil
	}
	return base, domains
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
		out.header.keywords = parseCreatureKeywords(header[0])
	}
	joined := strings.Join(header, " | ")
	if m := levelRe.FindStringSubmatch(joined); m != nil {
		out.header.level, _ = strconv.Atoi(m[1])
	}
	if m := evRe.FindStringSubmatch(joined); m != nil {
		out.header.ev = strings.TrimSpace(m[1])
	}
	// Cost vs EV (trailing header cell). The last column carries either an
	// Encounter Value ("EV 3 …", captured above as ev) or — first seen in the
	// Summoner book — a gametime summon cost written in plain language ("3 essence
	// for two minions", "2 Malice for two minions", "9 essence for one champion").
	// Storing the non-EV form as a distinct `cost` lets consumers tell the two
	// apart: ev is the Director's encounter-budget cost to field the creature;
	// cost is the in-play cost to summon it. The cell must be plain text — a
	// "**…**<br>Label" value cell (e.g. the summoner fixture's 2-column grid) is
	// not a cost.
	if out.header.ev == "" && len(header) > 0 {
		if last := linkDisplay(strings.TrimSpace(header[len(header)-1])); last != "" && last != "-" && last != "—" && !strings.Contains(last, "**") {
			out.header.cost = last
		}
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
				// Strip markdown links from structured stat values (e.g. a
				// Movement type rendered as [Teleport](scc:…) → "Teleport").
				value := linkDisplay(strings.TrimSpace(m[1]))
				if value == "" {
					value = "-"
				}
				out.labels[strings.TrimSpace(m[2])] = value
			}
		}
	}
	return out
}
