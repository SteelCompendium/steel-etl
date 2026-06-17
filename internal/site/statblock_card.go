package site

// High-Fantasy Steel STATBLOCK cards rendered at BUILD TIME.
//
// renderStatblockCard is a 1:1 Go port of v2/docs/javascripts/steel-statblock.js's
// render()/renderFeature()/… It emits the same .sb-wrap DOM the client script used
// to build, so steel-statblock.css and the data-sb-* preference system are unchanged
// and the slimmed client script only wires interactivity (collapsible bands + sticky
// header). Equivalence is locked by TestStatblockCard_GoldenEquivalence against HTML
// captured from the (previous) JS renderer. See the workspace-root spec
// docs/superpowers/specs/2026-06-14-statblock-build-time-render-design.md.
//
// The input sbIsland and its parsing (buildStatblockIsland, statblock_page.go) are
// unchanged — this file replaces only the JSON-island output stage.
//
// DELIBERATE DIVERGENCE from the ability_cards.go family (sbEsc/richSb/sbCostBadge
// vs html.EscapeString/richInline/costBadge): do NOT "unify" them. These mirror the
// JS renderer's esc()/rich() exactly — sbEsc escapes only & < > " (NOT ', which
// html.EscapeString would), and richSb keeps the already-resolved href verbatim
// (richInline re-runs cardHref). Equality with the captured golden depends on it.

import (
	"fmt"
	"regexp"
	"strings"
)

type sbAct struct{ glyph, label string }

// sbACT mirrors the JS ACT map (DrawSteelGlyphs placeholder + eyebrow label).
var sbACT = map[string]sbAct{
	"main":      {"l", "Main Action"},
	"maneuver":  {"f", "Maneuver"},
	"triggered": {")", "Triggered Action"},
	"move":      {"o", "Move Action"},
	"passive":   {"*", "Trait"},
	"villain":   {"*", "Villain Action"},
	"malice":    {"*", "Malice"},
}

var sbTierGlyph = map[string]string{"low": "!", "mid": "@", "high": "#"}

var (
	// JS rich() link regex (runs on already-escaped text): label = non-"]",
	// href = non-")"/non-space. Matched verbatim so the port is faithful.
	sbRichLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	// JS rich() bold regex: lazy inner match.
	sbRichBoldRe = regexp.MustCompile(`\*\*(.+?)\*\*`)
	// costBadge leading-count split. The leading `\s*` deliberately differs from
	// ability_cards.go's costNumRe (`^(\d+)…`) to match the JS costBadge regex
	// (`/^\s*(\d+)\s+(.*)$/`) — keep it for golden equality.
	sbCostNumRe = regexp.MustCompile(`^\s*(\d+)\s+(.*)$`)
)

// sbEsc matches the JS esc(): escape & < > " (NOT '). Single left-to-right pass,
// like the JS regex replace, so an inserted "amp;" is never re-processed.
func sbEsc(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

// richSb matches the JS rich(): esc, then [text](href) → sb-term anchor, then
// **bold** → <b>. Links are ALREADY resolved (resolveSbLinks in the parse stage),
// so the href passes through unchanged — no cardHref here.
func richSb(s string) string {
	s = sbEsc(s)
	s = sbRichLinkRe.ReplaceAllString(s, `<a class="sb-term" href="${2}">${1}</a>`)
	s = sbRichBoldRe.ReplaceAllString(s, `<b>${1}</b>`)
	return s
}

// sbCostBadge ports costBadge(): a leading integer gets the mono .num treatment.
func sbCostBadge(cost string) string {
	if cost == "" {
		return ""
	}
	var inner string
	if m := sbCostNumRe.FindStringSubmatch(cost); m != nil {
		inner = `<span class="num">` + sbEsc(m[1]) + `</span> ` + richSb(m[2])
	} else {
		inner = richSb(cost)
	}
	return `<div class="sc-ability__cost">` + inner + `</div>`
}

// renderStatblockSpecField ports specField(): one CSS-reflowable field cell.
func renderStatblockSpecField(mod, label, valueHTML string) string {
	return `<div class="sb__field sb__field--` + mod + `"><span class="sb__field-l">` +
		sbEsc(label) + `</span><span class="sb__field-v">` + valueHTML + `</span></div>`
}

// renderStatblockFeature ports renderFeature(): the flattened steel feature article.
func renderStatblockFeature(f sbFeature) string {
	a, ok := sbACT[f.Action]
	if !ok {
		a = sbACT["passive"]
	}
	const dia = `<span class="sc-ability__dia"></span>`
	var b strings.Builder
	fmt.Fprintf(&b, `<article class="sc-ability sb__feat" data-action="%s" data-kind="%s">`, sbEsc(f.Action), sbEsc(f.Kind))

	// head: crest + inline icon (CSS shows one) · (eyebrow=usage) name · cost
	b.WriteString(`<div class="sb__feat-head">`)
	b.WriteString(`<span class="sc-crest sb__feat-crest"><span class="sb__feat-glyph">` + a.glyph + `</span></span>`)
	b.WriteString(`<span class="sb__feat-icon"><span class="sb__feat-glyph">` + a.glyph + `</span></span>`)
	b.WriteString(`<div class="sb__feat-titles">`)
	eyebrow := f.Usage
	if eyebrow == "" && f.Kind == "passive" {
		eyebrow = "Trait"
	}
	if eyebrow != "" {
		b.WriteString(`<div class="sb__feat-eyebrow">` + dia + richSb(eyebrow) + `</div>`)
	}
	b.WriteString(`<h3 class="sb__feat-name sc-ability__name">` + richSb(f.Name) + `</h3>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="sb__feat-corner">` + sbCostBadge(f.Cost) + `</div>`)
	b.WriteString(`</div>`)

	// passive / malice → plain body paragraph, done.
	if f.Body != "" {
		b.WriteString(`<p class="sb__feat-body">` + richSb(f.Body) + `</p>`)
		b.WriteString(`</article>`)
		return b.String()
	}

	// keyword + usage block
	if len(f.Keywords) > 0 || f.Usage != "" {
		b.WriteString(`<div class="sb__ku">`)
		if len(f.Keywords) > 0 {
			var chips strings.Builder
			for _, k := range f.Keywords {
				chips.WriteString(`<span class="sc-ability__chip">` + sbEsc(k) + `</span>`)
			}
			b.WriteString(renderStatblockSpecField("kw", "Keywords", chips.String()))
		}
		if f.Usage != "" {
			b.WriteString(renderStatblockSpecField("usage", "Action", richSb(f.Usage)))
		}
		b.WriteString(`</div>`)
	}

	// distance + target block
	if f.Distance != "" || f.Target != "" {
		dist, tgt := f.Distance, f.Target
		if dist == "" {
			dist = "—"
		}
		if tgt == "" {
			tgt = "—"
		}
		b.WriteString(`<div class="sb__dt">`)
		b.WriteString(renderStatblockSpecField("dist", "Distance", richSb(dist)))
		b.WriteString(renderStatblockSpecField("tgt", "Target", richSb(tgt)))
		b.WriteString(`</div>`)
	}

	// power roll
	if f.PowerRoll != nil {
		b.WriteString(`<div class="sc-ability__pr">`)
		if f.PowerRoll.Formula != "" {
			b.WriteString(`<div class="sc-ability__pr-head">` + dia +
				`<span class="pre">Power Roll</span><span class="chars">` + sbEsc(f.PowerRoll.Formula) + `</span></div>`)
		}
		b.WriteString(`<div class="sc-ability__pr-rows">`)
		for _, t := range []string{"low", "mid", "high"} {
			if v, ok := f.PowerRoll.Tiers[t]; ok { // map only holds non-empty tiers; mirrors JS != null
				b.WriteString(`<div class="sc-ability__tier" data-tier="` + t + `"><span class="badge">` +
					sbTierGlyph[t] + `</span><span class="res">` + richSb(v) + `</span></div>`)
			}
		}
		b.WriteString(`</div></div>`)
	}

	// sections (Trigger / Effect / Special)
	for _, s := range f.Sections {
		b.WriteString(`<div class="sc-ability__section"><div class="sc-ability__section-head">` + dia +
			`<span class="tag">` + richSb(s.Label) + `</span></div><div class="sc-ability__section-body"><p>` +
			richSb(s.Text) + `</p></div></div>`)
	}

	// trailing note
	if f.Trailing != "" {
		b.WriteString(`<p class="sb__feat-trailing">` + richSb(f.Trailing) + `</p>`)
	}

	// enhancements (spend X rows)
	for _, e := range f.Enhancements {
		b.WriteString(`<div class="sc-ability__enh"><span class="cost">` + richSb(e.Cost) +
			`</span><span class="txt">` + richSb(e.Text) + `</span></div>`)
	}

	b.WriteString(`</article>`)
	return b.String()
}

// renderStatblockBand ports band(): a collapsible Villain/Malice section. Emitted
// open (data-open="true"); the slim client script toggles it.
func renderStatblockBand(kind, title, glyph, introHTML, featuresHTML string) string {
	return `<section class="sb__band sb__band--` + kind + `" data-open="true">` +
		`<button type="button" class="sb__band-head" aria-expanded="true">` +
		`<span class="sc-crest sb__band-crest"><span class="sb__band-glyph">` + glyph + `</span></span>` +
		`<span class="sb__band-title">` + sbEsc(title) + `</span>` +
		`<span class="sb__band-chev" aria-hidden="true">▾</span>` +
		`</button>` +
		`<div class="sb__band-body">` + introHTML + featuresHTML + `</div>` +
		`</section>`
}

func sbMetaCell(label, value string) string {
	return `<div class="sb__field sb__field--meta"><span class="sb__field-l">` + sbEsc(label) +
		`</span><span class="sb__field-v">` + richSb(value) + `</span></div>`
}

// renderStatblockMeta ports renderMeta(): the fixed 2×2 secondary stats.
func renderStatblockMeta(m sbMeta) string {
	return `<div class="sb__meta">` +
		sbMetaCell("Immunity", m.Immunity) +
		sbMetaCell("Weakness", m.Weakness) +
		sbMetaCell("Movement", m.Movement) +
		sbMetaCell(m.Captain.Label, m.Captain.Value) +
		`</div>`
}

// renderStatblockChars ports renderChars().
func renderStatblockChars(list []sbChar) string {
	var b strings.Builder
	b.WriteString(`<div class="sb__chars">`)
	for _, c := range list {
		b.WriteString(`<div class="sb__char"><span class="sb__char-box">` + sbEsc(c.K) +
			`</span><span class="sb__char-v">` + sbEsc(c.V) +
			`</span><span class="sb__char-l">` + sbEsc(c.L) + `</span></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// renderStatblockSticky ports renderSticky(): the mini-header node (present in
// markup; revealed by the client script on scroll).
func renderStatblockSticky(d sbIsland) string {
	var defs strings.Builder
	for _, x := range d.Defenses {
		defs.WriteString(`<span class="m"><b>` + sbEsc(x.V) + `</b>` + sbEsc(x.L) + `</span>`)
	}
	var chars strings.Builder
	for _, c := range d.Characteristics {
		chars.WriteString(`<span class="c"><b>` + sbEsc(c.V) + `</b><i>` + sbEsc(c.K) + `</i></span>`)
	}
	metaPairs := [][2]string{
		{"Movement", d.Meta.Movement},
		{d.Meta.Captain.Label, d.Meta.Captain.Value},
		{"Immunity", d.Meta.Immunity},
		{"Weakness", d.Meta.Weakness},
	}
	var meta strings.Builder
	for _, kv := range metaPairs {
		meta.WriteString(`<span class="sm"><b>` + sbEsc(kv[0]) + `</b>` + sbEsc(kv[1]) + `</span>`)
	}
	return `<div class="sb__sticky" aria-hidden="true">` +
		`<div class="sb__sticky-row1">` +
		`<span class="sb__sticky-id"><span class="sb__sticky-name">` + sbEsc(d.Name) + `</span>` +
		`<span class="sb__sticky-role" data-role="` + sbEsc(d.RoleKey) + `">` + sbEsc(d.Role) + `</span></span>` +
		`<span class="sb__sticky-stats"><span class="sb__sticky-defs">` + defs.String() + `</span>` +
		`<span class="sb__sticky-chars">` + chars.String() + `</span></span>` +
		`</div>` +
		`<div class="sb__sticky-row2">` + meta.String() + `</div>` +
		`</div>`
}

// renderStatblockHead emits the .sb__head identity band (ancestry/name on the
// left, level/role/EV on the right). Shared by the full card and the preview
// card (statblock_preview.go) so the header looks identical in both.
func renderStatblockHead(d sbIsland) string {
	ev := ""
	if strings.TrimSpace(d.EV) != "" {
		ev = `<div class="sb__ev">EV ` + sbEsc(d.EV) + `</div>`
	} else if strings.TrimSpace(d.Cost) != "" {
		// Summoner-book statblocks bought with a gametime resource rather than EV
		// (e.g. "3 essence for two minions") carry a generic cost; render it in the
		// EV slot without the "EV" prefix.
		ev = `<div class="sb__ev sb__cost">` + sbEsc(d.Cost) + `</div>`
	}
	// Summoner minions/champions carry no level; omit the "Level" line entirely
	// rather than render a bare "Level " label. (Every monster statblock has a
	// level, so this only affects the Summoner book.)
	level := ""
	if strings.TrimSpace(d.Level) != "" {
		level = `<div class="sb__level">Level ` + sbEsc(d.Level) + `</div>`
	}
	return `<header class="sb__head"><div class="sb__head-row">` +
		`<div class="sb__identity"><div class="sb__kw">` + sbEsc(d.Ancestry) + `</div>` +
		`<h2 class="sb__name">` + sbEsc(d.Name) + `</h2></div>` +
		`<div class="sb__class">` + level +
		`<div class="sb__role" data-role="` + sbEsc(d.RoleKey) + `">` + sbEsc(d.Role) + `</div>` +
		ev + `</div></div></header>`
}

// renderStatblockDefenses emits the .sb__defenses stat row (Size/Speed/Stamina/
// Stability/Free Strike). Shared by the full card and the preview card.
func renderStatblockDefenses(defenses []sbLV) string {
	var defs strings.Builder
	for _, x := range defenses {
		defs.WriteString(`<div class="sb__stat"><span class="v">` + sbEsc(x.V) +
			`</span><span class="l">` + sbEsc(x.L) + `</span></div>`)
	}
	return `<div class="sb__defenses">` + defs.String() + `</div>`
}

// renderStatblockCard ports render(): the full .sb-wrap card. Villain-kind
// features group into a collapsible "Villain Actions" band, matching the JS.
// The shared family Malice band stays omitted (not in island data; FOLLOWUPS #7).
func renderStatblockCard(d sbIsland) string {
	var feat, villain strings.Builder
	for _, f := range d.Features {
		if f.Kind == "villain" {
			villain.WriteString(renderStatblockFeature(f))
		} else {
			feat.WriteString(renderStatblockFeature(f))
		}
	}
	villainHTML := ""
	if villain.Len() > 0 {
		villainHTML = renderStatblockBand("villain", "Villain Actions", sbACT["villain"].glyph, "", villain.String())
	}

	var b strings.Builder
	b.WriteString(`<div class="sb-wrap" data-role="` + sbEsc(d.RoleKey) + `" data-creature="` + sbEsc(d.ID) + `">`)
	b.WriteString(renderStatblockSticky(d))
	b.WriteString(`<article class="sb md-typeset" data-role="` + sbEsc(d.RoleKey) + `">`)
	b.WriteString(renderStatblockHead(d))
	b.WriteString(renderStatblockDefenses(d.Defenses))
	b.WriteString(renderStatblockMeta(d.Meta))
	b.WriteString(renderStatblockChars(d.Characteristics))
	b.WriteString(`<div class="sb__features">` + feat.String() + villainHTML + `</div>`)
	b.WriteString(`</article></div>`)
	return b.String()
}
