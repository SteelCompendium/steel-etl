package site

// High-Fantasy Steel STATBLOCK PREVIEW cards — a compact variant of the full
// .sb-wrap card (statblock_card.go) used on index / group-landing pages. It
// reuses the full card's header / defenses / meta / chars zone renderers and
// adds a one-line-per-feature preview. The four optional zones are gated by the
// grid-level data-sbprev-* attributes (sbCardsOpen + the per-page toggle bar in
// statblock-preview.js); the header is always shown. SITE-ONLY (like cards.go):
// all data comes from the page's frontmatter + body via buildStatblockIsland.

import "strings"

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
