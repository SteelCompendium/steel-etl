package site

// Chapter opening header (.sc-cheyebrow / .sc-chtitle). buildChapterHead
// gives every `type: chapter` page a centered, book-style opening: a
// small-caps eyebrow ("<book> · Chapter <order>") above the title, which is
// tagged with a class via attr_list so it stays a real markdown heading (nav
// title, TOC entry, and ¶ permalink all survive). SITE-ONLY: the shared data
// repos are untouched. Runs AFTER injectH1 in buildSection — the h1 line it
// annotates is the one injectH1 guarantees. Styled by
// v2/docs/stylesheets/steel-indexes.css.

import (
	"html"
	"strings"
)

// bookLabelFromFM resolves a page's book display label: the printing_book
// frontmatter when present, else the config label for the page's scc book
// prefix. The printing stamp that injects printing_book runs AFTER
// buildSection, so during page transforms it is normally absent.
func bookLabelFromFM(cfg *Config, fm string) string {
	if b := unquote(strings.TrimSpace(parseFrontmatterField(fm, "printing_book"))); b != "" {
		return b
	}
	code := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
	pre, _, ok := strings.Cut(code, "/")
	if !ok || cfg == nil {
		return ""
	}
	b, _ := cfg.BookByKey(pre)
	return b.Label
}

// buildChapterHead prepends the eyebrow and tags the h1 on chapter pages;
// every other page type passes through unchanged.
func buildChapterHead(data []byte, cfg *Config) []byte {
	fm, body := splitFrontmatter(string(data))
	if strings.TrimSpace(parseFrontmatterField(fm, "type")) != "chapter" {
		return data
	}
	rest := strings.TrimLeft(body, "\n")
	if !strings.HasPrefix(rest, "# ") {
		return data
	}
	h1 := rest
	tail := ""
	if i := strings.Index(rest, "\n"); i >= 0 {
		h1, tail = rest[:i], rest[i:]
	}
	// tag the title — but never touch an h1 that already carries an attr list
	if !attrListRe.MatchString(h1) {
		h1 += " {.sc-chtitle}"
	}

	book := bookLabelFromFM(cfg, fm)
	order := strings.TrimSpace(parseFrontmatterField(fm, "order"))
	var parts []string
	if book != "" {
		parts = append(parts, book)
	}
	// order 0 is the unnumbered opener (Introduction); only real numbers read
	// as "Chapter N"
	if order != "" && order != "0" {
		parts = append(parts, "Chapter "+order)
	}
	eyebrow := ""
	if len(parts) > 0 {
		eyebrow = `<div class="sc-cheyebrow">` + html.EscapeString(strings.Join(parts, " · ")) + "</div>\n\n"
	}

	return []byte("---\n" + fm + "\n---\n\n" + eyebrow + h1 + tail)
}
