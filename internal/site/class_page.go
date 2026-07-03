package site

// Class landing header + jump bar (.sc-classhead / .sc-classnav).
// buildClassLandingPage prepends a renderCardHead-based header card and an
// anchor nav (from the body's ## sections) to every `type: class` Browse page —
// those pages are ~80,000px of book-order prose and previously opened with
// nothing but the intro paragraph. SITE-ONLY: the shared data repos are
// untouched. Styled by v2/docs/stylesheets/steel-class.css. See workspace
// docs/superpowers/plans/2026-07-01-p3-class-landing-header.md.

import (
	"html"
	"regexp"
	"strings"
)

var (
	attrListRe = regexp.MustCompile(`\s*\{[^{}]*\}\s*$`)
	slugDropRe = regexp.MustCompile(`[^a-z0-9 _-]`)
	// markdown links are unwrapped via mdLinkRe (ability_cards.go).
)

// headingText extracts the display text of a "## Heading" line: strips the
// hash prefix, a trailing {attr-list} (RenderSubtree's data-scc stamps), and
// unwraps markdown links.
func headingText(line string) string {
	s := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
	s = attrListRe.ReplaceAllString(s, "")
	s = mdLinkRe.ReplaceAllString(s, "$1")
	return strings.TrimSpace(s)
}

// pySlugify replicates python-markdown's default toc slugify so our anchor
// hrefs match the heading ids MkDocs generates: lowercase, drop everything but
// [a-z0-9 _-], collapse whitespace runs to single hyphens.
func pySlugify(s string) string {
	s = strings.ToLower(s)
	s = slugDropRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), "-")
}

// buildClassLandingPage rewrites a `type: class` page to open with a
// .sc-classhead card (shared 6-slot head) + a .sc-classnav jump bar over the
// body's ## sections. The body itself is preserved verbatim below. Returns
// (data, false) for every other page type. injectH1 (next in buildSection)
// still prepends the "# Name" + --- pair; steel-class.css hides that duplicate
// by the same h1+hr+card adjacency rule the other leaf cards use.
func buildClassLandingPage(data []byte, cfg *Config) ([]byte, bool) {
	fm, body := splitFrontmatter(string(data))
	if strings.TrimSpace(parseFrontmatterField(fm, "type")) != "class" {
		return data, false
	}
	name := stripMD(parseFrontmatterField(fm, "name"))
	book := bookLabelFromFM(cfg, fm)
	primaries := classPrimaries(fm)

	// The right rail balances the head the way the statblock's Level/role/EV
	// rail does: book chip up top, the primary characteristics as the rail
	// mini (the class's at-a-glance identity) with its caption as the deck
	// line directly beneath — ONE field (value + label), not two. The stat
	// strip below drops its starting-characteristics cell in exchange.
	slots := cardHeadSlots{
		NameTag:     "h2",
		LeftEyebrow: hLine("Class"),
		LeftPrimary: hLine(html.EscapeString(name)),
	}
	if book != "" {
		slots.RightEyebrow = hChip(html.EscapeString(book))
	}
	if len(primaries) > 0 {
		slots.RightPrimary = hMini(html.EscapeString(strings.Join(primaries, " · ")))
		slots.RightDeck = hLine("primary characteristics")
	}
	head := renderCardHead(slots)

	var card strings.Builder
	card.WriteString(`<section class="sc-classhead">`)
	card.WriteString(head)
	card.WriteString(classFlavor(fm))
	card.WriteString(classStatStrip(fm))
	card.WriteString(classPotencyStrip(fm))
	card.WriteString(classSkillsLine(fm))
	card.WriteString(`</section>`)

	if nav := classJumpNav(body); nav != "" {
		card.WriteString("\n" + nav)
	}
	body = dropDuplicateFlavor(body, fm)
	return []byte("---\n" + fm + "\n---\n\n" + card.String() + "\n\n" + body), true
}

// dropDuplicateFlavor removes the body's opening paragraph when it repeats the
// frontmatter flavor the card now displays — otherwise the same text renders
// twice within a screenful. Site-only; the data repos keep the full body.
func dropDuplicateFlavor(body, fm string) string {
	flavor := normalizeProse(stripMD(unquote(strings.TrimSpace(parseFrontmatterField(fm, "flavor")))))
	if flavor == "" {
		return body
	}
	rest := strings.TrimLeft(body, "\n")
	para := rest
	if i := strings.Index(rest, "\n\n"); i >= 0 {
		para = rest[:i]
	}
	if strings.HasPrefix(para, "#") || strings.HasPrefix(para, ">") || strings.HasPrefix(para, "|") {
		return body
	}
	if normalizeProse(stripMD(para)) != flavor {
		return body
	}
	return strings.TrimLeft(strings.TrimPrefix(rest, para), "\n")
}

// normalizeProse collapses whitespace so link-stripped body prose compares
// equal to the (already-plain) frontmatter copy.
func normalizeProse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// classCell renders one label/value cell of the card's stat rows.
func classCell(label, value string) string {
	return `<span class="sc-classhead__cell"><span class="l">` + html.EscapeString(label) +
		`</span><span class="v">` + html.EscapeString(value) + `</span></span>`
}

// classFlavor renders the frontmatter flavor paragraph inside the card, or ""
// when the class has none.
func classFlavor(fm string) string {
	f := stripMD(unquote(strings.TrimSpace(parseFrontmatterField(fm, "flavor"))))
	if f == "" {
		return ""
	}
	return `<p class="sc-classhead__flavor">` + html.EscapeString(f) + `</p>`
}

// classPrimaries returns the class's primary characteristics, link-stripped
// (they render in the head's right rail).
func classPrimaries(fm string) []string {
	var out []string
	for _, c := range parseFrontmatterList(fm, "primary_characteristics") {
		if v := stripMD(strings.TrimSpace(c)); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// classStatStrip renders the base-stat cells (starting stamina, stamina per
// level, recoveries); absent fields skip their cell (beastheart classes carry
// none of them). Primary characteristics live in the head's right rail.
func classStatStrip(fm string) string {
	var cells []string
	if v := strings.TrimSpace(parseFrontmatterField(fm, "starting_stamina")); v != "" {
		cells = append(cells, classCell("Starting stamina", v))
	}
	if v := strings.TrimSpace(parseFrontmatterField(fm, "stamina_per_level")); v != "" {
		cells = append(cells, classCell("Stamina per level", "+"+v))
	}
	if v := strings.TrimSpace(parseFrontmatterField(fm, "recoveries")); v != "" {
		cells = append(cells, classCell("Recoveries", v))
	}
	if len(cells) == 0 {
		return ""
	}
	return `<div class="sc-classhead__stats">` + strings.Join(cells, "") + `</div>`
}

// classPotencyStrip renders the Weak/Average/Strong potency cells, or "" when
// the class carries no potency frontmatter (beastheart).
func classPotencyStrip(fm string) string {
	type pot struct{ label, field string }
	pots := []pot{{"Weak", "weak_potency"}, {"Average", "average_potency"}, {"Strong", "strong_potency"}}
	var cells []string
	for _, p := range pots {
		v := stripMD(unquote(strings.TrimSpace(parseFrontmatterField(fm, p.field))))
		if v == "" {
			continue
		}
		cells = append(cells, classCell(p.label+" potency", v))
	}
	if len(cells) == 0 {
		return ""
	}
	return `<div class="sc-classhead__pot">` + strings.Join(cells, "") + `</div>`
}

// classSkillsLine renders the skills prose as the card's footer line.
func classSkillsLine(fm string) string {
	items := parseFrontmatterList(fm, "skills")
	if len(items) == 0 {
		if v := stripMD(unquote(strings.TrimSpace(parseFrontmatterField(fm, "skills")))); v != "" {
			items = []string{v}
		}
	}
	if len(items) == 0 {
		return ""
	}
	var parts []string
	for _, it := range items {
		parts = append(parts, stripMD(unquote(strings.TrimSpace(it))))
	}
	return `<p class="sc-classhead__skills"><span class="l">Skills</span> ` +
		html.EscapeString(strings.Join(parts, " ")) + `</p>`
}

// levelHeadingRe matches the ten "Nth-Level Features" section headings that
// dominate every class page's H2 list.
var levelHeadingRe = regexp.MustCompile(`^(\d+)(?:st|nd|rd|th)-Level Features$`)

// classJumpNav builds the anchor bar from the body's ## headings (H2 only —
// class pages have ~12: Basics, per-level features, subclass/kit sections).
// The ten "Nth-Level Features" headings collapse into one compact numbered
// "Level 1 2 … 10" group so the bar reads as a row of pills, not a wall.
func classJumpNav(body string) string {
	var links []string
	var lvls []string
	lvlPos := -1
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		txt := headingText(line)
		if txt == "" {
			continue
		}
		if m := levelHeadingRe.FindStringSubmatch(txt); m != nil {
			lvls = append(lvls, `<a href="#`+pySlugify(txt)+`" title="`+html.EscapeString(txt)+`">`+m[1]+`</a>`)
			if lvlPos < 0 {
				lvlPos = len(links)
				links = append(links, "") // placeholder, filled below
			}
			continue
		}
		links = append(links, `<a href="#`+pySlugify(txt)+`">`+html.EscapeString(txt)+`</a>`)
	}
	if lvlPos >= 0 {
		links[lvlPos] = `<span class="sc-classnav__lvls"><span class="l">Level</span>` +
			strings.Join(lvls, "") + `</span>`
	}
	if len(links) == 0 {
		return ""
	}
	return `<nav class="sc-classnav" aria-label="Class sections">` + strings.Join(links, "") + `</nav>`
}
