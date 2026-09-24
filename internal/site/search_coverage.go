package site

// Book-text search gap-fill (SC-329). Sections listed in `search_uncovered_only`
// (v2: Read) are indexed by Material's search EXCEPT headings whose section a
// fully-indexed section (v2: Browse) already carries: those get
// data-search-exclude="" in their attr_list, so every piece of book text is
// findable exactly once — from its Browse page when it has one, from the book
// otherwise. Material opens a new index section at EVERY heading
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

var (
	// scAtxHeadingRe matches an ATX heading line ("### Title {attrs}").
	scAtxHeadingRe = regexp.MustCompile(`^(#{1,6})[ \t]+\S`)
	// scHeadingSCCRe extracts the code RenderSubtree stamps on a coded heading.
	scHeadingSCCRe = regexp.MustCompile(`data-scc="([^"]+)"`)
	// scTrailingAttrListRe matches a heading's trailing attr_list block.
	scTrailingAttrListRe = regexp.MustCompile(`\{[^{}]*\}$`)
	// scFenceRe matches a fenced-code delimiter line.
	scFenceRe = regexp.MustCompile("^[ \t]{0,3}(```|~~~)")
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
			case m[1] == fence:
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

// collectIndexedCodes adds every SCC code represented in sectionDir to codes:
// each page's `scc:` frontmatter, plus each heading's data-scc (a container
// page renders its coded children inline under their own headings — e.g.
// rule.combat/condition carries every condition/*). A missing dir adds nothing.
func collectIndexedCodes(sectionDir string, codes map[string]bool) []string {
	var errs []string
	filepath.Walk(sectionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			errs = append(errs, fmt.Sprintf("read %s: %v", path, readErr))
			return nil
		}
		content := string(data)
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
		return nil
	})
	return errs
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
// trailing attr_list, or adds a new one.
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
	if _, err := os.Stat(sectionDir); os.IsNotExist(err) {
		return 0, 0, nil
	}
	pages, headings := 0, 0
	var errs []string
	filepath.Walk(sectionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			errs = append(errs, fmt.Sprintf("read %s: %v", path, readErr))
			return nil
		}
		pages++
		out, n := markCoveredHeadings(string(data), covered)
		if n == 0 {
			return nil
		}
		headings += n
		if writeErr := os.WriteFile(path, []byte(out), 0644); writeErr != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", path, writeErr))
		}
		return nil
	})
	return pages, headings, errs
}
