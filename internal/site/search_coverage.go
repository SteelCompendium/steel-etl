package site

// Book-text search gap-fill (SC-329). Sections listed in `search_uncovered_only`
// (v2: Read) are indexed by Material's search EXCEPT headings whose section a
// fully-indexed section (v2: Browse) already carries: those get
// data-search-exclude="" in their attr_list, so covered book text is findable
// from its Browse page and uncovered book text from the book itself, without
// duplicates. Not quite every word: a container page's own intro prose (e.g. a
// Monsters group landing's per-echelon introduction) is itself excluded as
// covered while its Browse copy renders only the coded children, not the
// container's own prose — deferred to Backlog SC-345. Material opens a new
// index section at EVERY heading
// (material/plugins/search/plugin.py Parser), so every covered heading is
// marked, not only the root of a covered subtree. Heading-level exclusion is
// safe from the tag-name-keyed skip-set bug that forced markSearchExcluded's
// <address> wrapper: headings never nest, so the heading's own close tag
// clears it. Spec: workspace
// docs/superpowers/specs/2026-09-23-book-search-gap-fill-design.md.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// scBlockquotePrefixRe is the leading-blockquote-marker portion shared by
// scAtxHeadingRe and scFenceRe: perks/complications/treasures render their
// ability card as a blockquote (e.g. `> ###### Arcane Trick`), and
// Python-Markdown recognizes a heading (or a fence) inside one just like an
// unindented one. Known divergence, harmless: Python-Markdown also treats
// "#Heading" (no space before the text) as a heading; the pipeline never
// emits that form, so scAtxHeadingRe does not need to match it.
const scBlockquotePrefixRe = `(?:[ \t]{0,3}>[ \t]?)*`

var (
	// scAtxHeadingRe matches an ATX heading line ("### Title {attrs}"),
	// optionally nested inside one or more blockquote markers.
	scAtxHeadingRe = regexp.MustCompile(`^` + scBlockquotePrefixRe + `(#{1,6})[ \t]+\S`)
	// scHeadingSCCRe extracts the code RenderSubtree stamps on a coded heading.
	scHeadingSCCRe = regexp.MustCompile(`data-scc="([^"]+)"`)
	// scTrailingAttrListRe matches a heading's trailing attr_list block.
	scTrailingAttrListRe = regexp.MustCompile(`\{[^{}]*\}$`)
	// scFenceRe matches a fenced-code delimiter line, optionally inside a
	// blockquote, capturing the whole run of the fence character (3 or more)
	// so the caller can apply CommonMark's closing rule: same character, run
	// length >= the opener's.
	scFenceRe = regexp.MustCompile("^" + scBlockquotePrefixRe + "[ \t]{0,3}(`{3,}|~{3,})")
)

// searchExcludeAttr is the explicit key="" form: it does not rely on
// attr_list's bare-key parsing, and Material only tests the key's presence.
const searchExcludeAttr = `data-search-exclude=""`

// bodyStart returns the index of the first line after YAML frontmatter (0 when
// there is none, or it is unterminated).
func bodyStart(lines []string) int {
	if len(lines) == 0 || lines[0] != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			return i + 1
		}
	}
	return 0
}

// forEachHeading calls fn with the index and level of every ATX heading line
// outside YAML frontmatter and fenced code.
func forEachHeading(lines []string, fn func(i, level int)) {
	fence := ""
	for i := bodyStart(lines); i < len(lines); i++ {
		if m := scFenceRe.FindStringSubmatch(lines[i]); m != nil {
			switch {
			case fence == "":
				fence = m[1]
			// Close only on a run of the SAME character at least as long as
			// the opener's (CommonMark's rule) — a shorter or different-char
			// run (e.g. a ``` line inside a ```` fence) is fence content.
			case m[1][0] == fence[0] && len(m[1]) >= len(fence):
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if m := scAtxHeadingRe.FindStringSubmatch(lines[i]); m != nil {
			fn(i, len(m[1]))
		}
	}
}

// walkMarkdown walks dir, executing fn for each markdown file found. Returns
// any walk, stat, or read errors encountered. A missing root directory is a
// silent no-op (returns empty errs).
func walkMarkdown(dir string, fn func(path, content string)) []string {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	var errs []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errs = append(errs, fmt.Sprintf("walk %s: %v", path, err))
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			errs = append(errs, fmt.Sprintf("read %s: %v", path, readErr))
			return nil
		}
		fn(path, string(data))
		return nil
	})
	return errs
}

// collectIndexedCodes adds every SCC code represented in sectionDir to codes:
// each page's `scc:` frontmatter, plus each heading's data-scc (a container
// page renders its coded children inline under their own headings — e.g.
// rule.combat/condition carries every condition/*). A missing dir adds nothing.
func collectIndexedCodes(sectionDir string, codes map[string]bool) []string {
	return walkMarkdown(sectionDir, func(path, content string) {
		fm, _ := splitFrontmatter(content)
		if code := parseFrontmatterField(fm, "scc"); code != "" {
			codes[code] = true
		}
		lines := strings.Split(content, "\n")
		forEachHeading(lines, func(i, _ int) {
			if m := scHeadingSCCRe.FindStringSubmatch(lines[i]); m != nil {
				codes[m[1]] = true
			}
		})
	})
}

// markCoveredHeadings adds data-search-exclude="" to every covered heading: one
// whose data-scc code is in covered, or that sits under such a heading.
// Returns the new content and how many headings it marked. Idempotent: an
// already-marked heading is left alone and not counted.
func markCoveredHeadings(content string, covered map[string]bool) (string, int) {
	lines := strings.Split(content, "\n")
	type open struct {
		level   int
		covered bool
	}
	var stack []open
	marked := 0
	forEachHeading(lines, func(i, level int) {
		for len(stack) > 0 && stack[len(stack)-1].level >= level {
			stack = stack[:len(stack)-1]
		}
		// The top of the stack already folds in its own ancestors' coverage.
		isCovered := len(stack) > 0 && stack[len(stack)-1].covered
		if m := scHeadingSCCRe.FindStringSubmatch(lines[i]); m != nil && covered[m[1]] {
			isCovered = true
		}
		stack = append(stack, open{level, isCovered})
		if isCovered && !strings.Contains(lines[i], "data-search-exclude") {
			lines[i] = withSearchExclude(lines[i])
			marked++
		}
	})
	return strings.Join(lines, "\n"), marked
}

// withSearchExclude appends the exclusion attribute inside a heading's
// trailing attr_list, or adds a new one. Appending at line end works the same
// way inside a blockquote (`> ###### Title {…}`): Python-Markdown's attr_list
// extension applies a trailing {…} block to a blockquoted heading exactly as
// it does to an unindented one.
func withSearchExclude(line string) string {
	line = strings.TrimRight(line, " \t")
	if scTrailingAttrListRe.MatchString(line) {
		return line[:len(line)-1] + " " + searchExcludeAttr + "}"
	}
	return line + " {" + searchExcludeAttr + "}"
}

// applyUncoveredOnlySearch marks the covered headings on every page of a
// search_uncovered_only section. Returns pages visited and headings marked;
// pages with nothing to mark are not rewritten.
func applyUncoveredOnlySearch(docsDir, sectionName string, covered map[string]bool) (int, int, []string) {
	sectionDir := filepath.Join(docsDir, sectionName)
	pages, headings := 0, 0
	var errs []string
	walkErrs := walkMarkdown(sectionDir, func(path, content string) {
		pages++
		out, n := markCoveredHeadings(content, covered)
		if n == 0 {
			return
		}
		headings += n
		if writeErr := os.WriteFile(path, []byte(out), 0644); writeErr != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", path, writeErr))
		}
	})
	return pages, headings, append(walkErrs, errs...)
}
