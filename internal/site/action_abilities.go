package site

// Common-action ⇄ ability cross-references (SC-179).
//
// A few of the book's universal actions are documented in TWO places that the
// site keeps apart: the rules text lives on a `feature.common.*` page (Browse →
// Feature → Common → Main Actions → Free Strike), while the ability card players
// actually roll from lives under `feature.ability.common/…`. Where the book nests
// the ability inside its rule section (Grab, Knockback), RenderSubtree already
// inlines the card and the reader sees both. The free strikes are the exception:
// Draw Steel: Heroes prints the Melee/Ranged Weapon Free Strike ability blocks in
// Chapter 2 (character creation, step 7) and the rule in Chapter 10, under an
// uncoded `feature-group` that gets no Browse page of its own. The result is a
// Free Strike page that explains the rule and never shows — or links — the two
// abilities.
//
// This post-pass reconnects them from data already in the corpus: the two ability
// pages carry `subtype: free-strike` frontmatter, so any ability whose subtype is
// registered in abilitySubtypeRelations is listed as a preview card under its host
// rule page. Site-only (like class_backlinks.go / rival_summons.go) — the data/
// repos are produced by the pipeline and never see it.
//
// Ordering: this MUST run after embedItemCards, for the same reason
// augmentClassOwnedBackLinks does. The Free Strike leaf card is transcluded into
// container pages (the Read tab's Combat chapter, the Main Actions landing);
// appending the block before embedding would copy the ability grid into every one
// of them.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// actionAbilityRelation ties an ability `subtype` to the common-action rule page
// that documents it, plus the section chrome rendered on that page.
type actionAbilityRelation struct {
	host    string // SCC code of the host rule page
	heading string // "## <heading>" appended to the host page
	lead    string // one-line editorial lead under the heading
}

// abilitySubtypeRelations maps an ability's `subtype` frontmatter value to its
// host rule page. Keyed by subtype (not by page path) so the relation survives
// any Browse-path reshuffle; add a row here to surface another split rule/ability
// pair. Only `free-strike` is split today — every other common ability is nested
// under its rule section in the book and already renders inline.
var abilitySubtypeRelations = map[string]actionAbilityRelation{
	"free-strike": {
		host:    "mcdm.heroes.v1/feature.common.main-actions/free-strike",
		heading: "Free Strike Abilities",
		lead: "Every hero has these standard free strike abilities. Your class might grant " +
			"additional free strike options, and your kit can improve the standard ones.",
	},
}

// abilityPage is one candidate ability leaf: its docs-relative URL directory
// (path with ".md" stripped) plus the raw file contents.
type abilityPage struct {
	urlDir string
	data   string
}

// collectSubtypeAbilities walks sectionDir/feature/ability and returns, per
// registered subtype, the ability leaf pages carrying it. Returns nil when the
// tree is absent (e.g. a build without the Heroes book).
func collectSubtypeAbilities(sectionDir string) map[string][]abilityPage {
	root := filepath.Join(sectionDir, "feature", "ability")
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	found := map[string][]abilityPage{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == "index.md" || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data := readFile(path)
		fm, _ := splitFrontmatter(data)
		if fm == "" {
			return nil
		}
		subtype := strings.TrimSpace(parseFrontmatterField(fm, "subtype"))
		rel, ok := abilitySubtypeRelations[subtype]
		if !ok {
			return nil
		}
		// Scope to the relation's own book: a same-named subtype in a
		// different book (e.g. a future monster/summoner ability that
		// happens to reuse "free-strike") must not join a Heroes host page's
		// grid (MED-2, SC-179 round 3 review).
		scc := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
		if bookKeyFromSCC(scc) != bookKeyFromSCC(rel.host) {
			return nil
		}
		// Scope to the common-ability bucket itself: the book check alone
		// still let a same-book CLASS-specific ability that happens to reuse
		// the subtype (e.g. a future class's signature ability annotated
		// `@subtype: free-strike`) onto a grid whose lead reads "Every hero
		// has these standard free strike abilities" — false for a
		// class-gated option. sccTypePath is the same SCC-type-path helper
		// search_boost.go uses for its own common-action rule (MED-2
		// residual, SC-179 round 5 re-review).
		if sccTypePath(scc) != "feature.ability.common" {
			return nil
		}
		relPath, rErr := filepath.Rel(sectionDir, path)
		if rErr != nil {
			return nil
		}
		found[subtype] = append(found[subtype], abilityPage{
			urlDir: strings.TrimSuffix(filepath.ToSlash(relPath), ".md"),
			data:   data,
		})
		return nil
	})
	return found
}

// findPageBySCC returns the path of the leaf page under sectionDir/feature whose
// `scc` frontmatter equals code, or "" when no such page exists. Keeps walking
// past the first match rather than stopping early: if more than one page
// carries the code, that is a registry/data-integrity problem this pass has no
// business guessing through, so it reports the duplicate via dupErr and
// returns "" instead of silently picking whichever path the filesystem walk
// happened to visit first (LOW-3, SC-179 round 3 review).
func findPageBySCC(sectionDir, code string) (path string, dupErr string) {
	root := filepath.Join(sectionDir, "feature")
	if _, err := os.Stat(root); err != nil {
		return "", ""
	}
	var matches []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == "index.md" || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		fm, _ := splitFrontmatter(readFile(p))
		if fm != "" && strings.TrimSpace(parseFrontmatterField(fm, "scc")) == code {
			matches = append(matches, p)
		}
		return nil
	})
	switch len(matches) {
	case 0:
		return "", ""
	case 1:
		return matches[0], ""
	default:
		return "", fmt.Sprintf("scc %q matched %d pages, want 1: %s", code, len(matches), strings.Join(matches, ", "))
	}
}

// relHref re-expresses toURLDir (docs-relative, ".md" stripped) as a directory URL
// relative to fromURLDir — the same MkDocs directory-URL arithmetic the raw-HTML
// index cards use, where each path segment (the page's own leaf included) is one
// "../" hop.
func relHref(fromURLDir, toURLDir string) string {
	rel, err := filepath.Rel(fromURLDir, toURLDir)
	if err != nil {
		return toURLDir + "/"
	}
	return filepath.ToSlash(rel) + "/"
}

// actionAbilityCards renders the ability leaves as the same .sc-prevs preview
// grid the feature index pages use (feature_index.go), so a card looks identical
// whether the reader drilled to it or found it here.
func actionAbilityCards(hostURLDir string, pages []abilityPage) string {
	sort.Slice(pages, func(i, j int) bool { return naturalLess(pages[i].urlDir, pages[j].urlDir) })
	var sb strings.Builder
	sb.WriteString("<div class=\"sc-prevs\">\n")
	for _, p := range pages {
		fm, body := splitFrontmatter(p.data)
		it := extractPreviewItem(fm, body, "ability", klassFromDir(filepath.Dir(p.urlDir), "ability"))
		it.Href = relHref(hostURLDir, p.urlDir)
		sb.WriteString(renderPrevCard(it, false))
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// augmentActionAbilityPages appends, to each common-action rule page registered in
// abilitySubtypeRelations, a preview-card grid of the abilities that implement it.
// Derived entirely from the pages' own frontmatter (`subtype` on the ability, `scc`
// on the host); no data edits, no SCC re-mint. Runs after pages are written and is
// idempotent (guards on the heading).
//
// Iterates the REGISTRY (abilitySubtypeRelations), not just the subtypes this
// build happened to find, so a relation neither side of which is present in
// this section (a book it doesn't apply to at all — e.g. a Monsters/Beastheart/
// Summoner-only build has neither the Heroes host page nor any free-strike
// subtyped ability) stays completely silent, but every other combination
// reports into the returned errs instead of silently doing nothing: abilities
// found with no host page, a host page found with zero matching abilities (the
// "annotation got renamed/typo'd" scenario — the host is right there, so this
// book IS in the build), or the SCC registry producing more than one page for
// the host code (LOW-2/LOW-3, SC-179 rounds 3/5 review). Returns the number of
// pages modified.
func augmentActionAbilityPages(sectionDir string) (int, []string) {
	bySubtype := collectSubtypeAbilities(sectionDir)

	count := 0
	var errs []string
	subtypes := make([]string, 0, len(abilitySubtypeRelations))
	for st := range abilitySubtypeRelations {
		subtypes = append(subtypes, st)
	}
	sort.Strings(subtypes)

	for _, st := range subtypes {
		rel := abilitySubtypeRelations[st]
		pages, found := bySubtype[st]
		hostPath, dupErr := findPageBySCC(sectionDir, rel.host)
		if dupErr != "" {
			errs = append(errs, dupErr)
			continue
		}
		switch {
		case hostPath == "" && !found:
			// Neither side present: this relation's book simply isn't part
			// of this build/section. Not an error.
			continue
		case hostPath == "" && found:
			// Half-match: the book shipped the subtyped ability(ies) but not
			// their rule page.
			errs = append(errs, fmt.Sprintf(
				"action-abilities: %d %q-subtype page(s) found but host %s not in this build",
				len(pages), st, rel.host))
			continue
		case hostPath != "" && !found:
			// Half-match, the other direction: the rule page is right here
			// (so this book IS in the build) but nothing carries the
			// registered subtype — most likely a renamed or typo'd
			// `@subtype:` annotation in the book source silently dropping
			// the cross-reference from production.
			errs = append(errs, fmt.Sprintf(
				"action-abilities: host %s present but 0 %q-subtype page(s) found (renamed or typo'd @subtype annotation?)",
				rel.host, st))
			continue
		}
		// hostPath != "" && found: both sides present, proceed.
		page := readFile(hostPath)
		if strings.Contains(page, "\n## "+rel.heading+"\n") {
			continue // idempotent
		}
		hostRel, err := filepath.Rel(sectionDir, hostPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("rel %s: %v", hostPath, err))
			continue
		}
		hostURLDir := strings.TrimSuffix(filepath.ToSlash(hostRel), ".md")

		block := "\n\n## " + rel.heading + "\n\n"
		if rel.lead != "" {
			block += rel.lead + "\n\n"
		}
		block += actionAbilityCards(hostURLDir, pages)

		page = strings.TrimRight(page, "\n") + block
		if wErr := os.WriteFile(hostPath, []byte(page), 0644); wErr != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", hostPath, wErr))
			continue
		}
		count++
	}
	return count, errs
}
