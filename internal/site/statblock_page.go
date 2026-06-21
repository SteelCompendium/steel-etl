package site

// High-Fantasy Steel STATBLOCK pages for the Steel Compendium MkDocs site.
//
// Where this file once emitted a JSON island for steel-statblock.js to mount,
// it now parses a `type: statblock` page into the sbIsland model and hands it to
// renderStatblockCard (statblock_card.go), which emits the finished .sb-wrap DOM
// at build time — the featureblock_page.go model. This file owns the PARSE stage
// (frontmatter + body blockquotes → sbIsland); statblock_card.go owns rendering.
//
// SITE-ONLY: runs inside `steel-etl site` against the generated md-linked pages;
// the shared data repos are never touched. The structured stats come from
// frontmatter; the features are parsed from the body's blockquotes (richer than
// the lossy frontmatter — it keeps Effect/Trigger sections, Malice enhancements,
// and trailing notes that the SDK statblock JSON drops once power-roll tiers are
// present). Reuses the ability-card body helpers (parseAbilityTable / parseTiers
// / prHeadRe / labelRe / paraSplitRe / …) from ability_cards.go.
//
// NOT YET: the shared family Malice band is intentionally omitted (the README
// marks it a non-blocking nice-to-have, and the family's malice featureblock
// still renders as its own page). Wiring it in is a follow-up.

import (
	"regexp"
	"strings"
)

// ── statblock model (the intermediate the parse stage builds; renderStatblockCard
//
//	in statblock_card.go turns it into the .sb-wrap HTML at build time) ──
type sbLV struct {
	L string `json:"l"`
	V string `json:"v"`
}
type sbChar struct {
	L string `json:"l"`
	K string `json:"k"`
	V string `json:"v"`
}
type sbCaptain struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type sbMeta struct {
	Immunity string    `json:"immunity"`
	Weakness string    `json:"weakness"`
	Movement string    `json:"movement"`
	Captain  sbCaptain `json:"captain"`
}
type sbPowerRoll struct {
	Formula string            `json:"formula"` // "" → a test result (renderer omits the head)
	Tiers   map[string]string `json:"tiers"`
}
type sbSection struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}
type sbEnh struct {
	Cost string `json:"cost"`
	Text string `json:"text"`
}
type sbFeature struct {
	Kind         string       `json:"kind"`   // ability | passive | villain
	Action       string       `json:"action"` // main | maneuver | triggered | move | passive | villain
	Name         string       `json:"name"`
	Cost         string       `json:"cost,omitempty"`
	Usage        string       `json:"usage,omitempty"`
	Keywords     []string     `json:"keywords,omitempty"`
	Distance     string       `json:"distance,omitempty"`
	Target       string       `json:"target,omitempty"`
	PowerRoll    *sbPowerRoll `json:"powerRoll,omitempty"`
	Sections     []sbSection  `json:"sections,omitempty"`
	Enhancements []sbEnh      `json:"enhancements,omitempty"`
	Body         string       `json:"body,omitempty"`
	Trailing     string       `json:"trailing,omitempty"`
}
type sbIsland struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Flavor          string      `json:"flavor,omitempty"`
	Ancestry        string      `json:"ancestry"`
	Level           string      `json:"level"`
	Role            string      `json:"role"`
	RoleKey         string      `json:"roleKey"`
	EV              string      `json:"ev"`
	Cost            string      `json:"cost"`
	Defenses        []sbLV      `json:"defenses"`
	Meta            sbMeta      `json:"meta"`
	Characteristics []sbChar    `json:"characteristics"`
	Features        []sbFeature `json:"features"`
}

var (
	// "EMOJI **Title**" — leading icon is a non-space, non-letter, non-* char.
	sbTitleRe = regexp.MustCompile(`^([^\sA-Za-z*][^*]*?)\s*\*\*(.+?)\*\*\s*$`)
	// (trailing "(…)" on a title is split by splitTrailingParen, not a regex, so
	// a link's own nested "(…)" doesn't break the match.)
	// "Name Nd10 + <char>" — summoner signatures encode the roll in the title.
	sbTitleDiceRe = regexp.MustCompile(`^(.*?)\s+(\d+d\d+\s*\+\s*\S.*?)$`)
	sbBareTierRe  = regexp.MustCompile(`^\d`)
	// a labeled paragraph whose label is a cost ("2 Malice", "5+ Malice", "Spend …").
	sbCostLabelRe = regexp.MustCompile(`(?i)^(?:\d+\+?\s+\S+.*|spend\b.*)$`)
	// captain / free-strike-bonus line that lives in the body, not frontmatter.
	sbCaptainRe = regexp.MustCompile(`(?im)^\s*with captain:\s*(.+?)\s*$`)
)

// knownRoleKeys are the data-role values steel-statblock.css colors. roleKey is
// snapped to one of these (grey "leader" fallback) so --role always resolves.
var knownRoleKeys = map[string]bool{
	"ambusher": true, "harrier": true, "artillery": true, "brute": true,
	"controller": true, "leader": true, "solo": true, "hexer": true,
	"mount": true, "support": true, "defender": true, "minion": true,
}

// buildStatblockIslandPage rewrites a `type: statblock` page body into the
// build-time .sb-wrap card (via renderStatblockCard). Returns (newData, true) for
// statblocks; (data, false) otherwise so the caller writes the page unchanged.
// Frontmatter is preserved verbatim; injectH1 (next in buildSection) prepends the
// "# Name" MkDocs needs for title/nav (the CSS hides it when .sb-wrap is present).
// (Name kept for its build.go/test call sites; it no longer emits a JSON island.)
func buildStatblockIslandPage(data []byte) ([]byte, bool) {
	fm, body := splitFrontmatter(string(data))
	if strings.TrimSpace(parseFrontmatterField(fm, "type")) != "statblock" {
		return data, false
	}
	// Fixtures are statblocks in `type` only; they render as Forged Band
	// featureblock cards (buildFixturePage), not the creature .sb-wrap card.
	if strings.TrimSpace(parseFrontmatterField(fm, "statblock_kind")) == "fixture" {
		return data, false
	}
	// Build-time HTML card (the featureblock_page.go model): renderStatblockCard
	// emits the same .sb-wrap DOM steel-statblock.js used to build client-side, so
	// the card can later be embedded inline on any page. Contiguous (no blank
	// lines) so md_in_html passes it through verbatim.
	//
	// Retainer advancement is no longer split out here: as of Plan 6 each retainer's
	// advancement abilities are a real `monster.retainer.advancement-features/<id>`
	// featureblock entity (its own paired page), so the statblock body holds only the
	// innate abilities and renders straight through.
	island := buildStatblockIsland(fm, body)
	if scc := strings.TrimSpace(parseFrontmatterField(fm, "scc")); scc != "" {
		statblockFeatureCache[scc] = island.Features
	}
	card := renderStatblockCard(island)
	return []byte("---\n" + fm + "\n---\n\n" + card), true
}

// collapseKeywords reconstructs the book-faithful keyword display string from the
// distributed keyword list emitted by parseCreatureKeywords. Per-domain entries
// that share a base ("Elemental (Air)", "Elemental (Earth)") recombine into the
// sourcebook form "Elemental (Air, Earth)"; plain keywords join with ", " in
// first-seen order.
func collapseKeywords(kws []string) string {
	type grp struct {
		base    string
		domains []string
	}
	var order []*grp
	byBase := map[string]*grp{}
	for _, kw := range kws {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		base, dom := kw, ""
		if open := strings.LastIndex(kw, "("); open >= 0 && strings.HasSuffix(kw, ")") {
			base = strings.TrimSpace(kw[:open])
			dom = strings.TrimSpace(kw[open+1 : len(kw)-1])
		}
		g := byBase[base]
		if g == nil {
			g = &grp{base: base}
			byBase[base] = g
			order = append(order, g)
		}
		if dom != "" {
			g.domains = append(g.domains, dom)
		}
	}
	parts := make([]string, 0, len(order))
	for _, g := range order {
		if len(g.domains) > 0 {
			parts = append(parts, g.base+" ("+strings.Join(g.domains, ", ")+")")
		} else {
			parts = append(parts, g.base)
		}
	}
	return strings.Join(parts, ", ")
}

func buildStatblockIsland(fm, body string) sbIsland {
	name := strings.TrimSpace(parseFrontmatterField(fm, "name"))
	org := strings.TrimSpace(parseFrontmatterField(fm, "organization"))
	role := strings.TrimSpace(parseFrontmatterField(fm, "role"))

	roleDisplay := strings.TrimSpace(org + " " + role)
	roleKey := strings.ToLower(role)
	if roleKey == "" {
		roleKey = strings.ToLower(org)
	}
	if !knownRoleKeys[roleKey] {
		roleKey = "leader" // grey neutral so --role always resolves
	}

	captain := "—"
	if m := sbCaptainRe.FindStringSubmatch(body); m != nil {
		captain = strings.TrimSpace(m[1])
	}

	// The keyword line (sb__kw eyebrow) reconstructs the book-faithful display
	// from the distributed keyword list (per-domain "Elemental (Air)"/"Elemental
	// (Earth)" recollapse to "Elemental (Air, Earth)"). For summoner-book
	// statblocks the line is otherwise "—" or junk, so the scc-derived provenance
	// label replaces it — but the faithful domain parenthetical is appended so the
	// head still reads like the sourcebook ("Summoner Minion · Elemental (Air,
	// Earth)"). Only summoner elementals carry domains, so the first "(" is theirs.
	faithful := collapseKeywords(parseFrontmatterList(fm, "keywords"))
	ancestry := faithful
	if eb := summonerProvenanceEyebrow(parseFrontmatterField(fm, "scc")); eb != "" {
		ancestry = eb
		if open := strings.Index(faithful, "("); open >= 0 {
			ancestry += " " + strings.TrimSpace(faithful[open:])
		}
	}

	return sbIsland{
		ID:       slugify(name),
		Name:     name,
		Flavor:   strings.TrimSpace(parseFrontmatterField(fm, "flavor")),
		Ancestry: ancestry,
		Level:    strings.TrimSpace(parseFrontmatterField(fm, "level")),
		Role:     roleDisplay,
		RoleKey:  roleKey,
		EV:       strings.TrimSpace(parseFrontmatterField(fm, "ev")),
		Cost:     strings.TrimSpace(parseFrontmatterField(fm, "cost")),
		Defenses: []sbLV{
			{L: "Size", V: orDash(parseFrontmatterField(fm, "size"))},
			{L: "Speed", V: orDash(parseFrontmatterField(fm, "speed"))},
			{L: "Stamina", V: orDash(parseFrontmatterField(fm, "stamina"))},
			{L: "Stability", V: orDash(parseFrontmatterField(fm, "stability"))},
			{L: "Free Strike", V: orDash(parseFrontmatterField(fm, "free_strike"))},
		},
		Meta: sbMeta{
			Immunity: joinOrDash(parseFrontmatterList(fm, "immunities")),
			Weakness: joinOrDash(parseFrontmatterList(fm, "weaknesses")),
			Movement: orDash(parseFrontmatterField(fm, "movement")),
			Captain:  sbCaptain{Label: "With Captain", Value: captain},
		},
		Characteristics: []sbChar{
			{L: "Might", K: "M", V: signValue(parseFrontmatterField(fm, "might"))},
			{L: "Agility", K: "A", V: signValue(parseFrontmatterField(fm, "agility"))},
			{L: "Reason", K: "R", V: signValue(parseFrontmatterField(fm, "reason"))},
			{L: "Intuition", K: "I", V: signValue(parseFrontmatterField(fm, "intuition"))},
			{L: "Presence", K: "P", V: signValue(parseFrontmatterField(fm, "presence"))},
		},
		Features: parseStatblockIslandFeatures(body),
	}
}

// parseStatblockIslandFeatures splits the body into feature blockquotes and
// parses each into the renderer's feature shape (full sections / enhancements /
// trailing — richer than the SDK statblock JSON).
func parseStatblockIslandFeatures(body string) []sbFeature {
	var out []sbFeature
	for _, block := range sbBlocks(body) {
		if f, ok := parseStatblockIslandFeature(block); ok {
			out = append(out, f)
		}
	}
	return out
}

// sbBlocks groups consecutive ">"-quoted lines into blocks (a non-quote line
// ends a block), then splits each block on internal "EMOJI **Title**" lines as a
// safety net for features not separated by a blank line. Mirrors the content
// package's splitBlockquoteBlocks.
func sbBlocks(body string) []string {
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
			c := strings.TrimPrefix(strings.TrimPrefix(t, ">"), " ")
			cur = append(cur, c)
		} else {
			flush()
		}
	}
	flush()

	var out []string
	for _, b := range blocks {
		out = append(out, sbSplitOnTitles(b)...)
	}
	return out
}

func sbSplitOnTitles(block string) []string {
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

func parseStatblockIslandFeature(block string) (sbFeature, bool) {
	block = strings.TrimSpace(block)
	paras := paraSplitRe.Split(block, -1)
	if len(paras) == 0 {
		return sbFeature{}, false
	}
	tm := sbTitleRe.FindStringSubmatch(strings.TrimSpace(paras[0]))
	if tm == nil {
		return sbFeature{}, false
	}

	var f sbFeature
	name := strings.TrimSpace(tm[2])

	// Trailing "(…)" → Signature / cost / Villain Action N. A balanced-paren
	// split so a link's own "(…)" inside the group — e.g. "(2 [Malice](url))"
	// or "([Villain Action](url) 3)" — doesn't truncate it.
	// Match on the link-stripped text but keep the raw link for display (richSb
	// resolves it at render).
	if base, inner, ok := splitTrailingParen(name); ok {
		name = base
		inner = strings.TrimSpace(inner)
		if strings.EqualFold(linkText(inner), "Signature Ability") {
			f.Cost = "Signature"
		} else {
			f.Cost = inner
		}
	}

	// Dice-in-title power roll (summoner signatures) → formula + clean name.
	diceFormula := ""
	if dm := sbTitleDiceRe.FindStringSubmatch(name); dm != nil {
		name = strings.TrimSpace(dm[1])
		diceFormula = linkText(strings.TrimSpace(dm[2]))
	}
	// The base name may itself be a link (e.g. "[Solo](url) Monster").
	f.Name = name

	var (
		tableSeen bool
		usage     string
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

		// 2×2 spec table → keywords / usage / distance / target.
		if strings.HasPrefix(tp, "|") {
			kws, act, dist, tgt := parseAbilityTable(para)
			if len(kws) > 0 {
				f.Keywords = kws
			}
			if act != "" {
				usage = act
			}
			if dist != "" {
				f.Distance = dist
			}
			if tgt != "" {
				f.Target = tgt
			}
			tableSeen = true
			continue
		}

		// Power-roll header → formula ("+ 4"); the next list holds the tiers.
		if m := prHeadRe.FindStringSubmatch(tp); m != nil {
			formula = "+ " + strings.TrimSpace(m[1])
			continue
		}

		// Labeled tier list ("- **≤11:** …") — present for both labeled rolls
		// and bare test results (no header).
		if looksLikeTiers(tp) {
			parseTiers(tp, &tiers)
			tiersSeen = true
			continue
		}

		// Dice-in-title abilities: bare digit-led lines below the table are the
		// ≤11 / 12-16 / 17+ tiers by position.
		if diceFormula != "" && bareIdx < 3 && sbBareTierRe.MatchString(tp) {
			tiers[bareIdx] = collapseLines(tp)
			bareIdx++
			tiersSeen = true
			continue
		}

		// Labeled paragraph → cost enhancement (2 Malice / Spend …) or a titled
		// Effect / Trigger / Special section.
		if m := labelRe.FindStringSubmatch(tp); m != nil {
			label := strings.TrimSpace(m[1])
			text := collapseLines(m[2])
			// Classify on the link-stripped label ("2 [Malice](url)" → "2 Malice")
			// but keep the raw link on the stored cost/label for display (richSb
			// resolves it at render).
			if sbCostLabelRe.MatchString(linkText(label)) {
				f.Enhancements = append(f.Enhancements, sbEnh{Cost: label, Text: text})
			} else {
				f.Sections = append(f.Sections, sbSection{Label: label, Text: text})
			}
			continue
		}

		// Unlabeled prose → trailing note (ability) or body (trait, no table).
		prose = append(prose, collapseLines(tp))
	}

	if !tableSeen {
		// No keyword/usage table → a passive trait (Monsters book trait home).
		f.Kind, f.Action = "passive", "passive"
		f.Body = strings.Join(prose, "\n\n")
		return f, true
	}

	f.Action, f.Kind = sbActionKind(usage, f.Cost)
	if usage != "" && usage != "-" {
		// A few usage cells link "Triggered Action" etc.; the raw [text](target)
		// is stored verbatim and richSb (statblock_card.go) resolves it to a
		// working <a> at render, the same as distance/target.
		f.Usage = usage
	}
	if tiersSeen {
		t := map[string]string{}
		for i, key := range []string{"low", "mid", "high"} {
			if tiers[i] != "" {
				t[key] = tiers[i]
			}
		}
		f.PowerRoll = &sbPowerRoll{Formula: formula, Tiers: t}
	}
	if len(prose) > 0 {
		f.Trailing = strings.Join(prose, " ")
	}
	return f, true
}

// sbActionKind maps a parsed usage + cost to the renderer's action/kind. A
// "Villain Action N" cost makes it a villain feature; otherwise the usage word
// picks the action accent (main is the default).
func sbActionKind(usage, cost string) (action, kind string) {
	// Strip any links first — usage/cost may now carry a resolved link
	// ("[Villain Action](url) 3") that would defeat the prefix/contains checks.
	cost = linkText(cost)
	usage = linkText(usage)
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(cost)), "villain action") {
		return "villain", "villain"
	}
	u := strings.ToLower(usage)
	switch {
	case strings.Contains(u, "trigger"):
		return "triggered", "ability"
	case strings.Contains(u, "maneuver"):
		return "maneuver", "ability"
	case strings.Contains(u, "move"):
		return "move", "ability"
	default:
		return "main", "ability"
	}
}

// linkText strips markdown links to their display text ("[R](scc:…)" → "R").
func linkText(s string) string { return mdLinkRe.ReplaceAllString(s, "$1") }

// splitTrailingParen splits "Name (inner)" into the name and the inner content
// of the LAST balanced top-level "(…)" group, matched by scanning from the end
// with depth counting so a markdown link's own "(…)" nested inside the group
// (e.g. "(2 [Malice](url))") doesn't truncate the match. ok is false when there
// is no balanced trailing group.
func splitTrailingParen(s string) (base, inner string, ok bool) {
	s = strings.TrimRight(s, " ")
	if !strings.HasSuffix(s, ")") {
		return s, "", false
	}
	depth := 0
	for i := len(s) - 1; i >= 0; i-- {
		switch s[i] {
		case ')':
			depth++
		case '(':
			depth--
			if depth == 0 {
				// A "(" right after "]" is a markdown link target, not a wrapping
				// parenthetical — e.g. a title that is itself a link "[End Effect](url)".
				if i > 0 && s[i-1] == ']' {
					return s, "", false
				}
				return strings.TrimRight(s[:i], " "), s[i+1 : len(s)-1], true
			}
		}
	}
	return s, "", false
}

func joinOrDash(list []string) string {
	if s := strings.TrimSpace(strings.Join(list, ", ")); s != "" && s != "—" && s != "-" {
		return s
	}
	return "—"
}

// signValue ensures a characteristic reads with an explicit sign ("+1", "−3").
// Frontmatter stores a bare or pre-signed value; a bare non-negative gets a "+".
func signValue(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	switch s[0] {
	case '+', '-':
		return s
	}
	if strings.HasPrefix(s, "−") { // U+2212 minus
		return s
	}
	return "+" + s
}

// slugify lowercases a name into a URL/id-safe slug ("Devil High Judge" →
// "devil-high-judge").
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
