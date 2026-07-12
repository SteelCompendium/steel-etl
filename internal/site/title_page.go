package site

// Title leaf pages: surface the echelon on the page itself.
//
// The book conveys a title's echelon through its "Nth Echelon Titles" group
// header, which a flat Browse/title/<id> leaf page loses. This transform reads
// the `echelon` frontmatter the TitleParser emits and inserts a book-styled
// "**Echelon:** <Nth>" paragraph above the Prerequisite line. Site-only — the
// shared data repos stay book-faithful.

import "strings"

// buildTitleEchelonPage inserts the "**Echelon:** <Nth>" paragraph into a
// `type: title` page carrying `echelon` frontmatter — above the first
// Prerequisite line (plain or link-wrapped label), or at the top of the body
// when no anchor exists so the echelon is never dropped. ok=false → not a
// title page / no echelon; the page passes through unchanged.
func buildTitleEchelonPage(data []byte) ([]byte, bool) {
	fm, body := splitFrontmatter(string(data))
	if fm == "" || strings.TrimSpace(parseFrontmatterField(fm, "type")) != "title" {
		return data, false
	}
	ech := strings.TrimSpace(parseFrontmatterField(fm, "echelon"))
	if ech == "" {
		return data, false
	}
	line := "**Echelon:** " + echelonOrdinal(ech)

	lines := strings.Split(body, "\n")
	at := -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "**Prerequisite") || strings.HasPrefix(t, "**[Prerequisite") {
			at = i
			break
		}
	}
	if at >= 0 {
		lines = append(lines[:at], append([]string{line, ""}, lines[at:]...)...)
		body = strings.Join(lines, "\n")
	} else {
		body = line + "\n\n" + strings.TrimLeft(body, "\n")
	}
	return []byte("---\n" + fm + "\n---\n\n" + strings.TrimLeft(body, "\n")), true
}
