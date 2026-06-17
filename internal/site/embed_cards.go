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
// statblock; featureblock_page.go: featureblock/dynamic-terrain).
var cardableType = map[string]bool{
	"ability":         true,
	"feature":         true,
	"trait":           true,
	"statblock":       true,
	"featureblock":    true,
	"dynamic-terrain": true,
}

// leafCard extracts a card-able leaf page's scc code and its card HTML (the file
// body with the injected "# Name\n\n---\n\n" head stripped). ok=false for pages
// whose type is not card-able or that lack an scc.
func leafCard(content string) (scc, card string, ok bool) {
	fm, body := splitFrontmatter(content)
	if fm == "" {
		return "", "", false
	}
	if !cardableType[strings.TrimSpace(parseFrontmatterField(fm, "type"))] {
		return "", "", false
	}
	scc = strings.TrimSpace(parseFrontmatterField(fm, "scc"))
	if scc == "" {
		return "", "", false
	}
	card = strings.TrimSpace(stripLeadingHeading(strings.TrimLeft(body, "\n")))
	return scc, card, true
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

// spliceCards rewrites a container page body: for every {data-scc} heading whose
// code is a card-able leaf in cards (and is not the page's own code), the
// heading is kept and its inlined sub-tree (down to the next heading of level <=
// its own) is replaced by the leaf card. Headings whose code is absent, or that
// carry no code, are left intact and descended into. Returns the new body and
// the number of cards spliced.
func spliceCards(body, ownSCC string, cards map[string]string) (string, int) {
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
		card, ok := cards[code]
		if !ok || code == ownSCC {
			out = append(out, line) // keep + descend; children may be card-able
			continue
		}
		// Card-able: keep the heading, drop its inlined sub-tree, insert the card.
		out = append(out, line, "", card, "")
		spliced++
		// Skip the swallowed sub-tree: every following line up to (not incl.) the
		// next heading whose level <= this heading's level.
		for i+1 < len(lines) {
			if lv := headingLevel(lines[i+1]); lv > 0 && lv <= level {
				break
			}
			i++
		}
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

	// Pass A: scc -> card HTML, from every card-able leaf page.
	cards := map[string]string{}
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
			if scc, card, ok := leafCard(string(data)); ok {
				cards[scc] = card
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
