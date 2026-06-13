package site

// High-Fantasy Steel FIXTURE pages for the Steel Compendium MkDocs site.
//
// Summoner fixtures (The Boil, Barrow Gates, …) are `type: statblock` +
// `statblock_kind: fixture` (stamped by applyFixtureGrid in the data layer,
// Plan 1). In ANATOMY they are featureblocks, not creature statblocks: a loose
// Stamina/Size header + a list of features, several gated behind "Level N
// Fixture Advancement Feature" tiers. So rather than the creature JSON island
// (steel-statblock.js), they route HERE and render as the same `.fb-wrap`
// Forged Band card that featureblocks/terrain use (renderFeatureblockCard),
// via a statblock→fbDoc adapter.
//
// SITE-ONLY: runs inside `steel-etl site` against the generated md-linked pages;
// the shared data repos are never touched. Feature blockquotes are parsed by the
// shared content.ParseRichFeatures (icon-keeping, raw .md links, advancement-
// level attachment) — the SAME parser that builds featureblock/terrain
// features[] frontmatter — so feature internals are identical across the three
// fb content types. The fixture's stamina/size frontmatter (Plan 1's grid parse)
// becomes the loose header stats.
//
// Plan 3 of the featureblock effort; spec
// docs/superpowers/specs/2026-06-12-featureblock-cards-design.md.

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// buildFixturePage rewrites a `type: statblock` + `statblock_kind: fixture` page
// body into the .fb-wrap card. Returns (newData, true) when handled; (data,
// false) otherwise so the caller writes the page unchanged. Frontmatter is
// preserved verbatim; injectH1 (next in buildSection) prepends the "# Name".
func buildFixturePage(data []byte) ([]byte, bool) {
	fm, body := splitFrontmatter(string(data))
	if strings.TrimSpace(parseFrontmatterField(fm, "type")) != "statblock" {
		return data, false
	}
	if strings.TrimSpace(parseFrontmatterField(fm, "statblock_kind")) != "fixture" {
		return data, false
	}
	doc := fbDoc{
		Name:        strings.TrimSpace(parseFrontmatterField(fm, "name")),
		Type:        "statblock",
		Role:        strings.TrimSpace(parseFrontmatterField(fm, "role")),
		TerrainType: strings.TrimSpace(parseFrontmatterField(fm, "terrain_type")),
		Stats:       fixtureStats(fm),
		Features:    fbFeaturesFromRich(content.ParseRichFeatures(body)),
	}
	card := renderFeatureblockCard(doc)
	return []byte("---\n" + fm + "\n---\n\n" + card), true
}

// fixtureStats builds the loose header from the fixture 2-col grid fields
// (Stamina, Size — the only stats applyFixtureGrid emits), in source order,
// omitting any that are empty.
func fixtureStats(fm string) []fbStat {
	var out []fbStat
	if v := strings.TrimSpace(parseFrontmatterField(fm, "stamina")); v != "" {
		out = append(out, fbStat{Name: "Stamina", Value: v})
	}
	if v := strings.TrimSpace(parseFrontmatterField(fm, "size")); v != "" {
		out = append(out, fbStat{Name: "Size", Value: v})
	}
	return out
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
			Body:     r.Body,
			Trailing: r.Trailing,
			Level:    r.Level,
		}
		if r.PowerRoll != nil {
			f.PowerRoll = &fbPowerRoll{Formula: r.PowerRoll.Formula, Tiers: r.PowerRoll.Tiers}
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
