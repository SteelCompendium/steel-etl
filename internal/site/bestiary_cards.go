package site

// Card renderers for the Bestiary entity types (statblock / dynamic-terrain /
// retainer) and the monster-group landing assembler. These pages moved from the
// Bestiary tab into Browse (2026-06-10); see docs/superpowers/specs/
// 2026-06-10-bestiary-restructure-and-search-design.md. SITE-ONLY, like cards.go:
// all data is read from existing frontmatter — no data-repo changes. The crest is
// the bestiary `skull` glyph throughout (see iconPaths in cards.go).

import "strings"

// statblockTypeLabel composes "<Organization> <Role>" (e.g. "Horde Harrier"),
// falling back to whichever is present, then "Statblock".
func statblockTypeLabel(fm string) string {
	org := strings.TrimSpace(parseFrontmatterField(fm, "organization"))
	role := strings.TrimSpace(parseFrontmatterField(fm, "role"))
	if label := strings.TrimSpace(org + " " + role); label != "" {
		return label
	}
	return "Statblock"
}

// statblockCard renders a .sc-card preview for a monster statblock leaf page.
func statblockCard(fm, body, file, name string) string {
	inner := ""
	if kw := parseFrontmatterList(fm, "keywords"); len(kw) > 0 {
		inner += tagsBlock(kw)
	}
	inner += statsBlock([][3]string{
		{orDash(parseFrontmatterField(fm, "level")), "Level", ""},
		{orDash(parseFrontmatterField(fm, "ev")), "EV", ""},
		{orDash(parseFrontmatterField(fm, "size")), "Size", ""},
		{orDash(parseFrontmatterField(fm, "speed")), "Speed", ""},
	})
	return card(file, "skull", statblockTypeLabel(fm), name, inner)
}
