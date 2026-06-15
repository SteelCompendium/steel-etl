package site

// High-Fantasy Steel STATBLOCK PREVIEW cards — a compact variant of the full
// .sb-wrap card (statblock_card.go) used on index / group-landing pages. It
// reuses the full card's header / defenses / meta / chars zone renderers and
// adds a one-line-per-feature preview. The four optional zones are gated by the
// grid-level data-sbprev-* attributes (sbCardsOpen + the per-page toggle bar in
// statblock-preview.js); the header is always shown. SITE-ONLY (like cards.go):
// all data comes from the page's frontmatter + body via buildStatblockIsland.

import (
	"html"
	"strings"
)

// statblockFeatureCache maps a statblock's scc code → its parsed features,
// populated at page-transform time (buildStatblockIslandPage) when the source
// blockquote body is still available. The group-landing assembler reads leaf
// pages AFTER they've been transformed to .sb-wrap HTML (their blockquote
// features are gone), so the preview card recovers features from this cache by
// scc. Build-scoped: Build() resets it at the start of each site build. Reads
// of a missing key fall back to whatever buildStatblockIsland parsed from the
// body (correct for unit tests that pass a blockquote body directly).
var statblockFeatureCache = map[string][]sbFeature{}

// renderStatblockFeatureLine emits one compact feature row: action glyph + name
// + usage eyebrow + cost. No body/keywords/power-roll (those belong on the full
// page). Links are stripped to plain text (linkText) so the line carries no
// nested anchors inside the whole-card stretched link.
func renderStatblockFeatureLine(f sbFeature) string {
	a, ok := sbACT[f.Action]
	if !ok {
		a = sbACT["passive"]
	}
	var b strings.Builder
	b.WriteString(`<li class="sb-prev__feat" data-action="` + sbEsc(f.Action) + `">`)
	b.WriteString(`<span class="sb-prev__feat-icon"><span class="sb__feat-glyph">` + a.glyph + `</span></span>`)
	b.WriteString(`<span class="sb-prev__feat-name">` + sbEsc(linkText(f.Name)) + `</span>`)
	usage := f.Usage
	if usage == "" && f.Kind == "passive" {
		usage = "Trait"
	}
	if usage != "" {
		b.WriteString(`<span class="sb-prev__feat-usage">` + sbEsc(linkText(usage)) + `</span>`)
	}
	if f.Cost != "" {
		b.WriteString(`<span class="sb-prev__feat-cost">` + sbEsc(linkText(f.Cost)) + `</span>`)
	}
	b.WriteString(`</li>`)
	return b.String()
}

// sbPreviewDefaults is the build-time baseline visibility for the four preview
// zones — the no-JS fallback. SINGLE SOURCE OF TRUTH for the default; the
// settings drawer (global pref) and per-page toggle bar refine it live. Mirror
// any change in v2 settings-core.js SBPREV_DEFAULTS and overrides/main.html.
var sbPreviewDefaults = [][2]string{
	{"stats", "on"}, {"meta", "off"}, {"chars", "off"}, {"feats", "off"},
}

// sbCardsOpen writes the opening tag of a statblock-preview grid, baking the
// default zone-visibility attributes onto the container. statblock-preview.js
// later overrides these from the global pref / per-page bar.
func sbCardsOpen() string {
	var b strings.Builder
	b.WriteString(`<div class="sb-cards"`)
	for _, kv := range sbPreviewDefaults {
		b.WriteString(` data-sbprev-` + kv[0] + `="` + kv[1] + `"`)
	}
	b.WriteString(">\n")
	return b.String()
}

// renderStatblockPreviewCard renders an sbIsland as a compact .sb-prev mini-card
// linking to the full statblock page (href, a relative ".md" path resolved by
// dirURL). source is an optional provenance label ("Summoner") shown as a chip;
// "" emits none. The header is always present; the four zones below it are
// gated by the grid's data-sbprev-* attributes (sbCardsOpen).
func renderStatblockPreviewCard(d sbIsland, href, source string) string {
	var b strings.Builder
	b.WriteString(`<div class="sb-wrap sb-prev" data-role="` + sbEsc(d.RoleKey) +
		`" data-creature="` + sbEsc(d.ID) + `">`)
	b.WriteString(`<a class="sb-prev__link" href="` + html.EscapeString(dirURL(href)) +
		`" aria-label="` + html.EscapeString(d.Name) + `"></a>`)
	b.WriteString(`<article class="sb sb-prev__body md-typeset" data-role="` + sbEsc(d.RoleKey) + `">`)
	if source != "" {
		b.WriteString(`<div class="sb-prev__src">` + sbEsc(source) + `</div>`)
	}
	b.WriteString(renderStatblockHead(d))
	b.WriteString(renderStatblockDefenses(d.Defenses))
	b.WriteString(renderStatblockMeta(d.Meta))
	b.WriteString(renderStatblockChars(d.Characteristics))
	if len(d.Features) > 0 {
		b.WriteString(`<ul class="sb-prev__feats">`)
		for _, f := range d.Features {
			b.WriteString(renderStatblockFeatureLine(f))
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</article></div>`)
	return b.String()
}
