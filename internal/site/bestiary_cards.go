package site

// Card renderers for the Bestiary entity types (statblock / dynamic-terrain /
// retainer) and the monster-group landing assembler. These pages moved from the
// Bestiary tab into Browse (2026-06-10); see docs/superpowers/specs/
// 2026-06-10-bestiary-restructure-and-search-design.md. SITE-ONLY, like cards.go:
// all data is read from existing frontmatter — no data-repo changes. The crest is
// the bestiary `skull` glyph throughout (see iconPaths in cards.go).

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// bestiarySource returns a provenance label derived from a page's SCC book
// prefix, so summoner-book creatures (portfolio minions/fixtures/champions, the
// rival summoner, summoner retainers) are marked as summoner-class content vs.
// Monsters-book creatures. "" → no marker (Monsters book is the unmarked default).
func bestiarySource(fm string) string {
	if strings.HasPrefix(parseFrontmatterField(fm, "scc"), "mcdm.summoner.") {
		return "Summoner"
	}
	return ""
}

// withSource prefixes a card's type label with its provenance ("Summoner · …")
// when the page is summoner-book content.
func withSource(fm, label string) string {
	if src := bestiarySource(fm); src != "" {
		return src + " · " + label
	}
	return label
}

// terrainStat extracts a value from the loose stats[] list in dynamic-terrain
// frontmatter by its pair name (e.g. "EV", "Size"). Terrain carries these as a
// YAML list of {name, value} objects rather than scalar keys; returns "" if the
// named stat is absent.
func terrainStat(fm, statName string) string {
	curName := ""
	inStats := false
	for _, line := range strings.Split(fm, "\n") {
		if !inStats {
			if strings.TrimSpace(line) == "stats:" {
				inStats = true
			}
			continue
		}
		// A non-indented, non-list line ends the stats block (next top-level key).
		if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "-") {
			break
		}
		item := strings.TrimPrefix(strings.TrimSpace(line), "- ")
		switch {
		case strings.HasPrefix(item, "name:"):
			curName = strings.Trim(strings.TrimSpace(strings.TrimPrefix(item, "name:")), "\"'")
		case strings.HasPrefix(item, "value:"):
			if curName == statName {
				return strings.Trim(strings.TrimSpace(strings.TrimPrefix(item, "value:")), "\"'")
			}
		}
	}
	return ""
}

// statField reads a stat that statblocks store as a scalar key but dynamic
// terrain stores in stats[]: scalar first, then the loose list.
func statField(fm, scalarKey, statName string) string {
	if v := parseFrontmatterField(fm, scalarKey); v != "" {
		return v
	}
	return terrainStat(fm, statName)
}

// terrainCard renders a .sc-card preview for a dynamic-terrain leaf page.
// Dynamic terrain has no role or keywords; it shows level, EV, and size stats.
func terrainCard(fm, body, file, name string) string {
	inner := statsBlock([][3]string{
		{orDash(parseFrontmatterField(fm, "level")), "Level", ""},
		{orDash(statField(fm, "ev", "EV")), "EV", ""},
		{orDash(statField(fm, "size", "Size")), "Size", ""},
	})
	if f := cardFlavor(fm, body); f != "" {
		inner += flavorDiv(f, 160)
	}
	return card(file, "skull", withSource(fm, "Dynamic Terrain"), name, inner)
}

// statblockPreviewCard builds the sbIsland for a statblock leaf and renders the
// compact .sb-prev preview card (statblock_preview.go), linking to the leaf's
// full page. The Summoner-book provenance chip is added via bestiarySource.
// Features are recovered from statblockFeatureCache when available: the
// group-landing assembler reads leaf pages after buildSection has already
// transformed their bodies to .sb-wrap HTML, so body-parsed features are
// empty by that point; the cache (populated at transform time when the source
// blockquote body was still present) restores them.
func statblockPreviewCard(fm, body, href, name string) string {
	d := buildStatblockIsland(fm, body)
	if d.Name == "" {
		d.Name = name
	}
	if scc := strings.TrimSpace(parseFrontmatterField(fm, "scc")); scc != "" {
		if feats, ok := statblockFeatureCache[scc]; ok {
			d.Features = feats
		}
	}
	return renderStatblockPreviewCard(d, href, bestiarySource(fm))
}

// bestiaryGroupParents are the statblock type roots whose direct child dirs are
// group landings: monster groups (monster/<group>), the summoner book's portfolio
// trees (minion/fixture/champion per <portfolio>), and the summoner/echelon trees
// (rival/<summoner>, retainer/<summoner>). All reuse the same group-landing
// assembler — lore (if any) + featureblock cards + statblock preview cards.
var bestiaryGroupParents = map[string]bool{
	"monster": true, "minion": true, "fixture": true,
	"champion": true, "rival": true, "retainer": true,
}

// isBestiaryGroupDir reports whether dir is a direct child group of a statblock
// type root (e.g. monster/goblins, minion/demon, rival/summoner) — the mixed node
// that becomes a group landing.
func isBestiaryGroupDir(dir string) bool {
	if bestiaryGroupParents[filepath.Base(filepath.Dir(dir))] {
		return true
	}
	// Summoner minion/champion trees insert a "summoner" class segment between the
	// type root and the portfolio group dir: monster/minion/summoner/<portfolio>.
	// The portfolio is the group dir; its grandparent is the type root. (Companions
	// put the class itself at the group-dir level, so they need no special case.)
	if filepath.Base(filepath.Dir(dir)) == "summoner" &&
		bestiaryGroupParents[filepath.Base(filepath.Dir(filepath.Dir(dir)))] {
		return true
	}
	return false
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
	// Fires for a group dir (monster/<group>, minion/<portfolio>, …) and also for
	// a bestiary type ROOT that directly holds statblock leaves AND a group subdir
	// — the mixed `retainer/` node (monster retainers + the summoner/ group).
	if !isBestiaryGroupDir(dir) && !isBestiaryTypeRootWithStatblocks(dir, files) {
		return "", false
	}
	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n")

	echelons, other := splitEchelonSubdirs(subdirs)
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

	// Flat group: featureblocks + statblock leaves are sibling files in the dir;
	// any non-echelon subdirs (e.g. retainer/summoner) render as folder cards.
	statblocks, features := splitByType(dir, "", files)
	sb.WriteString(featureblockCards(dir, "", features))
	sb.WriteString(statblockCards(dir, "", statblocks))
	sb.WriteString(groupSubdirCards(dir, other))
	return sb.String(), true
}

// isBestiaryTypeRootWithStatblocks reports whether dir is a bestiary type root
// (retainer/, minion/, …) that directly contains statblock leaf files — i.e. a
// mixed node carrying both leaves and group subdirs (the `retainer/` case, where
// monster retainers sit at the root and the summoner retainers form a subgroup).
func isBestiaryTypeRootWithStatblocks(dir string, files []string) bool {
	if !bestiaryGroupParents[filepath.Base(dir)] {
		return false
	}
	for _, f := range files {
		fm, _ := splitFrontmatter(readFile(filepath.Join(dir, f)))
		if strings.TrimSpace(parseFrontmatterField(fm, "type")) == "statblock" {
			return true
		}
	}
	return false
}

// groupSubdirCards renders folder cards for a group landing's child group dirs
// (e.g. the summoner/ subgroup under retainer/). "" when there are none.
func groupSubdirCards(dir string, subdirs []string) string {
	if len(subdirs) == 0 {
		return ""
	}
	sort.Slice(subdirs, func(i, j int) bool { return naturalLess(subdirs[i], subdirs[j]) })
	var sb strings.Builder
	sb.WriteString("<div class=\"sc-folders\">\n")
	for _, d := range subdirs {
		sb.WriteString(folderCard(d+"/", folderCrestIcon(dir, d), dirToTitle(d), countLeafFiles(filepath.Join(dir, d))))
	}
	sb.WriteString("</div>\n")
	return sb.String()
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
	sb.WriteString(sbCardsOpen())
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
		sb.WriteString(bestiaryLeafCard(dir, fm, body, href, name))
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// bestiaryLeafCard picks the preview card for a statblock leaf by its type root:
// dynamic terrain keeps the generic .sc-card (different model); every creature
// statblock (monster/minion/fixture/champion/rival AND retainers) renders as a
// rich .sb-prev mini-statblock.
func bestiaryLeafCard(dir, fm, body, href, name string) string {
	if pathHasSegment(dir, "dynamic-terrain") {
		return terrainCard(fm, body, href, name)
	}
	return statblockPreviewCard(fm, body, href, name)
}
