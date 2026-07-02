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
func buildClassLandingPage(data []byte) ([]byte, bool) {
	fm, body := splitFrontmatter(string(data))
	if strings.TrimSpace(parseFrontmatterField(fm, "type")) != "class" {
		return data, false
	}
	name := stripMD(parseFrontmatterField(fm, "name"))
	book := unquote(strings.TrimSpace(parseFrontmatterField(fm, "printing_book")))

	head := renderCardHead(cardHeadSlots{
		NameTag:     "h2",
		LeftEyebrow: hLine("Class"),
		LeftPrimary: hLine(html.EscapeString(name)),
		LeftDeck:    hLine(html.EscapeString(book)),
	})

	var card strings.Builder
	card.WriteString(`<section class="sc-classhead">`)
	card.WriteString(head)
	card.WriteString(classPotencyStrip(fm))
	card.WriteString(`</section>`)

	if nav := classJumpNav(body); nav != "" {
		card.WriteString("\n" + nav)
	}
	return []byte("---\n" + fm + "\n---\n\n" + card.String() + "\n\n" + body), true
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
		cells = append(cells, `<span class="sc-classhead__potcell"><span class="l">`+p.label+
			` potency</span><span class="v">`+html.EscapeString(v)+`</span></span>`)
	}
	if len(cells) == 0 {
		return ""
	}
	return `<div class="sc-classhead__pot">` + strings.Join(cells, "") + `</div>`
}

// classJumpNav builds the anchor bar from the body's ## headings (H2 only —
// class pages have ~12: Basics, per-level features, subclass/kit sections).
func classJumpNav(body string) string {
	var links []string
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		txt := headingText(line)
		if txt == "" {
			continue
		}
		links = append(links, `<a href="#`+pySlugify(txt)+`">`+html.EscapeString(txt)+`</a>`)
	}
	if len(links) == 0 {
		return ""
	}
	return `<nav class="sc-classnav" aria-label="Class sections">` + strings.Join(links, "") + `</nav>`
}
