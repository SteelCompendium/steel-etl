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
type fbFeature struct {
	Icon         string       `yaml:"icon"`
	Name         string       `yaml:"name"`
	Cost         string       `yaml:"cost"`
	Usage        string       `yaml:"usage"`
	Keywords     []string     `yaml:"keywords"`
	Distance     string       `yaml:"distance"`
	Target       string       `yaml:"target"`
	PowerRoll    *fbPowerRoll `yaml:"power_roll"`
	Sections     []fbSection  `yaml:"sections"`
	Enhancements []fbEnh      `yaml:"enhancements"`
	Body         string       `yaml:"body"`
	Trailing     string       `yaml:"trailing"`
	Level        int          `yaml:"level"`
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
	card := renderFeatureblockCard(doc)
	return []byte("---\n" + fm + "\n---\n\n" + card), true
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

// fbEyebrow composes the head eyebrow line: "Level N <TerrainType> · <Role>" for
// terrain/fixtures, else "Malice Features" / "Features".
func fbEyebrow(doc fbDoc) string {
	if doc.TerrainType != "" {
		s := doc.TerrainType
		if doc.Level > 0 {
			s = fmt.Sprintf("Level %d %s", doc.Level, doc.TerrainType)
		}
		if r := strings.TrimSpace(doc.Role); r != "" {
			s += " · " + r
		}
		return s
	}
	if doc.Kind == "malice" {
		return "Malice Features"
	}
	return "Features"
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

	// head: eyebrow + name
	b.WriteString("<header class=\"fb__head\">\n")
	fmt.Fprintf(&b, "<div class=\"fb__eyebrow\">%s</div>\n", html.EscapeString(fbEyebrow(doc)))
	fmt.Fprintf(&b, "<h2 class=\"fb__name\">%s</h2>\n", html.EscapeString(name))
	b.WriteString("</header>\n")

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
// action-accent vocabulary steel-featureblock.css colors. Keys are STRING
// literals matched with Contains — robust to the trailing U+FE0F variation
// selector book emoji carry (a rune-literal map would choke on those).
var fbIconAction = map[string]string{
	"🗡": "main", "🏹": "main", "❇": "main",
	"👤": "maneuver",
	"❗": "triggered", "❕": "triggered",
	"⭐": "passive",
	"☠": "villain",
	"🌀": "special",
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
	for k, a := range fbIconAction {
		if strings.Contains(icon, k) {
			return a
		}
	}
	return "passive"
}

// renderFbFeats renders the feature list. Each feature is article.sc-ability so
// it inherits steel-ability-cards.css internals; the one-line head (icon · name
// · cost) replaces the ability card's crest/eyebrow ceremony.
func renderFbFeats(feats []fbFeature) string {
	if len(feats) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<div class=\"fb__feats\">\n")
	for _, f := range feats {
		fmt.Fprintf(&b, "<article class=\"sc-ability fb__feat\" data-action=\"%s\">\n", fbFeatureAction(f))

		// head: icon · name · cost
		b.WriteString("<div class=\"fb__feat-head\">")
		if ic := strings.TrimSpace(f.Icon); ic != "" {
			fmt.Fprintf(&b, "<span class=\"fb__feat-icon\">%s</span>", html.EscapeString(ic))
		}
		fmt.Fprintf(&b, "<h3 class=\"fb__feat-name sc-ability__name\">%s</h3>", html.EscapeString(strings.TrimSpace(f.Name)))
		fmt.Fprintf(&b, "<div class=\"fb__feat-corner\">%s</div>", costBadge(strings.TrimSpace(f.Cost)))
		b.WriteString("</div>\n")

		// keyword chips (reused ability-card grammar)
		if len(f.Keywords) > 0 {
			b.WriteString("<div class=\"sc-ability__kw\">")
			for _, k := range f.Keywords {
				fmt.Fprintf(&b, "<span class=\"sc-ability__chip\">%s</span>", richInline(strings.TrimSpace(k)))
			}
			b.WriteString("</div>\n")
		}

		// distance / target rail
		if strings.TrimSpace(f.Distance) != "" || strings.TrimSpace(f.Target) != "" {
			b.WriteString("<div class=\"sc-ability__rail\">")
			fmt.Fprintf(&b, "<div class=\"sc-ability__cell\"><div class=\"l\">Distance</div><div class=\"v\">%s</div></div>", railValue(f.Distance))
			fmt.Fprintf(&b, "<div class=\"sc-ability__cell\"><div class=\"l\">Targets</div><div class=\"v\">%s</div></div>", railValue(f.Target))
			b.WriteString("</div>\n")
		}

		// power roll
		if f.PowerRoll != nil {
			b.WriteString(fbPowerRollHTML(*f.PowerRoll))
		}

		// titled sections (Effect / Trigger / Special …)
		for _, s := range f.Sections {
			b.WriteString("<div class=\"sc-ability__section\">")
			if l := strings.TrimSpace(s.Label); l != "" {
				fmt.Fprintf(&b, "<div class=\"sc-ability__section-head\"><span class=\"sc-ability__dia\"></span><span class=\"tag\">%s</span></div>", html.EscapeString(l))
			}
			fmt.Fprintf(&b, "<div class=\"sc-ability__section-body\">%s</div>", renderSectionBlock(strings.TrimSpace(s.Text)))
			b.WriteString("</div>\n")
		}

		// cost enhancements (2 Malice / Spend …)
		for _, e := range f.Enhancements {
			fmt.Fprintf(&b, "<div class=\"sc-ability__enh\"><span class=\"cost\">%s</span><span class=\"txt\">%s</span></div>\n",
				html.EscapeString(strings.TrimSpace(e.Cost)), richInline(strings.TrimSpace(e.Text)))
		}

		// table-less prose body / post-table trailing note
		if body := strings.TrimSpace(f.Body); body != "" {
			fmt.Fprintf(&b, "<div class=\"fb__feat-body\">%s</div>\n", richInline(body))
		}
		if tr := strings.TrimSpace(f.Trailing); tr != "" {
			fmt.Fprintf(&b, "<div class=\"fb__feat-trailing\">%s</div>\n", richInline(tr))
		}

		b.WriteString("</article>\n")
	}
	b.WriteString("</div>\n")
	return b.String()
}

// fbPowerRollHTML renders the steel power-roll panel: an optional
// "Power Roll <formula>" head (omitted for a bare test, where formula is "")
// followed by the glyph-badged tier rows. Reuses tierGlyph / tierKey
// (ability_cards.go). Unlike the ability card's tierPanelHTML (which hardcodes
// "Power Roll +" before the characteristics), this prints the stored formula
// verbatim — it already carries its sign ("+ 2") or full dice ("2d10 + R").
func fbPowerRollHTML(pr fbPowerRoll) string {
	var b strings.Builder
	b.WriteString("<div class=\"sc-ability__pr\">")
	if f := strings.TrimSpace(pr.Formula); f != "" {
		fmt.Fprintf(&b, "<div class=\"sc-ability__pr-head\"><span class=\"sc-ability__dia\"></span><span class=\"pre\">Power Roll</span><span class=\"chars\">%s</span></div>", richInline(f))
	}
	b.WriteString("<div class=\"sc-ability__pr-rows\">")
	for i := 0; i < 3; i++ {
		if v := strings.TrimSpace(pr.Tiers[tierKey[i]]); v != "" {
			fmt.Fprintf(&b, "<div class=\"sc-ability__tier\" data-tier=\"%s\"><span class=\"badge\">%s</span><span class=\"res\">%s</span></div>",
				tierKey[i], tierGlyph[i], richInline(v))
		}
	}
	b.WriteString("</div></div>\n")
	return b.String()
}
