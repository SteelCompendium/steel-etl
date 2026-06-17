package site

// Site-only post-pass: replace embeddable items inlined in a container page's
// RenderSubtree body (ability/feature/trait/statblock/featureblock sections,
// each carrying a {data-scc="<code>"} heading marker) with that item's finished
// leaf card, transcluded by code. The card renderers (ability_cards.go,
// statblock_card.go, featureblock_page.go, trait_cards.go) are unchanged — this
// only relocates the HTML they already produced into the pages that contain the
// item. The data/ output repos are produced by the pipeline and never see this.
// Design: docs/superpowers/specs/2026-06-16-inline-item-cards-design.md.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// embedCardSections returns the configured section names, defaulting to Browse.
func embedCardSections(cfg *Config) []string {
	if len(cfg.EmbedCardSections) == 0 {
		return []string{"Browse"}
	}
	return cfg.EmbedCardSections
}

// cardableType is the set of frontmatter `type` values whose leaf page body is
// a finished card eligible for inline transclusion. Mirrors the leaf transforms
// in buildSection (ability_cards.go: ability/feature/trait; statblock_card.go:
// statblock; featureblock_page.go: featureblock/dynamic-terrain;
// companion_statblock.go: feature-group → the beastheart companion .sb-wrap).
var cardableType = map[string]bool{
	"ability":         true,
	"feature":         true,
	"trait":           true,
	"statblock":       true,
	"featureblock":    true,
	"dynamic-terrain": true,
	"feature-group":   true,
}

// standaloneType is the subset of card-able types whose card is NOT reproduced
// by a recursive feature/trait card (those are frontmatter-driven: statblock,
// featureblock, companion). A recursive feature card embeds its feature/ability
// descendants but never these, so a standalone item must get its own card and
// must never be swallowed by an ancestor feature — see spliceCards. Summoner
// minions (type statblock nested under a feature) are the motivating case.
var standaloneType = map[string]bool{
	"statblock":       true,
	"featureblock":    true,
	"dynamic-terrain": true,
	"feature-group":   true,
}

// cardEntry is a leaf's finished card HTML plus whether it is a standalone card
// (statblock/featureblock/feature-group) that an ancestor feature card cannot
// contain.
type cardEntry struct {
	html       string
	standalone bool
}

// leafCard extracts a card-able leaf page's scc code and its card entry (the
// file body with the injected "# Name\n\n---\n\n" head stripped). ok=false for
// pages whose type is not card-able or that lack an scc.
func leafCard(content string) (scc string, entry cardEntry, ok bool) {
	fm, body := splitFrontmatter(content)
	if fm == "" {
		return "", cardEntry{}, false
	}
	t := strings.TrimSpace(parseFrontmatterField(fm, "type"))
	if !cardableType[t] {
		return "", cardEntry{}, false
	}
	scc = strings.TrimSpace(parseFrontmatterField(fm, "scc"))
	if scc == "" {
		return "", cardEntry{}, false
	}
	html := strings.TrimSpace(stripLeadingHeading(strings.TrimLeft(body, "\n")))
	return scc, cardEntry{html: html, standalone: standaloneType[t]}, true
}

// dataSCCHeadingRe matches an ATX heading carrying a {data-scc="<code>"}
// attr_list marker (the per-item markers RenderSubtree stamps on descendants).
var dataSCCHeadingRe = regexp.MustCompile(`^(#{1,6})\s+.*\{data-scc="([^"]+)"\}\s*$`)

// atxHeadingRe matches any ATX heading line; len(submatch 1) is the level.
// Headings deeper than H6 were already demoted to bold by RenderSubtree
// (demoteOverflowHeadings), so 1-6 covers every heading reaching this pass.
var atxHeadingRe = regexp.MustCompile(`^(#{1,6})\s`)

// headingLevel returns a line's ATX heading level (1-6), or 0 if not a heading.
func headingLevel(line string) int {
	if m := atxHeadingRe.FindStringSubmatch(line); m != nil {
		return len(m[1])
	}
	return 0
}

// subtreeHasStandalone reports whether any {data-scc} heading in lines maps to a
// standalone card (statblock/featureblock/feature-group) in cards.
func subtreeHasStandalone(lines []string, cards map[string]cardEntry) bool {
	for _, line := range lines {
		if m := dataSCCHeadingRe.FindStringSubmatch(line); m != nil {
			if e, ok := cards[m[2]]; ok && e.standalone {
				return true
			}
		}
	}
	return false
}

// spliceCards rewrites a container page body: for every {data-scc} heading whose
// code is a card-able leaf in cards (and is not the page's own code), the
// heading is kept and its inlined sub-tree (down to the next heading of level <=
// its own) is replaced by the leaf card. Two cases are left intact and descended
// into instead: a heading whose code is absent/own, and a recursive container
// (feature/ability/trait) whose sub-tree contains a standalone card its leaf
// card cannot reproduce — descending lets that inner statblock/featureblock get
// its own card. Returns the new body and the number of cards spliced.
func spliceCards(body, ownSCC string, cards map[string]cardEntry) (string, int) {
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	spliced := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		m := dataSCCHeadingRe.FindStringSubmatch(line)
		if m == nil {
			out = append(out, line)
			continue
		}
		level, code := len(m[1]), m[2]
		entry, ok := cards[code]
		if !ok || code == ownSCC {
			out = append(out, line) // keep + descend; children may be card-able
			continue
		}
		// Sub-tree extent: from the next line up to (not incl.) the next heading
		// whose level <= this heading's level.
		j := i + 1
		for j < len(lines) {
			if lv := headingLevel(lines[j]); lv > 0 && lv <= level {
				break
			}
			j++
		}
		// A recursive container whose sub-tree holds a standalone item cannot be
		// monolithically carded (its leaf card omits that item) — descend so the
		// inner item gets its own card.
		if !entry.standalone && subtreeHasStandalone(lines[i+1:j], cards) {
			out = append(out, line)
			continue
		}
		// Keep the heading, drop the inlined sub-tree, insert the card.
		out = append(out, line, "", entry.html, "")
		spliced++
		i = j - 1
	}
	return strings.Join(out, "\n"), spliced
}

// embedItemCards is the Build() post-pass. Over the configured sections it
// builds a scc -> card-HTML map from every card-able leaf, then rewrites each
// container page in place, splicing leaf cards under their {data-scc} headings.
// Returns the number of container pages rewritten plus any errors.
func embedItemCards(cfg *Config) (int, []string) {
	var dirs []string
	for _, s := range embedCardSections(cfg) {
		dir := filepath.Join(cfg.DocsDir, s)
		if _, err := os.Stat(dir); err == nil {
			dirs = append(dirs, dir)
		}
	}

	var errs []string

	// Pass A: scc -> card entry, from every card-able leaf page.
	cards := map[string]cardEntry{}
	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			data, rErr := os.ReadFile(path)
			if rErr != nil {
				errs = append(errs, fmt.Sprintf("embed read %s: %v", path, rErr))
				return nil
			}
			if scc, entry, ok := leafCard(string(data)); ok {
				cards[scc] = entry
			}
			return nil
		})
	}

	// Pass B: splice into container pages (those still holding markdown
	// {data-scc} heading markers; leaf cards are HTML and never match).
	count := 0
	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			data, rErr := os.ReadFile(path)
			if rErr != nil {
				errs = append(errs, fmt.Sprintf("embed read %s: %v", path, rErr))
				return nil
			}
			fm, body := splitFrontmatter(string(data))
			if fm == "" || !strings.Contains(body, `{data-scc="`) {
				return nil
			}
			ownSCC := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
			newBody, n := spliceCards(body, ownSCC, cards)
			if n == 0 {
				return nil
			}
			out := "---\n" + fm + "\n---" + newBody
			if wErr := os.WriteFile(path, []byte(out), 0644); wErr != nil {
				errs = append(errs, fmt.Sprintf("embed write %s: %v", path, wErr))
				return nil
			}
			count++
			return nil
		})
	}

	return count, errs
}
