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
// md_in_html passes it through verbatim. Features land in Task 3.
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

// renderFbStats / renderFbFeats are filled in Tasks 2 and 3.
func renderFbStats(stats []fbStat) string   { return "" }
func renderFbFeats(feats []fbFeature) string { return "" }
