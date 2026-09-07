package site

// High-Fantasy Steel FEATUREBLOCK pages for the Steel Compendium MkDocs site.
//
// Featureblocks (malice blocks, named feature blocks like Ajax's Tactical
// Stance) and dynamic terrain are a titled COLLECTION of features under a
// loose-stat header — statblock-like in anatomy, not rigor. Where
// statblock_page.go emits a JSON island the client renderer mounts, this emits
// a finished `.fb-wrap` card at BUILD TIME (the ability_cards.go model), so the
// same renderer can later embed cards inside non-focused pages (spec
// docs/superpowers/specs/2026-06-12-featureblock-cards-design.md, "Architecture
// choice B").
//
// SITE-ONLY: runs inside `steel-etl site` against the generated md-linked pages;
// the shared data repos are never touched. Plan 1 made the page frontmatter
// non-lossy (kind/level/flavor/role/terrain_type/stats[]/features[], validated
// by featureblock.schema.json), so this reads frontmatter directly — NO body
// re-parse. Each feature is `article.sc-ability.fb__feat`, reusing the
// ability-card grammar (costBadge / richInline / cardHref / tierGlyph /
// renderSectionBlock / sbActionKind from ability_cards.go + statblock_page.go).
//
// SCOPE (Plan 2): type:featureblock + type:dynamic-terrain only. Fixture routing
// (Plan 3), retainer advancement split (Plan 4), and companion advancement cards
// (Plan 5) are NOT handled here.

import (
	"fmt"
	"html"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"gopkg.in/yaml.v3"
)

// ── frontmatter shape (mirrors featureblock.schema.json) ──
type fbPowerRoll struct {
	Formula string            `yaml:"formula"`
	Tiers   map[string]string `yaml:"tiers"`
}
type fbSection struct {
	Label string `yaml:"label"`
	Text  string `yaml:"text"`
}
type fbEnh struct {
	Cost string `yaml:"cost"`
	Text string `yaml:"text"`
}

// fbEffect is one entry of a feature's ordered body (SC-308 round 3b): the
// SDK feature.schema.json `effect` shape — a labeled section (Name), a cost
// enhancement (Cost), or bare prose (neither), plus the tier list (Roll/
// Tier1..3) that attaches to it (see fbFeaturesFromRich). YAML tags match the
// `effects` frontmatter key exactly (featureblock.schema.json). Roll is the
// full "**Power Roll + N:**" header text ("Power Roll + 3") or a bare dice
// formula ("2d10 + R"); "" for a header-less list — the site derives its
// display head from THIS entry's own Effect text at RENDER time
// (fbEffectRollHTML), never stored here.
type fbEffect struct {
	Name   string `yaml:"name,omitempty"`
	Cost   string `yaml:"cost,omitempty"`
	Effect string `yaml:"effect,omitempty"`
	Roll   string `yaml:"roll,omitempty"`
	Tier1  string `yaml:"tier1,omitempty"`
	Tier2  string `yaml:"tier2,omitempty"`
	Tier3  string `yaml:"tier3,omitempty"`
}
type fbFeature struct {
	Icon         string       `yaml:"icon"`
	Name         string       `yaml:"name"`
	ID           string       `yaml:"-"` // SC-306: id minted by featID for this rendering pass, not sourced from YAML
	Cost         string       `yaml:"cost"`
	Usage        string       `yaml:"usage"`
	Keywords     []string     `yaml:"keywords"`
	Distance     string       `yaml:"distance"`
	Target       string       `yaml:"target"`
	PowerRoll    *fbPowerRoll `yaml:"power_roll"`
	Sections     []fbSection  `yaml:"sections"`
	Enhancements []fbEnh      `yaml:"enhancements"`
	Intro        string       `yaml:"intro"`
	Body         string       `yaml:"body"`
	Trailing     string       `yaml:"trailing"`
	Level        int          `yaml:"level"`
	// Effects is the AUTHORITATIVE, ordered, non-lossy render order (SC-308
	// round 3b) — emitted IN ADDITION TO the flat fields above (kept for the
	// SDK featureblock schema + its round-trip test). renderFbFeat walks THIS.
	Effects []fbEffect `yaml:"effects"`
}
type fbStat struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
type fbDoc struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"`
	Kind        string      `yaml:"kind"`
	Level       int         `yaml:"level"`
	Flavor      string      `yaml:"flavor"`
	Role        string      `yaml:"role"`
	TerrainType string      `yaml:"terrain_type"`
	Stats       []fbStat    `yaml:"stats"`
	Features    []fbFeature `yaml:"features"`
	Eyebrow     string      `yaml:"-"` // synthetic override (retainer advancement); not from frontmatter
}

// buildFeatureblockPage rewrites a type:featureblock | type:dynamic-terrain page
// body into the .fb-wrap card. Returns (newData, true) when handled; (data,
// false) otherwise so the caller writes the page unchanged. Frontmatter is
// preserved verbatim; injectH1 (next in buildSection) prepends the "# Name"
// MkDocs needs for title/nav (CSS hides it once .fb-wrap is present).
func buildFeatureblockPage(data []byte) ([]byte, bool) {
	fm, _ := splitFrontmatter(string(data))
	switch strings.TrimSpace(parseFrontmatterField(fm, "type")) {
	case "featureblock", "dynamic-terrain":
	default:
		return data, false
	}
	var doc fbDoc
	if err := yaml.Unmarshal([]byte(fm), &doc); err != nil {
		return data, false // malformed frontmatter → leave page as-is
	}
	// A summoner fixture's left-deck provenance ("Summoner · <Element>") derives from
	// its SCC; don't clobber a synthetic Eyebrow (retainer advancement) if already set.
	if strings.TrimSpace(doc.Eyebrow) == "" {
		doc.Eyebrow = fbOrigin(parseFrontmatterField(fm, "scc"))
	}
	card := renderFeatureblockCard(doc)
	return []byte("---\n" + fm + "\n---\n\n" + card), true
}

// fbFeaturesFromRich maps the shared content.RichFeature shape onto the site
// renderer's fbFeature. The two are intentionally congruent (spec §2). The icon
// is preserved so a table-less fixture passive (⭐) gets its action accent from
// the emoji (fbFeatureAction) rather than flattening to "passive".
func fbFeaturesFromRich(rfs []content.RichFeature) []fbFeature {
	out := make([]fbFeature, 0, len(rfs))
	for _, r := range rfs {
		f := fbFeature{
			Icon:     r.Icon,
			Name:     r.Name,
			Cost:     r.Cost,
			Usage:    r.Usage,
			Keywords: r.Keywords,
			Distance: r.Distance,
			Target:   r.Target,
			Intro:    r.Intro,
			Body:     r.Body,
			Trailing: r.Trailing,
			Level:    r.Level,
		}
		if r.PowerRoll != nil {
			f.PowerRoll = &fbPowerRoll{Formula: r.PowerRoll.Formula, Tiers: r.PowerRoll.Tiers}
		}
		for _, e := range r.Effects {
			f.Effects = append(f.Effects, fbEffect{
				Name: e.Name, Cost: e.Cost, Effect: e.Effect, Roll: e.Roll,
				Tier1: e.Tier1, Tier2: e.Tier2, Tier3: e.Tier3,
			})
		}
		for _, s := range r.Sections {
			f.Sections = append(f.Sections, fbSection{Label: s.Label, Text: s.Text})
		}
		for _, e := range r.Enhancements {
			f.Enhancements = append(f.Enhancements, fbEnh{Cost: e.Cost, Text: e.Text})
		}
		out = append(out, f)
	}
	return out
}

// fbDataRole maps a doc to the [data-role] the CSS colors. Terrain/fixtures use
// their combat role; malice/feature blocks fall back to grey via the
// "malice"/"feature" keys (defined in steel-featureblock.css).
func fbDataRole(doc fbDoc) string {
	if r := strings.ToLower(strings.TrimSpace(doc.Role)); r != "" {
		return r
	}
	if doc.Kind == "malice" {
		return "malice"
	}
	return "feature"
}

// fbKindNoun is the left-eyebrow kind-noun for a featureblock family card.
func fbKindNoun(doc fbDoc) string {
	switch doc.Kind {
	case "dynamic-terrain":
		return "Dynamic Terrain"
	case "fixture":
		return "Fixture"
	case "malice":
		return "Malice"
	case "advancement":
		return "Advancement"
	default:
		return "Featureblock"
	}
}

// fbTypeRole combines the descriptive type and combat role the way the book does
// ("Trap Hazard") for the right-primary mini-title. Either part may be absent.
func fbTypeRole(doc fbDoc) string {
	return strings.TrimSpace(strings.TrimSpace(doc.TerrainType) + " " + strings.TrimSpace(doc.Role))
}

// fbEV pulls the "EV" loose stat for the right-deck chip ("" if absent).
func fbEV(stats []fbStat) string {
	for _, st := range stats {
		if strings.EqualFold(strings.TrimSpace(st.Name), "EV") {
			return "EV " + strings.TrimSpace(st.Value)
		}
	}
	return ""
}

// fbOrigin derives the left-deck provenance for a summoner featureblock from its
// SCC type-path:
//
//	monster.fixture.<circle>.featureblock                → "Summoner · <Circle>"
//	monster.champion.summoner.<circle>.advancement-features → "Summoner Champion · <Circle>"
//
// The champion form (SC-138) mirrors summonerProvenanceEyebrow's label for the
// base champion stat block, so the advancement page reads as the same entity.
// Returns "" for anything else.
func fbOrigin(scc string) string {
	_, rest, ok := strings.Cut(strings.TrimSpace(scc), "/")
	if !ok {
		return ""
	}
	typePath, _, _ := strings.Cut(rest, "/")
	seg := strings.Split(typePath, ".")
	switch {
	case len(seg) >= 4 && seg[0] == "monster" && seg[1] == "fixture" && seg[3] == "featureblock":
		return "Summoner · " + titleCase(seg[2])
	case len(seg) >= 5 && seg[0] == "monster" && seg[1] == "champion" &&
		seg[2] == "summoner" && seg[4] == "advancement-features":
		return "Summoner Champion · " + titleCase(seg[3])
	}
	return ""
}

// renderFeatureblockCard builds the contiguous (no blank-line) raw-HTML card so
// md_in_html passes it through verbatim.
func renderFeatureblockCard(doc fbDoc) string {
	name := strings.TrimSpace(doc.Name)
	if name == "" {
		name = "Featureblock"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<div class=\"fb-wrap\" data-role=\"%s\"", html.EscapeString(fbDataRole(doc)))
	if doc.Kind != "" {
		fmt.Fprintf(&b, " data-kind=\"%s\"", html.EscapeString(doc.Kind))
	}
	b.WriteString(">\n")
	b.WriteString("<article class=\"fb md-typeset\">\n")

	// head: shared 6-slot header. type+role → right-primary mini (role-colored),
	// Level → right-eyebrow chip, EV → right-deck chip; the rest of the loose stats
	// stay in the fb__stats body grid below.
	level := ""
	if doc.Level > 0 {
		level = fmt.Sprintf("Level %d", doc.Level)
	}
	b.WriteString(renderCardHead(cardHeadSlots{
		NameTag:      "h2",
		Class:        "fb__head", // re-attach the role-gradient band + centered diamond
		RoleKey:      fbDataRole(doc),
		LeftEyebrow:  hLine(html.EscapeString(fbKindNoun(doc))),
		LeftPrimary:  hLine(html.EscapeString(name)),
		LeftDeck:     hLine(html.EscapeString(strings.TrimSpace(doc.Eyebrow))),
		RightEyebrow: hChip(html.EscapeString(level)),
		RightPrimary: hMini(html.EscapeString(fbTypeRole(doc))),
		RightDeck:    hChip(html.EscapeString(fbEV(doc.Stats))),
	}))
	b.WriteString("\n")

	if f := strings.TrimSpace(doc.Flavor); f != "" {
		fmt.Fprintf(&b, "<div class=\"fb__flavor\">%s</div>\n", richInline(f))
	}

	b.WriteString(renderFbStats(doc.Stats))
	b.WriteString(renderFbFeats(doc.Features))

	b.WriteString("</article>\n")
	b.WriteString("</div>\n")
	return b.String()
}

// renderFbStats lays out the loose header stats ("EV: 2", "Stamina: 3 per
// square"). The grid-vs-ledger layout is a pure CSS reflow (data-fb-stats), so
// the markup is layout-agnostic: an ordered list of label/value cells.
func renderFbStats(stats []fbStat) string {
	if len(stats) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<div class=\"fb__stats\">\n")
	for _, st := range stats {
		fmt.Fprintf(&b,
			"<div class=\"fb__stat\"><div class=\"fb__stat-l\">%s</div><div class=\"fb__stat-v\">%s</div></div>\n",
			html.EscapeString(strings.TrimSpace(st.Name)), richInline(strings.TrimSpace(st.Value)))
	}
	b.WriteString("</div>\n")
	return b.String()
}

// fbIconAction maps a table-less feature's source emoji to an action accent so
// terrain's 🌀 Deactivate / ❕ Activate and malice passives don't all flatten to
// "passive" (spec §3). Mirrors ability-cards.js EMOJI_MAP, collapsed onto the
// action-accent vocabulary steel-featureblock.css colors. Icons are STRING
// literals matched with Contains — robust to the trailing U+FE0F variation
// selector book emoji carry (a rune-literal map would choke on those).
//
// An ORDERED slice, not a map: fbFeatureAction returns on the first Contains
// match, and Go's map iteration order is randomized per process, so a map here
// would make the picked action nondeterministic run-to-run for any icon string
// that happens to contain more than one of these glyphs (FOLLOWUPS #29). No
// current book content does — verified by scanning every "icon" field emitted
// across all four books' generated output — but the parser's icon regex
// (sbTitleRe) is permissive enough to allow it, so keep the priority fixed
// rather than relying on content never colliding.
var fbIconAction = []struct{ icon, action string }{
	{"🗡", "main"}, {"🏹", "main"}, {"❇", "main"},
	{"👤", "maneuver"},
	{"❗", "triggered"}, {"❕", "triggered"},
	{"⭐", "passive"},
	{"☠", "villain"},
	{"🌀", "special"},
}

// fbFeatureAction picks the [data-action] accent. Abilities with a usage word
// (or villain cost) route through sbActionKind exactly like statblock features;
// table-less features fall back to their icon emoji, then to "passive".
func fbFeatureAction(f fbFeature) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(f.Cost)), "villain action") {
		return "villain"
	}
	if strings.TrimSpace(f.Usage) != "" {
		action, _ := sbActionKind(f.Usage, f.Cost)
		return action
	}
	icon := strings.TrimSpace(f.Icon)
	for _, m := range fbIconAction {
		if strings.Contains(icon, m.icon) {
			return m.action
		}
	}
	return "passive"
}

// renderFbFeats renders the feature list. Features with Level == 0 render in the
// main flow; a contiguous run sharing a Level > 0 wraps in a .fb__band--adv with
// a "Level N Advancement" sub-head (fixture/retainer advancement tiers, spec §3).
// Fixture/featureblock data is document-ordered (base features first, then
// ascending advancement groups), so a single-pass state machine groups them.
func renderFbFeats(feats []fbFeature) string {
	if len(feats) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<div class=\"fb__feats\">\n")
	curLevel, bandOpen := 0, false
	seen := map[string]int{}
	for _, f := range feats {
		f.ID = featID(seen, f.Name)
		if f.Level != curLevel {
			if bandOpen {
				b.WriteString("</div>\n") // close previous .fb__band--adv
				bandOpen = false
			}
			curLevel = f.Level
			if curLevel > 0 {
				fmt.Fprintf(&b, "<div class=\"fb__band--adv\" data-level=\"%d\">\n", curLevel)
				fmt.Fprintf(&b, "<div class=\"fb__adv-head\">Level %d Advancement</div>\n", curLevel)
				bandOpen = true
			}
		}
		renderFbFeat(&b, f)
	}
	if bandOpen {
		b.WriteString("</div>\n")
	}
	b.WriteString("</div>\n")
	return b.String()
}

// fbBlankOrDash reports whether a featureblock cell is "empty" for display: blank
// or made up only of dash-like placeholder glyphs (hyphen, en/em dash, minus, …)
// that source tables use to mean "no value".
func fbBlankOrDash(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	for _, r := range s {
		switch r {
		case '-', '‐', '‑', '‒', '–', '—', '―', '−':
			// dash-like placeholder
		default:
			return false
		}
	}
	return true
}

// fbRailValue renders a kept rail cell: a blank-or-dash value collapses to a clean
// em-dash rather than a literal "-"; otherwise it defers to railValue.
func fbRailValue(s string) string {
	if fbBlankOrDash(s) {
		return "—"
	}
	return railValue(s)
}

// renderFbFeat writes one feature: article.sc-ability.fb__feat with the one-line
// head (icon · name · cost), reused ability-card internals (kw / rail / power
// roll / sections / enhancements), and the table-less body / trailing note.
func renderFbFeat(b *strings.Builder, f fbFeature) {
	fmt.Fprintf(b, "<article class=\"sc-ability fb__feat\" data-action=\"%s\">\n", fbFeatureAction(f))

	// head: shared 6-slot header (name + cost mini + usage chip). The fb__feat-head
	// wrapper is kept so the flat-list CSS hook still matches.
	crest := ""
	if ic := strings.TrimSpace(f.Icon); ic != "" {
		crest = fmt.Sprintf("<span class=\"fb__feat-icon\">%s</span>", html.EscapeString(ic))
	}
	b.WriteString("<div class=\"fb__feat-head\">")
	b.WriteString(renderCardHead(cardHeadSlots{
		Crest:        crest,
		LeftPrimary:  hLine(html.EscapeString(strings.TrimSpace(f.Name))),
		NameID:       f.ID,
		RightPrimary: hMini(html.EscapeString(strings.TrimSpace(f.Cost))),
		RightDeck:    hChip(richInline(strings.TrimSpace(f.Usage))),
	}))
	b.WriteString("</div>\n")

	// keyword chips — drop placeholder dashes ("-"/"—") and empties so usage-only
	// features (Field Ballista's Reload/Spot) don't render a chip of nothing.
	var kw []string
	for _, k := range f.Keywords {
		if !fbBlankOrDash(k) {
			kw = append(kw, k)
		}
	}
	if len(kw) > 0 {
		b.WriteString("<div class=\"sc-ability__kw\">")
		for _, k := range kw {
			fmt.Fprintf(b, "<span class=\"sc-ability__chip\">%s</span>", richInline(strings.TrimSpace(k)))
		}
		b.WriteString("</div>\n")
	}

	// distance / target rail — drop the whole row when BOTH are blank-or-dash;
	// if either carries a real value keep it, rendering the dash cell as an em-dash.
	if !fbBlankOrDash(f.Distance) || !fbBlankOrDash(f.Target) {
		b.WriteString("<div class=\"sc-ability__rail\">")
		fmt.Fprintf(b, "<div class=\"sc-ability__cell\"><div class=\"l\">Distance</div><div class=\"v\">%s</div></div>", fbRailValue(f.Distance))
		fmt.Fprintf(b, "<div class=\"sc-ability__cell\"><div class=\"l\">Targets</div><div class=\"v\">%s</div></div>", fbRailValue(f.Target))
		b.WriteString("</div>\n")
	}

	// SC-308: walk the ordered Effects list in exact document order — a named
	// section, a cost enhancement, bare prose, or a roll-only panel, each with
	// its tier panel (if any) rendered directly below/within it. No hoisting;
	// no special Intro/Body/Trailing casing — they are ordinary entries now
	// (.fb__feat-intro / .fb__feat-body / .fb__feat-trailing share the same
	// base CSS declaration, so the render class doesn't need to distinguish
	// "before" from "after" any more — see docs/site-builder.md).
	for _, e := range f.Effects {
		switch {
		case e.Name != "":
			b.WriteString("<div class=\"sc-ability__section\">")
			fmt.Fprintf(b, "<div class=\"sc-ability__section-head\"><span class=\"sc-ability__dia\"></span><span class=\"tag\">%s</span></div>", html.EscapeString(e.Name))
			fmt.Fprintf(b, "<div class=\"sc-ability__section-body\">%s</div>", renderSectionBlock(strings.TrimSpace(e.Effect)))
			if e.Roll != "" || e.Tier1 != "" || e.Tier2 != "" || e.Tier3 != "" {
				b.WriteString(fbEffectRollHTML(e))
			}
			b.WriteString("</div>\n")
		case e.Cost != "":
			fmt.Fprintf(b, "<div class=\"sc-ability__enh\"><span class=\"cost\">%s</span><span class=\"txt\">%s</span></div>\n",
				html.EscapeString(strings.TrimSpace(e.Cost)), richInline(strings.TrimSpace(e.Effect)))
			if e.Roll != "" || e.Tier1 != "" || e.Tier2 != "" || e.Tier3 != "" {
				b.WriteString(fbEffectRollHTML(e))
			}
		case e.Effect != "":
			fmt.Fprintf(b, "<div class=\"fb__feat-trailing\">%s</div>\n", richInline(strings.TrimSpace(e.Effect)))
			if e.Roll != "" || e.Tier1 != "" || e.Tier2 != "" || e.Tier3 != "" {
				b.WriteString(fbEffectRollHTML(e))
			}
		default:
			b.WriteString(fbEffectRollHTML(e))
		}
	}

	b.WriteString("</article>\n")
}

// fbEffectRollHTML renders one effects[] entry's tier panel: an optional
// "Power Roll <formula>" head, or — for a header-less list — a head derived
// at RENDER time from THIS entry's own Effect prose (content.DeriveTestLabel;
// e.g. "Agility Test"), or no head at all if neither applies. e.Roll holds
// the full header text ("Power Roll + 3") or a bare dice formula
// ("2d10 + R"); the "Power Roll " prefix (present only for the headered form)
// is stripped so the literal "Power Roll" label and the formula/dice always
// render as separate spans, matching the pre-existing head markup. Reuses
// tierGlyph / tierKey (ability_cards.go).
func fbEffectRollHTML(e fbEffect) string {
	var b strings.Builder
	b.WriteString("<div class=\"sc-ability__pr\">")
	switch formula := strings.TrimPrefix(strings.TrimSpace(e.Roll), "Power Roll "); {
	case formula != "":
		fmt.Fprintf(&b, "<div class=\"sc-ability__pr-head\"><span class=\"sc-ability__dia\"></span><span class=\"pre\">Power Roll</span><span class=\"chars\">%s</span></div>", richInline(formula))
	default:
		if label := content.DeriveTestLabel(e.Effect); label != "" {
			fmt.Fprintf(&b, "<div class=\"sc-ability__pr-head\"><span class=\"sc-ability__dia\"></span><span class=\"pre\">%s</span></div>", html.EscapeString(label))
		}
	}
	b.WriteString("<div class=\"sc-ability__pr-rows\">")
	for i, v := range []string{e.Tier1, e.Tier2, e.Tier3} {
		if v = strings.TrimSpace(v); v != "" {
			fmt.Fprintf(&b, "<div class=\"sc-ability__tier\" data-tier=\"%s\"><span class=\"badge\">%s</span><span class=\"res\">%s</span></div>",
				tierKey[i], tierGlyph[i], richInline(v))
		}
	}
	b.WriteString("</div></div>\n")
	return b.String()
}
