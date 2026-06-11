package site

// High-Fantasy Steel STATBLOCK pages for the Steel Compendium MkDocs site.
//
// Where ability_cards.go emits a finished `.sc-ability` card directly, statblock
// pages take the client-side-renderer path the design handoff chose: this file
// replaces a `type: statblock` page body with a `<script class="sc-statblock-data">`
// JSON island, which v2/docs/javascripts/steel-statblock.js mounts into the
// `.sb-wrap` DOM styled by steel-statblock.css. The island shape mirrors the
// handoff's statblock-data.js so the renderer is shared verbatim.
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
	"encoding/json"
	"regexp"
	"strings"
)

// ── island shape (matches statblock-data.js consumed by steel-statblock.js) ──
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
	Ancestry        string      `json:"ancestry"`
	Level           string      `json:"level"`
	Role            string      `json:"role"`
	RoleKey         string      `json:"roleKey"`
	EV              string      `json:"ev"`
	Defenses        []sbLV      `json:"defenses"`
	Meta            sbMeta      `json:"meta"`
	Characteristics []sbChar    `json:"characteristics"`
	Features        []sbFeature `json:"features"`
}

var (
	// "EMOJI **Title**" — leading icon is a non-space, non-letter, non-* char.
	sbTitleRe = regexp.MustCompile(`^([^\sA-Za-z*][^*]*?)\s*\*\*(.+?)\*\*\s*$`)
	// trailing "(…)" on a title → Signature Ability / cost / Villain Action N.
	sbParenRe = regexp.MustCompile(`^(.*?)\s*\(([^)]+)\)\s*$`)
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
// sc-statblock-data JSON island. Returns (newData, true) for statblocks; (data,
// false) otherwise so the caller writes the page unchanged. Frontmatter is
// preserved verbatim; injectH1 (next in buildSection) prepends the "# Name"
// MkDocs needs for title/nav (the CSS hides it once .sb-wrap mounts).
func buildStatblockIslandPage(data []byte) ([]byte, bool) {
	fm, body := splitFrontmatter(string(data))
	if strings.TrimSpace(parseFrontmatterField(fm, "type")) != "statblock" {
		return data, false
	}
	js, err := json.Marshal(buildStatblockIsland(fm, body))
	if err != nil {
		return data, false
	}
	// Wrap the island in a .sc-statblock-mount container. Material's
	// navigation.instant recreates inline <script>s and strips their attributes
	// (class + type), so after a client-side nav the script is no longer findable
	// by `script.sc-statblock-data` — but the container DIV's class survives.
	// steel-statblock.js locates the mount, then reads the child <script> body.
	// Same pattern as .sc-browse-mount / .sc-bestiary-mount. See
	// v2/.repo-docs/decisions/2026-06-11-client-scripts-navigation-instant.md and
	// .../2026-06-09-instant-nav-strips-script-attrs.md.
	island := "<div class=\"sc-statblock-mount\">" +
		"<script type=\"application/json\" class=\"sc-statblock-data\">\n" + string(js) + "\n</script>" +
		"</div>\n"
	return []byte("---\n" + fm + "\n---\n\n" + island), true
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

	return sbIsland{
		ID:       slugify(name),
		Name:     name,
		Ancestry: strings.Join(parseFrontmatterList(fm, "keywords"), ", "),
		Level:    strings.TrimSpace(parseFrontmatterField(fm, "level")),
		Role:     roleDisplay,
		RoleKey:  roleKey,
		EV:       strings.TrimSpace(parseFrontmatterField(fm, "ev")),
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

	f := sbFeature{Name: strings.TrimSpace(tm[2])}

	// Parenthetical → Signature / cost / Villain Action N.
	if pm := sbParenRe.FindStringSubmatch(f.Name); pm != nil {
		f.Name = strings.TrimSpace(pm[1])
		paren := strings.TrimSpace(pm[2])
		if strings.EqualFold(paren, "Signature Ability") {
			f.Cost = "Signature"
		} else {
			f.Cost = paren
		}
	}

	// Dice-in-title power roll (summoner signatures) → formula + clean name.
	diceFormula := ""
	if dm := sbTitleDiceRe.FindStringSubmatch(f.Name); dm != nil {
		f.Name = strings.TrimSpace(dm[1])
		diceFormula = linkText(strings.TrimSpace(dm[2]))
	}

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
				f.Distance = resolveSbLinks(dist)
			}
			if tgt != "" {
				f.Target = resolveSbLinks(tgt)
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
			if sbCostLabelRe.MatchString(label) {
				f.Enhancements = append(f.Enhancements, sbEnh{Cost: label, Text: resolveSbLinks(text)})
			} else {
				f.Sections = append(f.Sections, sbSection{Label: label, Text: resolveSbLinks(text)})
			}
			continue
		}

		// Unlabeled prose → trailing note (ability) or body (trait, no table).
		prose = append(prose, collapseLines(tp))
	}

	if !tableSeen {
		// No keyword/usage table → a passive trait (Monsters book trait home).
		f.Kind, f.Action = "passive", "passive"
		f.Body = resolveSbLinks(strings.Join(prose, "\n\n"))
		return f, true
	}

	f.Action, f.Kind = sbActionKind(usage, f.Cost)
	if usage != "" && usage != "-" {
		f.Usage = usage
	}
	if tiersSeen {
		t := map[string]string{}
		for i, key := range []string{"low", "mid", "high"} {
			if tiers[i] != "" {
				t[key] = resolveSbLinks(tiers[i])
			}
		}
		f.PowerRoll = &sbPowerRoll{Formula: formula, Tiers: t}
	}
	if len(prose) > 0 {
		f.Trailing = resolveSbLinks(strings.Join(prose, " "))
	}
	return f, true
}

// sbActionKind maps a parsed usage + cost to the renderer's action/kind. A
// "Villain Action N" cost makes it a villain feature; otherwise the usage word
// picks the action accent (main is the default).
func sbActionKind(usage, cost string) (action, kind string) {
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

// linkText strips markdown links to their display text ("[R](scc:…)" → "R");
// used only for the cosmetic dice formula.
func linkText(s string) string { return mdLinkRe.ReplaceAllString(s, "$1") }

// resolveSbLinks rewrites each markdown link's TARGET in a feature text field
// to the directory-URL form MkDocs serves, keeping the [text](href) markdown so
// steel-statblock.js's rich() emits a working <a>. The island is raw JSON that
// MkDocs never post-processes, so — exactly like the ability cards (cardHref) —
// we resolve the ".md" + use_directory_urls depth here instead of shipping a
// dead ".md" link client-side. (Feature text is link-swept; frontmatter is not,
// so meta/ancestry need no resolution.)
func resolveSbLinks(s string) string {
	return mdLinkRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdLinkRe.FindStringSubmatch(m)
		return "[" + sub[1] + "](" + cardHref(sub[2]) + ")"
	})
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
