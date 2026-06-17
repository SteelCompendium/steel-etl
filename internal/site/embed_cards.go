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
	"path"
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
// contain, plus the leaf's URL directory (docs-relative path, ".md" stripped)
// that its relative links were computed against — needed to rebase those links
// when the card is transcluded into a container page at a different depth.
type cardEntry struct {
	html       string
	standalone bool
	dir        string
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

// hrefSrcRe matches a relative-or-absolute href/src attribute value.
var hrefSrcRe = regexp.MustCompile(`(href|src)="([^"]*)"`)

// embedMdLinkRe matches a Markdown link target: the "](target)" tail of [text](target).
var embedMdLinkRe = regexp.MustCompile(`\]\(([^)\s]+)\)`)

// rebaseURL re-expresses a relative link authored against fromDir so it resolves
// the same from toDir (both docs-relative directories). Absolute, protocol,
// anchor-only, and special-scheme values pass through unchanged.
func rebaseURL(val, fromDir, toDir string) string {
	if val == "" || strings.HasPrefix(val, "/") || strings.HasPrefix(val, "#") ||
		strings.Contains(val, "://") || strings.HasPrefix(val, "mailto:") ||
		strings.HasPrefix(val, "tel:") || strings.HasPrefix(val, "data:") {
		return val
	}
	pathPart, suffix := val, ""
	if k := strings.IndexAny(val, "#?"); k >= 0 {
		pathPart, suffix = val[:k], val[k:]
	}
	if pathPart == "" {
		return val
	}
	trailing := strings.HasSuffix(pathPart, "/")
	target := path.Clean(path.Join(fromDir, pathPart))
	rel, err := filepath.Rel(toDir, target)
	if err != nil {
		return val
	}
	rel = filepath.ToSlash(rel)
	if trailing && !strings.HasSuffix(rel, "/") {
		rel += "/"
	}
	return rel + suffix
}

// rebaseLinks rewrites every relative link in a transcluded card from the leaf's
// location (fromURLDir) to the container's (toURLDir), in both href/src
// attributes and Markdown "](target)" tails. A link is rebased against one of
// two bases by form: a `.md` target is resolved by MkDocs relative to the source
// FILE directory (path.Dir of the URL dir), while every other relative target is
// a final URL relative to the page's URL directory (which, with directory URLs,
// includes the page name as a segment). A no-op when the directories match.
func rebaseLinks(html, fromURLDir, toURLDir string) string {
	if fromURLDir == toURLDir {
		return html
	}
	fromFileDir, toFileDir := path.Dir(fromURLDir), path.Dir(toURLDir)
	rebase := func(val string) string {
		p := val
		if k := strings.IndexAny(val, "#?"); k >= 0 {
			p = val[:k]
		}
		if strings.HasSuffix(p, ".md") {
			return rebaseURL(val, fromFileDir, toFileDir)
		}
		return rebaseURL(val, fromURLDir, toURLDir)
	}
	html = hrefSrcRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := hrefSrcRe.FindStringSubmatch(m)
		if nv := rebase(sub[2]); nv != sub[2] {
			return sub[1] + `="` + nv + `"`
		}
		return m
	})
	html = embedMdLinkRe.ReplaceAllStringFunc(html, func(m string) string {
		sub := embedMdLinkRe.FindStringSubmatch(m)
		if nv := rebase(sub[1]); nv != sub[1] {
			return "](" + nv + ")"
		}
		return m
	})
	return html
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
// its own card. Each spliced card's relative links are rebased from the leaf's
// URL directory to containerDir. Returns the new body and the number of cards
// spliced.
func spliceCards(body, ownSCC, containerDir string, cards map[string]cardEntry) (string, int) {
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
		// Full sub-tree extent: from the next line up to (not incl.) the next
		// heading whose level <= this heading's level. Used for the descend test.
		full := i + 1
		for full < len(lines) {
			if lv := headingLevel(lines[full]); lv > 0 && lv <= level {
				break
			}
			full++
		}
		// A recursive container whose sub-tree holds a standalone item cannot be
		// monolithically carded (its leaf card omits that item) — descend so the
		// inner item gets its own card.
		if !entry.standalone && subtreeHasStandalone(lines[i+1:full], cards) {
			out = append(out, line)
			continue
		}
		// Swallow extent: like full, but also stop before a nested standalone item
		// (a statblock/featureblock the card can't contain — it gets its own card
		// next). A companion's advancement-features featureblock is nested under
		// its statblock heading, so without this the statblock card would eat it.
		sw := i + 1
		for sw < full {
			if hm := dataSCCHeadingRe.FindStringSubmatch(lines[sw]); hm != nil {
				if e, ok := cards[hm[2]]; ok && e.standalone {
					break
				}
			}
			sw++
		}
		// Keep the heading, drop the inlined sub-tree, insert the card (with its
		// relative links rebased to the container's depth).
		out = append(out, line, "", rebaseLinks(entry.html, entry.dir, containerDir), "")
		spliced++
		i = sw - 1
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
				if rel, rErr := filepath.Rel(cfg.DocsDir, path); rErr == nil {
					entry.dir = strings.TrimSuffix(filepath.ToSlash(rel), ".md")
				}
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
			containerDir := ""
			if rel, rErr := filepath.Rel(cfg.DocsDir, path); rErr == nil {
				containerDir = strings.TrimSuffix(filepath.ToSlash(rel), ".md")
			}
			newBody, n := spliceCards(body, ownSCC, containerDir, cards)
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
