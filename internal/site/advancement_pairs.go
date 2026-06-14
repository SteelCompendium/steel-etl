package site

// Paired index cards for the flattened beastheart-companion and summoner-fixture
// group dirs. After flattenAdvancementFeaturesPath (build.go) runs in buildSection,
// each group dir holds <id>.md + <id>-advancement-features.md as flat siblings.
// This builder pairs them into a 2-up .sc-cards grid (base card immediately
// followed by its advancement card) so a pair shares a row. SITE-ONLY: it reads
// the generated md-linked pages' frontmatter; the shared data repos are untouched.
// Styled by docs/stylesheets/steel-redesign.css (.sc-cards--pairs).

import (
	"path/filepath"
	"sort"
	"strings"
)

const advFeatSuffix = "-advancement-features"

// buildAdvancementPairContent renders a group dir whose leaves come in
// <id>.md + <id>-advancement-features.md pairs as a 2-column pair grid.
// ok=false → caller falls through to the default index builders.
func buildAdvancementPairContent(dir, dirName string, files, subdirs []string) (string, bool) {
	advByBase := map[string]string{}
	var bases []string
	for _, f := range files {
		id := strings.TrimSuffix(f, ".md")
		if strings.HasSuffix(id, advFeatSuffix) {
			advByBase[strings.TrimSuffix(id, advFeatSuffix)] = f
		} else {
			bases = append(bases, f)
		}
	}
	if len(advByBase) == 0 {
		return "", false
	}
	sort.Slice(bases, func(i, j int) bool { return naturalLess(bases[i], bases[j]) })

	baseEyebrow, icon := "Companion", "paw"
	if pathHasSegment(dir, "fixture") {
		baseEyebrow, icon = "Fixture", "skull"
	}

	cardName := func(file string) string {
		if n := readFrontmatterName(filepath.Join(dir, file)); n != "" {
			return n
		}
		return fileToTitle(file)
	}

	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n")
	sb.WriteString("<div class=\"sc-cards sc-cards--pairs\">\n")

	seen := map[string]bool{}
	for _, bf := range bases {
		id := strings.TrimSuffix(bf, ".md")
		seen[id] = true
		name := cardName(bf)
		sb.WriteString(card(bf, icon, baseEyebrow, name, ""))
		if af, ok := advByBase[id]; ok {
			// Advancement card shares its base's name; the eyebrow distinguishes it.
			sb.WriteString(card(af, icon, "Advancement Features", name, ""))
		}
	}
	// Defensive: an advancement page with no base sibling renders on its own.
	var orphans []string
	for base, af := range advByBase {
		if !seen[base] {
			orphans = append(orphans, af)
		}
	}
	sort.Slice(orphans, func(i, j int) bool { return naturalLess(orphans[i], orphans[j]) })
	for _, af := range orphans {
		sb.WriteString(card(af, icon, "Advancement Features", cardName(af), ""))
	}

	sb.WriteString("</div>\n")
	return sb.String(), true
}
