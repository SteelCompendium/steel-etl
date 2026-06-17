# Callout Owner-Based Suppression Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let book callouts (annotated blockquotes) be hidden on the narrow entity page they incidentally landed on while still rendering on broader, book-faithful pages.

**Architecture:** A callout is marked `<!-- @type: callout | @owner: self|loose -->` immediately before its blockquote. `@owner: loose` callouts are stripped only from the **root body** of a `RenderSubtree` render (the page that is *about* the section the callout sits in); they survive in descendant bodies (class/chapter pages). `@owner: self` callouts are never stripped. All callout grammar lives in one new file, `internal/content/callout.go`; the only wiring is a one-flag change to `nodeBody` and a warning scan in `validate`.

**Tech Stack:** Go 1.26 (run via `devbox run -- …`), standard `go test`, `regexp`, `strings`.

---

## Background the engineer needs

- **Build/test must be run through devbox** — Go is not on PATH:
  - Build: `devbox run -- go build ./...`
  - Test a package: `devbox run -- go test ./internal/content/ -run TestName -v`
- `internal/content/render_subtree.go` turns a parsed `*parser.Section` tree into a single
  book-order markdown string (`PageBody`). Every site page (Browse leaf, Browse class page,
  Read chapter) is one `RenderSubtree` call **rooted at one section**. A callout that lives
  in section *C*'s body therefore appears in *C*'s **own** body when the page is rooted at
  *C*, and in a **descendant** body when the page is rooted at an ancestor of *C*. That is
  the entire basis for the suppression rule.
- `parser.Section` has fields `Heading string`, `HeadingLevel int`, `BodySource string`,
  `Children []*parser.Section`, and a method `Type() string` (reads `Annotation["type"]`).
- Callout comments are **body-level** — they sit inside `BodySource`, not in
  `Section.Annotation` — so the parser never treats `callout` as a section type and
  `validate`'s existing "unknown @type" check never fires for them.

---

## File Structure

- **Create** `internal/content/callout.go` — all callout grammar: regexes, `isLooseCalloutComment`, `stripLooseCallouts`, and `ScanCallouts` (+ `CalloutAnnotation`) for validation. One responsibility: understanding callout annotations.
- **Create** `internal/content/callout_test.go` — unit tests for the above.
- **Modify** `internal/content/render_subtree.go` — thread `isRoot bool` into `nodeBody`; call `stripLooseCallouts` when root.
- **Modify** `internal/content/render_subtree_test.go` — integration tests through `RenderSubtree`.
- **Modify** `internal/cli/validate.go` — warn on a callout missing/with-unknown `@owner`.
- **Modify** `input/summoner/Draw Steel Summoner.md` — tag the real "Minions and Treasures" callout `@owner: loose` (the concrete payoff).

---

## Task 1: Callout grammar (detection, strip, scan)

**Files:**
- Create: `internal/content/callout.go`
- Test: `internal/content/callout_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/content/callout_test.go`:

```go
package content

import (
	"reflect"
	"strings"
	"testing"
)

func TestIsLooseCalloutComment(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{"loose", "<!-- @type: callout | @owner: loose -->", true},
		{"loose trailing space", "<!-- @type: callout | @owner: loose --> ", true},
		{"loose reordered keys", "<!-- @owner: loose | @type: callout -->", true},
		{"self is not loose", "<!-- @type: callout | @owner: self -->", false},
		{"callout without owner", "<!-- @type: callout -->", false},
		{"unrelated comment", "<!-- @type: feature | @id: x -->", false},
		{"prose mentioning callout", "This callout explains loose treasure rules.", false},
		{"blockquote line", "> **Minions and Treasures**", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isLooseCalloutComment(c.line); got != c.want {
				t.Errorf("isLooseCalloutComment(%q) = %v, want %v", c.line, got, c.want)
			}
		})
	}
}

func TestStripLooseCallouts(t *testing.T) {
	t.Run("removes loose callout at end of body", func(t *testing.T) {
		body := "Para one.\n\nPara two.\n\n<!-- @type: callout | @owner: loose -->\n> **Title**\n>\n> Body of callout.\n> - bullet"
		got := stripLooseCallouts(body)
		want := "Para one.\n\nPara two."
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("removes loose callout in middle, collapses blanks", func(t *testing.T) {
		body := "Para one.\n\n<!-- @type: callout | @owner: loose -->\n> **Title**\n> line\n\nPara two."
		got := stripLooseCallouts(body)
		want := "Para one.\n\nPara two."
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("keeps self callout", func(t *testing.T) {
		body := "Para.\n\n<!-- @type: callout | @owner: self -->\n> **Alt Rule**\n> Use this instead."
		got := stripLooseCallouts(body)
		if !strings.Contains(got, "Alt Rule") {
			t.Errorf("self callout was stripped: %q", got)
		}
	})

	t.Run("keeps untagged blockquote", func(t *testing.T) {
		body := "Para.\n\n> *flavor quote*\n> — Someone"
		got := stripLooseCallouts(body)
		if got != body {
			t.Errorf("untagged blockquote altered: got %q", got)
		}
	})

	t.Run("strips loose, keeps adjacent untagged blockquote", func(t *testing.T) {
		body := "> *flavor*\n\n<!-- @type: callout | @owner: loose -->\n> **Drop me**\n> gone"
		got := stripLooseCallouts(body)
		want := "> *flavor*"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("body with no callout is unchanged", func(t *testing.T) {
		body := "Just some prose.\n\nWith two paragraphs."
		if got := stripLooseCallouts(body); got != body {
			t.Errorf("unchanged body altered: %q", got)
		}
	})
}

func TestScanCallouts(t *testing.T) {
	body := "x\n<!-- @type: callout | @owner: loose -->\n> a\n\ny\n<!-- @type: callout -->\n> b\n\nz\n<!-- @type: callout | @owner: bogus -->\n> c"
	got := ScanCallouts(body)
	want := []CalloutAnnotation{
		{Owner: "loose", HasOwner: true, OwnerKnown: true},
		{Owner: "", HasOwner: false, OwnerKnown: false},
		{Owner: "bogus", HasOwner: true, OwnerKnown: false},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d callouts, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Owner != want[i].Owner || got[i].HasOwner != want[i].HasOwner || got[i].OwnerKnown != want[i].OwnerKnown {
			t.Errorf("callout %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	_ = reflect.DeepEqual // keep import if future use
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `devbox run -- go test ./internal/content/ -run 'Callout' -v`
Expected: FAIL — `undefined: isLooseCalloutComment`, `stripLooseCallouts`, `ScanCallouts`, `CalloutAnnotation`.

- [ ] **Step 3: Implement `internal/content/callout.go`**

```go
package content

import (
	"regexp"
	"strings"
)

// Callout annotations are body-level directives (not section types):
//
//	<!-- @type: callout | @owner: self|loose -->
//	> blockquote text...
//
// @owner records what the callout semantically belongs to. `self` belongs to the
// immediate enclosing header and always renders; `loose` is incidental (the publisher
// just had whitespace there) and is stripped from the body of the page that is rooted
// at the section the callout sits in. Key order and trailing whitespace are tolerated,
// matching the parser's single-line annotation form.

// calloutKnownOwners is the Phase 1 value set. The space is intentionally open so a
// later coarser scope (e.g. "chapter") or an SCC reference can be added without a
// grammar change.
var calloutKnownOwners = map[string]bool{"self": true, "loose": true}

// calloutCommentLineRe matches a one-line HTML comment carrying @type: callout.
var calloutCommentLineRe = regexp.MustCompile(`^<!--.*@type:\s*callout\b.*-->\s*$`)

// ownerValueRe extracts the @owner value from such a comment.
var ownerValueRe = regexp.MustCompile(`@owner:\s*([\w-]+)`)

// blankLineRunRe collapses 3+ newlines (left after removing a callout from the middle
// of a body) back to a single blank line.
var blankLineRunRe = regexp.MustCompile(`\n{3,}`)

// CalloutAnnotation is a parsed callout comment, used by validation.
type CalloutAnnotation struct {
	Owner      string // @owner value, "" if absent
	HasOwner   bool
	OwnerKnown bool // Owner is in calloutKnownOwners
}

// isLooseCalloutComment reports whether a single line is a callout comment with
// @owner: loose.
func isLooseCalloutComment(line string) bool {
	t := strings.TrimSpace(line)
	if !calloutCommentLineRe.MatchString(t) {
		return false
	}
	m := ownerValueRe.FindStringSubmatch(t)
	return m != nil && m[1] == "loose"
}

// ScanCallouts returns one CalloutAnnotation per callout comment line in body.
func ScanCallouts(body string) []CalloutAnnotation {
	var out []CalloutAnnotation
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if !calloutCommentLineRe.MatchString(t) {
			continue
		}
		ca := CalloutAnnotation{}
		if m := ownerValueRe.FindStringSubmatch(t); m != nil {
			ca.Owner = m[1]
			ca.HasOwner = true
			ca.OwnerKnown = calloutKnownOwners[m[1]]
		}
		out = append(out, ca)
	}
	return out
}

// stripLooseCallouts removes every `@owner: loose` callout comment and the contiguous
// blockquote run that immediately follows it. Callouts with any other owner and untagged
// blockquotes are left untouched. Operates line-wise because a blockquote spans lines.
func stripLooseCallouts(body string) string {
	if !strings.Contains(body, "callout") { // cheap guard: nothing to do
		return body
	}
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		if !isLooseCalloutComment(lines[i]) {
			out = append(out, lines[i])
			continue
		}
		// Skip the comment line, any blank lines, then the blockquote run.
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), ">") {
			j++
		}
		i = j - 1 // for-loop ++ lands on the first line after the run
	}
	result := strings.Join(out, "\n")
	result = blankLineRunRe.ReplaceAllString(result, "\n\n")
	return strings.TrimSpace(result)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `devbox run -- go test ./internal/content/ -run 'Callout' -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/content/callout.go internal/content/callout_test.go
git commit -m "feat: callout annotation grammar (detect, strip, scan)"
```

---

## Task 2: Wire suppression into RenderSubtree (root-body only)

**Files:**
- Modify: `internal/content/render_subtree.go` (the `nodeBody` call at line ~36 and the `nodeBody` definition at line ~66)
- Test: `internal/content/render_subtree_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/content/render_subtree_test.go`:

```go
func TestRenderSubtree_CalloutSuppression(t *testing.T) {
	loose := "<!-- @type: callout | @owner: loose -->\n> **Incidental**\n> just landed here"
	self := "<!-- @type: callout | @owner: self -->\n> **Alt Rule**\n> use this instead"

	t.Run("loose callout stripped from root body", func(t *testing.T) {
		sec := &parser.Section{
			Heading: "Leader Formation", HeadingLevel: 4,
			Annotation: map[string]string{"type": "feature"},
			BodySource: "Feature text.\n\n" + loose,
		}
		got := RenderSubtree(sec, nil)
		if strings.Contains(got, "Incidental") {
			t.Errorf("loose callout should be stripped from root body, got:\n%s", got)
		}
		if !strings.Contains(got, "Feature text.") {
			t.Errorf("feature text missing, got:\n%s", got)
		}
	})

	t.Run("loose callout kept in descendant body", func(t *testing.T) {
		parent := &parser.Section{
			Heading: "Summoner", HeadingLevel: 2,
			Annotation: map[string]string{"type": "class"},
			BodySource: "Class intro.",
			Children: []*parser.Section{
				{
					Heading: "Leader Formation", HeadingLevel: 4,
					Annotation: map[string]string{"type": "feature"},
					BodySource: "Feature text.\n\n" + loose,
				},
			},
		}
		got := RenderSubtree(parent, nil)
		if !strings.Contains(got, "Incidental") {
			t.Errorf("loose callout should survive in descendant body, got:\n%s", got)
		}
	})

	t.Run("self callout kept in root body", func(t *testing.T) {
		sec := &parser.Section{
			Heading: "Some Rule", HeadingLevel: 4,
			Annotation: map[string]string{"type": "feature"},
			BodySource: "Rule text.\n\n" + self,
		}
		got := RenderSubtree(sec, nil)
		if !strings.Contains(got, "Alt Rule") {
			t.Errorf("self callout should be kept on its own page, got:\n%s", got)
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `devbox run -- go test ./internal/content/ -run TestRenderSubtree_CalloutSuppression -v`
Expected: FAIL — "loose callout should be stripped from root body" (suppression not wired yet).

- [ ] **Step 3: Wire `isRoot` into `nodeBody`**

In `internal/content/render_subtree.go`, change the root-body render call (currently
`if body := nodeBody(section); body != "" {`) to pass whether this section is the page root:

```go
	if body := nodeBody(section, section.HeadingLevel == rootLevel); body != "" {
		parts = append(parts, body)
	}
```

And change the `nodeBody` definition to accept and act on `isRoot`:

```go
// nodeBody returns a section's immediate body, un-blockquoted for ability
// sections (whose statblocks are blockquoted in source), with any overflow
// (7+ hash) heading demoted to bold. When isRoot is true (this section is the
// page's own root, not a descendant), incidental `@owner: loose` callouts are
// stripped — they belong to the section's broader context, not to its own page.
func nodeBody(section *parser.Section, isRoot bool) string {
	body := section.BodySource
	if section.Type() == "ability" {
		body = stripBlockquotePrefix(body)
	}
	if isRoot {
		body = stripLooseCallouts(body)
	}
	return demoteOverflowHeadings(body)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `devbox run -- go test ./internal/content/ -v`
Expected: PASS (new suppression tests plus all existing render_subtree tests still green).

- [ ] **Step 5: Commit**

```bash
git add internal/content/render_subtree.go internal/content/render_subtree_test.go
git commit -m "feat: strip @owner:loose callouts from root-body renders"
```

---

## Task 3: Validate warns on missing/unknown @owner

**Files:**
- Modify: `internal/cli/validate.go` (inside the `walkSections` closure, after the annotation block, ~line 111, before `walkSections(sec.Children, depth+1)`)
- Test: covered by `TestScanCallouts` (Task 1), which exercises the exact logic the warning is built from. The CLI wiring below is a thin adapter over `ScanCallouts`.

- [ ] **Step 1: Add the callout scan to the section walk**

In `internal/cli/validate.go`, inside `walkSections`, immediately before
`walkSections(sec.Children, depth+1)`:

```go
			for _, ca := range content.ScanCallouts(sec.BodySource) {
				if !ca.HasOwner {
					issues = append(issues, validationIssue{
						level:   "warn",
						heading: sec.Heading,
						hlevel:  sec.HeadingLevel,
						msg:     "callout missing @owner (expected self|loose)",
					})
				} else if !ca.OwnerKnown {
					issues = append(issues, validationIssue{
						level:   "warn",
						heading: sec.Heading,
						hlevel:  sec.HeadingLevel,
						msg:     fmt.Sprintf("callout has unknown @owner: %q (expected self|loose)", ca.Owner),
					})
				}
			}
```

(`content` and `fmt` are already imported in `validate.go`.)

- [ ] **Step 2: Build to verify it compiles**

Run: `devbox run -- go build ./...`
Expected: no output (success).

- [ ] **Step 3: Re-run the content scan tests**

Run: `devbox run -- go test ./internal/content/ -run TestScanCallouts -v`
Expected: PASS (the warning logic's source of truth).

- [ ] **Step 4: Commit**

```bash
git add internal/cli/validate.go
git commit -m "feat: validate warns on callout missing/unknown @owner"
```

---

## Task 4: Tag the real callout in the Summoner source

**Files:**
- Modify: `input/summoner/Draw Steel Summoner.md` (the "Minions and Treasures" callout, ~line 681)

- [ ] **Step 1: Add `@owner: loose` to the callout annotation**

Change the existing line (note: it currently has a trailing space):

```
<!-- @type: callout -->
```

to:

```
<!-- @type: callout | @owner: loose -->
```

This callout is incidental to Leader Formation (it concerns minions + treasures generally),
so `loose` hides it on the Leader Formation Browse leaf while keeping it on the class and
chapter pages.

- [ ] **Step 2: Verify end to end by regenerating the Summoner book**

Run:
```bash
devbox run -- go run ./cmd/steel-etl gen --book mcdm.summoner.v1 --config pipeline.yaml
```
Then check the leaf no longer has it but the chapter does:
```bash
grep -c "Minions and Treasures" ../data/data-summoner/en/md-linked/feature/summoner/level-1/leader-formation.md
grep -c "Minions and Treasures" ../data/data-summoner/en/md-linked/chapter/the-summoner-class.md
grep -c "Minions and Treasures" ../data/data-summoner/en/md-linked/class/summoner.md
```
Expected: `0` for the leaf; `1` (or more) for the chapter **and** the class page.

(`data/` is gitignored build output — do not commit it.)

- [ ] **Step 3: Run validate to confirm no new warnings for this callout**

Run:
```bash
devbox run -- go run ./cmd/steel-etl validate "input/summoner/Draw Steel Summoner.md"
```
Expected: the callout produces **no** warning (it now has a known `@owner`). Pre-existing
unrelated warnings/infos are fine.

- [ ] **Step 4: Commit the source tag**

```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "content: tag Minions and Treasures callout @owner: loose"
```

---

## Task 5: Full test + build sweep

- [ ] **Step 1: Run the full content package tests with race**

Run: `devbox run -- go test -race ./internal/content/ ./internal/cli/`
Expected: PASS.

- [ ] **Step 2: Build everything**

Run: `devbox run -- go build ./...`
Expected: success.

- [ ] **Step 3: Document the new annotation in ANNOTATION-GUIDE.md**

Add a short entry to `ANNOTATION-GUIDE.md` describing `@type: callout | @owner: self|loose`
as a **body-level** directive (not a section type): `self` = belongs to the immediate
header (always shown); `loose` = incidental, hidden on the section's own entity page but
shown on broader (class/chapter) pages. Note `@owner` is required and `validate` warns if
it is missing or unknown.

- [ ] **Step 4: Commit docs**

```bash
git add ANNOTATION-GUIDE.md
git commit -m "docs: document @type: callout / @owner annotation"
```

---

## Self-Review

- **Spec coverage:**
  - Grammar `@type: callout | @owner: self|loose`, owner required → Task 1 (grammar), Task 3 (required-warning), Task 5 (guide). ✅
  - Render rule (strip loose from root body only; self always kept; descendant kept; untagged untouched) → Task 1 (`stripLooseCallouts`) + Task 2 (`isRoot` wiring, integration tests for all four cases). ✅
  - Validation warnings (missing / unknown owner) → Task 3, logic tested in Task 1 `TestScanCallouts`. ✅
  - Grow-later open value space → `calloutKnownOwners` map + open `ownerValueRe` capture in Task 1. ✅
  - Concrete "Minions and Treasures" case incl. leaf-hidden / class+chapter-shown matrix → Task 4 end-to-end verification. ✅
  - Non-goals (no site-builder/schema/SCC change; kept-comment tidying out of scope) → respected; no such tasks. ✅
- **Placeholder scan:** none — every code/command step is complete.
- **Type consistency:** `CalloutAnnotation{Owner, HasOwner, OwnerKnown}`, `ScanCallouts`, `isLooseCalloutComment`, `stripLooseCallouts`, and `nodeBody(section, isRoot)` are named identically across Tasks 1–3.
