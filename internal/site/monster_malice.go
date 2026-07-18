package site

// Site-only post-pass: bands each Monsters-book statblock's own standalone
// Browse leaf page with its family's shared Malice featureblock (FOLLOWUPS #7
// piece 1), rendered as a native collapsible <details> — the same DOM/CSS the
// Villain Actions band already ships with (renderStatblockBand,
// statblock_card.go; `.sb__band--malice` has been sitting ready in
// steel-statblock.css since the High-Fantasy Steel statblock redesign landed).
// The family's `…-malice.md` featureblock keeps rendering as its own Browse
// page too — this only ADDS a rendering of the same content onto each sibling
// statblock, it doesn't move anything.
//
// WHY A POST-PASS, not baked into buildStatblockIslandPage/renderStatblockCard
// alongside the villain band: that shared render function's output is reused
// verbatim wherever embedItemCards later transcludes the statblock elsewhere —
// most importantly the Read tab's book-faithful chapter pages (site.yaml's
// embed_card_sections: [Browse, Read]), which ALSO separately embed the
// family's Malice featureblock as its own `{data-scc}` section (the
// sourcebook lists Malice once per family, after that family's statblocks).
// Baking the band into the shared renderer would duplicate the family's
// Malice text once per embedded statblock PLUS once standalone on every Read
// chapter page — confirmed against the built output (`Read/bestiary/monsters`
// carries an independent "Devil Malice" `{data-scc}` section alongside each
// Devil statblock). So this pass runs LAST (Build(), after embedItemCards —
// same reasoning as augmentClassOwnedBackLinks) and only ever writes to a
// statblock's own canonical Browse leaf file, never a transcluded copy,
// so the band appears exactly once: on the statblock's own page.
//
// Association is by SCC type-path, not directory: group dirs get reshuffled
// (echelon subdirs, the hoisted `statblock/` segment — see hoistStatblockPath
// in build.go), so a raw path comparison would miss cases like the per-echelon
// Demon/Undead/War Dog Malice tiers. A statblock's type-path always ends in
// ".statblock" (docs/statblocks.md → "SCC hierarchy"); stripping that suffix
// gives the exact type-path its family's Malice featureblock is coded under:
// `monster.devil.statblock/devil-high-judge` → `monster.devil` (the shared
// Devil Malice); `monster.demon.2nd-echelon.statblock/<id>` →
// `monster.demon.2nd-echelon` (that echelon's own "Level 4+" tier, not a merge
// of all four — each echelon dir carries its own Malice file at the same
// type-path). One family (Dragons) codes multiple species' Malice files under
// the SAME type-path (`monster.dragon`, one per species — the SCC scheme
// doesn't give species its own subcategory segment there); those are
// disambiguated by a shared filename slug (`crucible-dragon.md` statblock ↔
// `crucible-dragon-malice.md` featureblock) — see maliceCandidate.slug.
// Retainers/companions/fixtures/summoner statblocks have no `kind: malice`
// sibling at all (verified: 0 Malice featureblocks exist in the Summoner
// book) and simply get no band.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maliceKey derives the "source/typepath-without-.statblock" cache key shared
// by buildMaliceBandCache (keyed from a Malice featureblock's own scc, whose
// type-path already has no ".statblock" suffix) and augmentMonsterMaliceBands
// (keyed from each statblock's scc, suffix stripped). ok=false when scc is
// empty or malformed (e.g. the `@classify:false` inline example, which has no
// scc at all and is correctly excluded from banding).
func maliceKey(scc string) (key string, ok bool) {
	source, rest, found := strings.Cut(strings.TrimSpace(scc), "/")
	if !found || source == "" {
		return "", false
	}
	typePath, _, _ := strings.Cut(rest, "/")
	typePath = strings.TrimSuffix(typePath, ".statblock")
	if typePath == "" {
		return "", false
	}
	return source + "/" + typePath, true
}

// maliceCandidate is one family's rendered Malice band plus the disambiguating
// slug for the (rare) case where several Malice files share a type-path.
type maliceCandidate struct {
	slug string // malice file's basename minus a trailing "-malice"; "" when the key has only one candidate (slug is never consulted)
	html string
}

// buildMaliceBandCache pre-scans the RAW (pre-transform) source entries for
// `type: featureblock` pages with `kind: malice` and renders each into its
// finished collapsible-band HTML, keyed by maliceKey(scc). Runs BEFORE
// buildSection: a family's Malice file and its statblock siblings can land on
// either side of each other in entries' walk order (the Malice file sits in
// the group dir while pre-hoist statblocks sit one level deeper, in a
// `statblock/` subdir — filepath.Walk gives no cross-directory ordering
// guarantee), and once a Malice entry's own body is replaced by
// buildFeatureblockPage's .fb-wrap HTML (later in the SAME buildSection pass)
// its raw blockquote features are gone — mirrors statblockFeatureCache's
// reason for existing (statblock_preview.go).
//
// A Malice featureblock never matches a Browse group/hoist/flatten rule
// (verified against v2/site.yaml + build.go: the Browse section's only
// `groups:` rule is "kit"; hoistStatblockPath only rewrites paths containing
// "/statblock/"; flattenAdvancementFeaturesPath only rewrites
// "advancement-features" paths — a Malice file's relPath hits none of these),
// so its Browse destRel is always its bare source relPath; rewriteSectionLinks
// is run against that identity destRel so the cached HTML's links resolve
// correctly wherever a same-directory statblock sibling splices it in.
func buildMaliceBandCache(cfg *Config, entries []sourceEntry) map[string][]maliceCandidate {
	bands := map[string][]maliceCandidate{}
	for _, e := range entries {
		data, err := os.ReadFile(e.absPath)
		if err != nil {
			continue
		}
		fm, _ := splitFrontmatter(string(data))
		if strings.TrimSpace(parseFrontmatterField(fm, "type")) != "featureblock" ||
			strings.TrimSpace(parseFrontmatterField(fm, "kind")) != "malice" {
			continue
		}
		scc := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
		key, ok := maliceKey(scc)
		if !ok {
			continue
		}

		srcBookFolder := ""
		if book, ok := cfg.BookByKey(bookKeyFromSCC(scc)); ok {
			srcBookFolder = book.Folder
		}
		rewritten := rewriteSectionLinks(string(data), e.relPath, e.relPath, "Browse", srcBookFolder, cfg.Sections, cfg.SourceDirList())
		rfm, body := splitFrontmatter(rewritten)
		name := strings.TrimSpace(parseFrontmatterField(rfm, "name"))
		flavor := strings.TrimSpace(parseFrontmatterField(rfm, "flavor"))

		feats := parseStatblockIslandFeatures(body)
		for i := range feats {
			// Malice features share the villain band's blockquote shape (icon +
			// bold title + optional trailing-paren cost), so the same parser
			// applies unchanged; force the action/kind malice classifies them as
			// "passive" (no keyword/usage table) into the design's distinct malice
			// accent color (steel-statblock.css [data-action="malice"] = grey,
			// visually distinct from a plain passive trait).
			feats[i].Action, feats[i].Kind = "malice", "malice"
		}
		var featHTML strings.Builder
		for _, f := range feats {
			featHTML.WriteString(renderStatblockFeature(f))
		}
		intro := ""
		if flavor != "" {
			// Matches the design handoff's statblock-render.js band intro: the
			// family flavor line plus a small provenance badge naming the source
			// featureblock ("Devil Malice") — the band's own <summary> title stays
			// the generic "Malice Features" used across every family.
			intro = `<p class="sb__band-intro">` + richSb(flavor) +
				` <span class="sb__band-source">` + sbEsc(name) + `</span></p>`
		}
		html := renderStatblockBand("malice", "Malice Features", sbACT["malice"].glyph, intro, featHTML.String())

		slug := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(e.relPath), ".md"), "-malice")
		bands[key] = append(bands[key], maliceCandidate{slug: slug, html: html})
	}
	return bands
}

// lookupMaliceBand resolves a statblock's family band. A key with exactly one
// candidate always matches (every family but Dragons); a key with several
// (Dragons: one Malice file per species, all coded under the same
// `monster.dragon` type-path) is disambiguated by the statblock's own file
// slug against each candidate's slug (e.g. "crucible-dragon" statblock ↔
// "crucible-dragon-malice" featureblock).
func lookupMaliceBand(bands map[string][]maliceCandidate, key, statblockSlug string) (string, bool) {
	cands := bands[key]
	switch len(cands) {
	case 0:
		return "", false
	case 1:
		return cands[0].html, true
	default:
		for _, c := range cands {
			if c.slug == statblockSlug {
				return c.html, true
			}
		}
		return "", false
	}
}

// sbCardClose is the exact, unique tail renderStatblockCard emits: the
// .sb__features div's close, then .sb (article), then .sb-wrap (div). It
// appears exactly once per statblock page, before any later-appended content
// (the sc-src export template, a summoner retainer's Advancement
// Features/Summons sections, …), so splicing the band in right before its
// FIRST occurrence lands it as the last child of .sb__features — the same
// slot renderStatblockCard would put it in (features, then Villain Actions,
// then Malice, matching the design handoff's DOM order).
const sbCardClose = `</div></article></div>`

// spliceMaliceBand inserts bandHTML as the last child of a rendered
// statblock's .sb__features div. ok=false when the page isn't a rendered
// .sb-wrap card (defensive; augmentMonsterMaliceBands already filters to
// `type: statblock` pages, which are always carded) or already carries a band
// (idempotent, mirroring augmentClassOwnedBackLinks's own guard).
func spliceMaliceBand(page, bandHTML string) (string, bool) {
	if strings.Contains(page, `sb__band--malice`) {
		return page, false
	}
	idx := strings.Index(page, sbCardClose)
	if idx < 0 {
		return page, false
	}
	return page[:idx] + bandHTML + page[idx:], true
}

// augmentMonsterMaliceBands walks a section dir's rendered Browse leaves,
// splicing each `type: statblock` page's family Malice band (from bands, see
// buildMaliceBandCache) into its own card. Returns the number of pages
// modified. No-op when bands is empty (non-Monsters-book builds).
func augmentMonsterMaliceBands(sectionDir string, bands map[string][]maliceCandidate) (int, []string) {
	if len(bands) == 0 {
		return 0, nil
	}
	count := 0
	var errs []string
	walkErr := filepath.Walk(sectionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		page := readFile(path)
		fm, _ := splitFrontmatter(page)
		if strings.TrimSpace(parseFrontmatterField(fm, "type")) != "statblock" {
			return nil
		}
		scc := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
		key, ok := maliceKey(scc)
		if !ok {
			return nil
		}
		slug := strings.TrimSuffix(filepath.Base(path), ".md")
		band, ok := lookupMaliceBand(bands, key, slug)
		if !ok {
			return nil
		}
		updated, spliced := spliceMaliceBand(page, band)
		if !spliced {
			return nil
		}
		if werr := os.WriteFile(path, []byte(updated), 0644); werr != nil {
			errs = append(errs, fmt.Sprintf("malice band %s: %v", path, werr))
			return nil
		}
		count++
		return nil
	})
	if walkErr != nil {
		errs = append(errs, fmt.Sprintf("walk %s: %v", sectionDir, walkErr))
	}
	return count, errs
}
