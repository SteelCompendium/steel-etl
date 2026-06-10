package site

// Card renderers for the Bestiary entity types (statblock / dynamic-terrain /
// retainer) and the monster-group landing assembler. These pages moved from the
// Bestiary tab into Browse (2026-06-10); see docs/superpowers/specs/
// 2026-06-10-bestiary-restructure-and-search-design.md. SITE-ONLY, like cards.go:
// all data is read from existing frontmatter — no data-repo changes. The crest is
// the bestiary `skull` glyph throughout (see iconPaths in cards.go).

import (
	"html"
	"strings"
)

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

// terrainCard renders a .sc-card preview for a dynamic-terrain leaf page.
// Dynamic terrain has no role or keywords; it shows level, EV, and size stats.
func terrainCard(fm, body, file, name string) string {
	inner := statsBlock([][3]string{
		{orDash(parseFrontmatterField(fm, "level")), "Level", ""},
		{orDash(parseFrontmatterField(fm, "ev")), "EV", ""},
		{orDash(parseFrontmatterField(fm, "size")), "Size", ""},
	})
	if f := cardFlavor(fm, body); f != "" {
		inner += flavorDiv(f, 160)
	}
	return card(file, "skull", "Dynamic Terrain", name, inner)
}

// retainerCard renders a .sc-card preview for a retainer statblock leaf page.
// The type label is "Retainer <Role>" (e.g. "Retainer Harrier"); immunities are
// rendered as a line block when present; EV may be '-'.
func retainerCard(fm, body, file, name string) string {
	label := strings.TrimSpace("Retainer " + strings.TrimSpace(parseFrontmatterField(fm, "role")))
	inner := ""
	if kw := parseFrontmatterList(fm, "keywords"); len(kw) > 0 {
		inner += tagsBlock(kw)
	}
	inner += statsBlock([][3]string{
		{orDash(parseFrontmatterField(fm, "level")), "Level", ""},
		{orDash(parseFrontmatterField(fm, "ev")), "EV", ""},
		{orDash(parseFrontmatterField(fm, "size")), "Size", ""},
	})
	if im := parseFrontmatterList(fm, "immunities"); len(im) > 0 {
		inner += lineBlock("Immunities", html.EscapeString(strings.Join(im, ", ")))
	}
	return card(file, "skull", label, name, inner)
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
