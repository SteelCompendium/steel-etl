package site

// Card renderers for the Bestiary entity types (statblock / dynamic-terrain /
// retainer) and the monster-group landing assembler. These pages moved from the
// Bestiary tab into Browse (2026-06-10); see docs/superpowers/specs/
// 2026-06-10-bestiary-restructure-and-search-design.md. SITE-ONLY, like cards.go:
// all data is read from existing frontmatter — no data-repo changes. The crest is
// the bestiary `skull` glyph throughout (see iconPaths in cards.go).

import (
	"html"
	"path/filepath"
	"regexp"
	"sort"
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
	label := strings.TrimSpace("Retainer " + parseFrontmatterField(fm, "role"))
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

// isMonsterGroupDir reports whether dir is a direct child group of monster/
// (e.g. monster/goblins), the mixed node that becomes a group landing.
func isMonsterGroupDir(dir string) bool {
	return filepath.Base(filepath.Dir(dir)) == "monster"
}

// buildMonsterGroupContent renders a monster group landing's listing: the
// featureblock card(s) then the statblock preview cards, with echelon groups
// split under a "## <Echelon>" sub-header each. It emits the standard
// "# Title\n\n---\n\n" head so mergeGroupLanding can strip it and prepend the
// group lore. ok=false → caller falls through to the default index.
//
// Statblocks are hoisted out of a statblock/ folder (see hoistStatblockPath in
// build.go), so they now sit as direct files in the group/echelon dir alongside
// the Malice/Tactical-Stance featureblock(s); the two are split by frontmatter
// `type` rather than by directory.
func buildMonsterGroupContent(dir, dirName string, files, subdirs []string) (string, bool) {
	if !isMonsterGroupDir(dir) {
		return "", false
	}
	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n")

	echelons, _ := splitEchelonSubdirs(subdirs)
	if len(echelons) > 0 {
		for _, ech := range echelons {
			sb.WriteString("## " + dirToTitle(ech) + "\n\n")
			ef, _ := listDirChildren(filepath.Join(dir, ech))
			statblocks, features := splitByType(dir, ech, ef)
			sb.WriteString(featureblockCards(dir, ech, features))
			sb.WriteString(statblockCards(dir, ech, statblocks))
		}
		return sb.String(), true
	}

	// Flat group: featureblocks + statblocks are sibling files in the group dir.
	statblocks, features := splitByType(dir, "", files)
	sb.WriteString(featureblockCards(dir, "", features))
	sb.WriteString(statblockCards(dir, "", statblocks))
	return sb.String(), true
}

// splitByType partitions a dir's leaf files into statblock pages (frontmatter
// `type: statblock`) and everything else (featureblocks). relPrefix is the
// sub-path from dir to the files (an echelon dir, or "" for the group root).
func splitByType(dir, relPrefix string, files []string) (statblocks, features []string) {
	for _, f := range files {
		fm, _ := splitFrontmatter(readFile(filepath.Join(dir, relPrefix, f)))
		if strings.TrimSpace(parseFrontmatterField(fm, "type")) == "statblock" {
			statblocks = append(statblocks, f)
		} else {
			features = append(features, f)
		}
	}
	return statblocks, features
}

var echelonDirRe = regexp.MustCompile(`(?i)^\d(st|nd|rd|th)-echelon$`)

// splitEchelonSubdirs separates "Nst-echelon" subdirs (natural-sorted) from the
// rest (e.g. "statblock").
func splitEchelonSubdirs(subdirs []string) (echelons, other []string) {
	for _, d := range subdirs {
		if echelonDirRe.MatchString(d) {
			echelons = append(echelons, d)
		} else {
			other = append(other, d)
		}
	}
	sort.Slice(echelons, func(i, j int) bool { return naturalLess(echelons[i], echelons[j]) })
	return echelons, other
}

// featureblockCards renders the group's (or echelon's) featureblock .md files as
// cards. relPrefix is the href prefix from the group landing to the files' dir
// ("" for flat group root, "<echelon>" for echelon files).
func featureblockCards(dir, relPrefix string, files []string) string {
	if len(files) == 0 {
		return ""
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })
	var sb strings.Builder
	sb.WriteString("<div class=\"sc-cards\">\n")
	for _, f := range files {
		fm, _ := splitFrontmatter(readFile(filepath.Join(dir, relPrefix, f)))
		name := parseFrontmatterField(fm, "name")
		if name == "" {
			name = fileToTitle(f)
		}
		href := f
		if relPrefix != "" {
			href = filepath.ToSlash(filepath.Join(relPrefix, f))
		}
		sb.WriteString(card(href, "skull", featureblockLabel(name), name, ""))
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// featureblockLabel labels a featureblock card by its name: Malice / Tactical
// Stance get their own label; anything else is a generic "Feature".
func featureblockLabel(name string) string {
	switch {
	case strings.Contains(name, "Malice"):
		return "Malice"
	case strings.Contains(name, "Tactical Stance"):
		return "Tactical Stance"
	default:
		return "Feature"
	}
}

// statblockCards renders the given statblock .md files (siblings in dir/relPrefix)
// as a card grid, with hrefs relative to the group landing ("" prefix → bare
// filename; "<echelon>" prefix → echelon-relative).
func statblockCards(dir, relPrefix string, files []string) string {
	if len(files) == 0 {
		return ""
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })
	var sb strings.Builder
	sb.WriteString("<div class=\"sc-cards\">\n")
	for _, f := range files {
		fm, body := splitFrontmatter(readFile(filepath.Join(dir, relPrefix, f)))
		name := parseFrontmatterField(fm, "name")
		if name == "" {
			name = fileToTitle(f)
		}
		href := f
		if relPrefix != "" {
			href = filepath.ToSlash(filepath.Join(relPrefix, f))
		}
		sb.WriteString(statblockCard(fm, body, href, name))
	}
	sb.WriteString("</div>\n")
	return sb.String()
}
