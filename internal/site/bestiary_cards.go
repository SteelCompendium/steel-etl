package site

// Card renderers for the Bestiary entity types (statblock / dynamic-terrain /
// retainer) and the monster-group landing assembler. These pages moved from the
// Bestiary tab into Browse (2026-06-10); see docs/superpowers/specs/
// 2026-06-10-bestiary-restructure-and-search-design.md. SITE-ONLY, like cards.go:
// all data is read from existing frontmatter — no data-repo changes. The crest is
// the bestiary `skull` glyph throughout (see iconPaths in cards.go).

import (
	"html"
	"os"
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
func buildMonsterGroupContent(dir, dirName string, files, subdirs []string) (string, bool) {
	if !isMonsterGroupDir(dir) {
		return "", false
	}
	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n")

	echelons, plain := splitEchelonSubdirs(subdirs)
	if len(echelons) > 0 {
		for _, ech := range echelons {
			sb.WriteString("## " + dirToTitle(ech) + "\n\n")
			ef, es := listDirChildren(filepath.Join(dir, ech))
			sb.WriteString(featureblockCards(dir, ech, ef))
			for _, sd := range es {
				if sd == "statblock" {
					sb.WriteString(statblockCardsFromDir(filepath.Join(dir, ech, sd), filepath.ToSlash(filepath.Join(ech, sd))))
				}
			}
		}
		return sb.String(), true
	}

	// Flat group: featureblock files at the group root, statblocks under statblock/.
	sb.WriteString(featureblockCards(dir, "", files))
	for _, sd := range plain {
		if sd == "statblock" {
			sb.WriteString(statblockCardsFromDir(filepath.Join(dir, sd), sd))
		}
	}
	return sb.String(), true
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
		fm, _ := splitFrontmatter(readBestiaryFile(filepath.Join(dir, relPrefix, f)))
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

// statblockCardsFromDir renders every statblock .md under sbDir as a card, with
// hrefs prefixed by relPrefix (the path from the group landing to sbDir).
func statblockCardsFromDir(sbDir, relPrefix string) string {
	entries, err := os.ReadDir(sbDir)
	if err != nil {
		return ""
	}
	var files []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasSuffix(n, ".md") && n != "index.md" {
			files = append(files, n)
		}
	}
	if len(files) == 0 {
		return ""
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })
	var sb strings.Builder
	sb.WriteString("<div class=\"sc-cards\">\n")
	for _, f := range files {
		fm, body := splitFrontmatter(readBestiaryFile(filepath.Join(sbDir, f)))
		name := parseFrontmatterField(fm, "name")
		if name == "" {
			name = fileToTitle(f)
		}
		sb.WriteString(statblockCard(fm, body, filepath.ToSlash(filepath.Join(relPrefix, f)), name))
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// readBestiaryFile is a thin os.ReadFile wrapper returning "" on error.
func readBestiaryFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
