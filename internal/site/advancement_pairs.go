package site

// Paired index cards for the flattened beastheart-companion and summoner-fixture
// group dirs. After flattenAdvancementFeaturesPath (build.go) runs in buildSection,
// each group dir holds <id>.md + <id>-advancement-features.md as flat siblings.
// This builder pairs them into a 2-up .sc-cards grid (base card immediately
// followed by its advancement card) so a pair shares a row, and a matching
// base-first .nav.yml order (advancementPairNavOrder) keeps the left sidebar in
// the same base-then-advancement sequence. SITE-ONLY: it reads the generated
// md-linked pages' frontmatter; the shared data repos are untouched. Styled by
// docs/stylesheets/steel-redesign.css (.sc-cards--pairs).

import (
	"fmt"
	"html"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const advFeatSuffix = "-advancement-features"

// roleAdvSubdir is the one subdir a flattened pair dir may carry without losing
// its pair-grid index: the retainer role-advancement landing (Plan 6). It sits
// beside the 21 base+advancement pairs under monster/retainer/ and is surfaced
// as a folder card rather than disqualifying the pairing.
const roleAdvSubdir = "role-advancement"

// hasRoleAdvSubdir reports whether subdirs contains the role-advancement landing.
func hasRoleAdvSubdir(subdirs []string) bool {
	for _, s := range subdirs {
		if s == roleAdvSubdir {
			return true
		}
	}
	return false
}

// advPair is one base entity and its advancement-features sibling. base is "" for
// an orphan advancement page (no base); adv is "" for a base with no advancement.
type advPair struct {
	base string
	adv  string
}

// advancementPairs groups a flattened group dir's leaves base-first: each base
// (natural-sorted) followed by its advancement file, then any orphan advancement
// files. ok=false when the dir has no advancement-features leaves or has subdirs
// (i.e. it isn't a pure flattened pair dir — fall through to the other builders).
func advancementPairs(files, subdirs []string) ([]advPair, bool) {
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
	// A `role-advancement` subdir (retainers) is an expected companion to the
	// pairs, not a disqualifier; any OTHER subdir means this isn't a pure
	// flattened pair dir, so fall through to the generic builders.
	extraSubdirs := 0
	for _, s := range subdirs {
		if s != roleAdvSubdir {
			extraSubdirs++
		}
	}
	if len(advByBase) == 0 || extraSubdirs > 0 {
		return nil, false
	}
	sort.Slice(bases, func(i, j int) bool { return naturalLess(bases[i], bases[j]) })

	var pairs []advPair
	seen := map[string]bool{}
	for _, bf := range bases {
		id := strings.TrimSuffix(bf, ".md")
		seen[id] = true
		pairs = append(pairs, advPair{base: bf, adv: advByBase[id]}) // adv may be ""
	}
	// Defensive: an advancement page with no base sibling stands on its own.
	var orphans []string
	for base, af := range advByBase {
		if !seen[base] {
			orphans = append(orphans, af)
		}
	}
	sort.Slice(orphans, func(i, j int) bool { return naturalLess(orphans[i], orphans[j]) })
	for _, af := range orphans {
		pairs = append(pairs, advPair{adv: af})
	}
	return pairs, true
}

// companionPreviewCard renders a companion base leaf as a .sb-prev preview from the
// cached island (companion_statblock.go). ok=false when the file isn't a cached
// companion (e.g. fixtures) — caller falls back to the generic card.
func companionPreviewCard(dir, baseFile string) (string, bool) {
	fm, _ := splitFrontmatter(readFile(filepath.Join(dir, baseFile)))
	scc := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
	island, hit := companionStatblockCache[scc]
	if !hit {
		return "", false
	}
	return renderStatblockPreviewCard(island, baseFile, ""), true
}

// advancementCardInner builds the index-card inner HTML for an advancement-features
// leaf: a compact one-row-per-feature list of the level each feature is gained at
// plus its name (e.g. "L3 Cat and Mouse"). Data comes from the leaf's frontmatter
// features[] (the same fbDoc shape featureblock_page.go renders on the full page),
// which survives the leaf's HTML transform — so no cache is needed. Names are
// link-stripped (linkText) then escaped. Returns "" when the leaf has no features,
// so the caller falls back to the bare "Advancement Features" card.
func advancementCardInner(dir, advFile string) string {
	fm, _ := splitFrontmatter(readFile(filepath.Join(dir, advFile)))
	var doc fbDoc
	if err := yaml.Unmarshal([]byte(fm), &doc); err != nil || len(doc.Features) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<ul class="sc-card__advlist">`)
	for _, f := range doc.Features {
		b.WriteString(`<li class="sc-card__advfeat">`)
		if f.Level > 0 {
			fmt.Fprintf(&b, `<span class="sc-card__advlvl">L%d</span>`, f.Level)
		}
		b.WriteString(`<span class="sc-card__advname">` +
			html.EscapeString(linkText(f.Name)) + `</span></li>`)
	}
	b.WriteString("</ul>\n")
	return b.String()
}

// buildAdvancementPairContent renders a group dir whose leaves come in
// <id>.md + <id>-advancement-features.md pairs as a 2-column pair grid.
// ok=false → caller falls through to the default index builders.
func buildAdvancementPairContent(dir, dirName string, files, subdirs []string) (string, bool) {
	pairs, ok := advancementPairs(files, subdirs)
	if !ok {
		return "", false
	}

	baseEyebrow, icon := "Companion", "paw"
	if pathHasSegment(dir, "fixture") {
		baseEyebrow, icon = "Fixture", "skull"
	} else if pathHasSegment(dir, "retainer") {
		baseEyebrow, icon = "Retainer", "sword-cross"
	}

	cardName := func(file string) string {
		if n := readFrontmatterName(filepath.Join(dir, file)); n != "" {
			return n
		}
		return fileToTitle(file)
	}

	// Detect cached companion previews; when present the grid doubles as a
	// .sb-prev preview grid (so statblock-preview.js + the zone CSS apply) and
	// carries the build-time zone defaults.
	previews := map[string]string{}
	for _, p := range pairs {
		if p.base == "" {
			continue
		}
		if cardHTML, ok := companionPreviewCard(dir, p.base); ok {
			previews[p.base] = cardHTML
		}
	}

	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n")
	if len(previews) > 0 {
		sb.WriteString(`<div class="sc-cards sc-cards--pairs sb-cards"` + sbPreviewDefaultAttrs() + ">\n")
	} else {
		sb.WriteString("<div class=\"sc-cards sc-cards--pairs\">\n")
	}
	for _, p := range pairs {
		if p.base != "" {
			if cardHTML, ok := previews[p.base]; ok {
				sb.WriteString(cardHTML + "\n")
			} else {
				sb.WriteString(card(p.base, icon, baseEyebrow, cardName(p.base), ""))
			}
		}
		if p.adv != "" {
			// Advancement card shares its base's name; the eyebrow distinguishes it.
			// (For an orphan advancement, fall back to its own name.)
			name := cardName(p.adv)
			if p.base != "" {
				name = cardName(p.base)
			}
			sb.WriteString(card(p.adv, icon, "Advancement Features", name, advancementCardInner(dir, p.adv)))
		}
	}
	sb.WriteString("</div>\n")
	// Retainers carry a sibling role-advancement landing (the 9 role groups);
	// surface it as a folder card below the base+advancement pairs.
	if hasRoleAdvSubdir(subdirs) {
		sb.WriteString("\n<div class=\"sc-folders\">\n")
		sb.WriteString(folderCard(roleAdvSubdir+"/", "sword-cross", "Role Advancement Abilities", 0))
		sb.WriteString("</div>\n")
	}
	return sb.String(), true
}

// advancementPairNavOrder returns the base-first file order for a flattened pair
// dir's .nav.yml (index.md, then base, advancement, base, advancement, …), so the
// left sidebar matches the index page's pairing instead of filename-sorting the
// advancement page ahead of its base. ok=false → caller writes a plain title-only
// .nav.yml.
func advancementPairNavOrder(files, subdirs []string) ([]string, bool) {
	pairs, ok := advancementPairs(files, subdirs)
	if !ok {
		return nil, false
	}
	order := []string{"index.md"}
	for _, p := range pairs {
		if p.base != "" {
			order = append(order, p.base)
		}
		if p.adv != "" {
			order = append(order, p.adv)
		}
	}
	// Keep the role-advancement landing in the sidebar (an explicit nav: list
	// excludes anything not named); it sorts last, after the pairs.
	if hasRoleAdvSubdir(subdirs) {
		order = append(order, roleAdvSubdir)
	}
	return order, true
}
