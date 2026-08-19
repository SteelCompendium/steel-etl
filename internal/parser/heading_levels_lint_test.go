package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Structural lint (SC-173): a feature's heading level decides which container it
// belongs to, and a heading authored one level too shallow silently escapes its
// container.
//
// The Talent's "Repel" shipped as `#### Repel` among `##### ` siblings. One `#`
// short made it a sibling of the "1st-Level Tradition Features" container rather
// than a member of it, so on the class page it fell out of the container's card
// and rendered as raw markdown prose (the publisher's 📏/🎯 glyphs, bare table)
// while every sibling rendered as an ability card -- and it promoted into the
// page's right-hand section TOC as if it were a top-level class feature.
//
// Two rules, both false-positive-free across all four books:
//
//	R1 sibling-run uniformity -- a maximal run of CONSECUTIVE `@type: feature`
//	   headings that each carry an `@subclass` must be level-uniform.
//
//	R2 container membership -- when a container feature's body holds a member
//	   table (`| Tradition | Feature |` rows linking members by `scc:` code),
//	   every member named there must be a DESCENDANT of that container, not a
//	   sibling of it.
//
// R2 is deliberately convention-relative. A book that uniformly keeps members at
// the container's own level is following its own house style, not carrying a bug
// -- the Beastheart book does exactly that (it skips H4 entirely, running H3 then
// H5, and reserves H6 for ability headings). So a flat container is only reported
// when the SAME FILE also contains a container that does nest its members. That
// is what separates the Null's 5th-Level Tradition Feature (flat, while the Null's
// own 2nd- and 8th-level containers nest at H5 -- a real bug, fixed) from the
// Beastheart's uniformly flat containers (a pending book-wide decision, not a
// defect this test should fail on).

var (
	lintHeadingRe = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*$`)
	lintSCCLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(scc(?:\.v\d+)?:[^)]+\)`)
)

// lintHeading is one document heading plus the annotation fields (if any) that
// immediately precede it.
type lintHeading struct {
	idx    int
	line   int
	level  int
	name   string
	fields map[string]string
}

// lintDoc is a parsed book source: every heading in document order, with the
// annotated ones carrying their fields.
type lintDoc struct {
	lines []string
	heads []lintHeading
}

// parseLintDoc collects every heading in src. Blockquote lines are skipped:
// `> ###### Name` ability callouts are content inside a section, not document
// structure.
func parseLintDoc(src string) *lintDoc {
	d := &lintDoc{lines: strings.Split(src, "\n")}

	byEndLine := map[int]map[string]string{}
	for _, a := range ExtractAnnotations(src) {
		byEndLine[a.EndLine] = a.Fields
	}

	var pending map[string]string
	for i, line := range d.lines {
		lineNo := i + 1
		if strings.HasPrefix(line, ">") {
			continue
		}
		if f, ok := byEndLine[lineNo]; ok {
			pending = f
			continue
		}
		if m := lintHeadingRe.FindStringSubmatch(line); m != nil {
			d.heads = append(d.heads, lintHeading{
				idx:    len(d.heads),
				line:   lineNo,
				level:  len(m[1]),
				name:   m[2],
				fields: pending,
			})
			pending = nil
			continue
		}
		if strings.TrimSpace(line) != "" {
			pending = nil
		}
	}
	return d
}

// extent returns the half-open heading-index range of h's descendants.
func (d *lintDoc) extent(h lintHeading) (int, int) {
	end := h.idx + 1
	for end < len(d.heads) && d.heads[end].level > h.level {
		end++
	}
	return h.idx + 1, end
}

// parentEnd returns the heading index at which h's parent subtree ends, i.e. the
// limit of the sibling group h lives in.
func (d *lintDoc) parentEnd(h lintHeading) int {
	for j := h.idx - 1; j >= 0; j-- {
		if d.heads[j].level < h.level {
			_, end := d.extent(d.heads[j])
			return end
		}
	}
	return len(d.heads)
}

// subtreeLines returns every source line covered by h and its descendants -- a
// container's member table usually sits under its own `###### … Table` heading.
func (d *lintDoc) subtreeLines(h lintHeading) []string {
	_, end := d.extent(h)
	last := len(d.lines)
	if end < len(d.heads) {
		last = d.heads[end].line - 1
	}
	return d.lines[h.line:last]
}

func isFeature(h lintHeading) bool {
	return h.fields != nil && h.fields["type"] == "feature"
}

// normName folds a heading/link name for comparison. The books mix straight and
// curly apostrophes between a table cell and the heading it names.
func normName(s string) string {
	s = strings.ReplaceAll(s, "’", "'")
	return strings.ToLower(strings.TrimSpace(s))
}

// R1 -- sibling-run uniformity.
func lintSiblingRuns(t *testing.T, rel string, d *lintDoc) {
	var run []lintHeading

	flush := func() {
		defer func() { run = nil }()
		if len(run) < 3 {
			return
		}
		counts := map[int]int{}
		for _, h := range run {
			counts[h.level]++
		}
		if len(counts) == 1 {
			return
		}
		modal, modalN := 0, -1
		for lv, n := range counts {
			if n > modalN || (n == modalN && lv < modal) {
				modal, modalN = lv, n
			}
		}
		for _, h := range run {
			if h.level != modal {
				t.Errorf("%s:%d: %q is H%d but its sibling subclass-feature run is H%d — "+
					"a shallower heading escapes its container, rendering as raw prose and "+
					"adding a stray section-TOC entry (SC-173)",
					rel, h.line, h.name, h.level, modal)
			}
		}
	}

	for _, h := range d.heads {
		if isFeature(h) && h.fields["subclass"] != "" {
			run = append(run, h)
			continue
		}
		flush()
	}
	flush()
}

// container is one member-table container plus how its named members sit
// relative to it.
type container struct {
	head    lintHeading
	nested  int
	escaped []lintHeading
}

// collectContainers finds every feature whose subtree holds a member table and
// resolves where each named member's own section actually sits.
func collectContainers(d *lintDoc) []container {
	byName := map[string][]lintHeading{}
	for _, h := range d.heads {
		byName[normName(h.name)] = append(byName[normName(h.name)], h)
	}

	var out []container
	for _, c := range d.heads {
		if !isFeature(c) {
			continue
		}
		names := map[string]bool{}
		for _, l := range d.subtreeLines(c) {
			if !strings.HasPrefix(l, "|") {
				continue
			}
			for _, m := range lintSCCLinkRe.FindAllStringSubmatch(l, -1) {
				names[normName(m[1])] = true
			}
		}
		if len(names) < 2 {
			continue
		}

		cStart, cEnd := d.extent(c)
		limit := d.parentEnd(c)
		cur := container{head: c}
		for nm := range names {
			for _, t := range byName[nm] {
				if !isFeature(t) || t.idx <= c.idx {
					continue
				}
				switch {
				case t.idx >= cStart && t.idx < cEnd:
					cur.nested++
				case t.idx < limit:
					cur.escaped = append(cur.escaped, t)
				}
			}
		}
		if cur.nested > 0 || len(cur.escaped) > 0 {
			out = append(out, cur)
		}
	}
	return out
}

// R2 -- container membership, relative to the file's own convention.
func lintContainerMembership(t *testing.T, rel string, d *lintDoc) {
	cs := collectContainers(d)

	fileNests := false
	for _, c := range cs {
		if c.nested > 0 {
			fileNests = true
			break
		}
	}

	for _, c := range cs {
		if len(c.escaped) == 0 {
			continue
		}
		mixed := c.nested > 0
		if !mixed && !fileNests {
			// Uniformly flat container in a uniformly flat file: house style.
			continue
		}
		for _, m := range c.escaped {
			t.Errorf("%s:%d: %q is H%d but it is named in the %q (H%d) member table, "+
				"so it must be a child of that container (H%d), not its sibling (SC-173)",
				rel, m.line, m.name, m.level, c.head.name, c.head.level, c.head.level+1)
		}
	}
}

func TestBookHeadingStructure(t *testing.T) {
	paths, err := filepath.Glob("../../input/*/*.md")
	if err != nil || len(paths) == 0 {
		t.Skipf("skipping structural lint: no book sources found (%v)", err)
	}

	for _, path := range paths {
		data, rErr := os.ReadFile(path)
		if rErr != nil {
			t.Skipf("skipping structural lint: %v", rErr)
		}
		rel := filepath.ToSlash(path)
		d := parseLintDoc(string(data))

		lintSiblingRuns(t, rel, d)
		lintContainerMembership(t, rel, d)
	}
}
