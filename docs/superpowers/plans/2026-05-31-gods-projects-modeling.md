# Deeper Modeling of Gods & Downtime Projects — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enrich the `god` and `project` parsers with structured fields, capture the discarded ancestry `(N Point)` trait cost, annotate 27 individual saints/heroes/evil-gods (nested under their patron god in SCC), and correct docs that overstate SCC codes as immutable.

**Architecture:** Pure-parser enrichments read fields already in each section body and add frontmatter. The 27 figures get `@type: god` annotations added to the source markdown; `GodParser` resolves each figure's patron by walking the real section tree (`section.Parent`) for the nearest ancestor `god`, producing a nested type path `god/<patron>/<id>` (flat `god/<id>` when there is no patron). All changes are additive — no existing SCC code changes.

**Tech Stack:** Go 1.26 (via devbox), standard `go test` (table-driven), the steel-etl content-parser registry.

**Toolchain note:** Go is not on PATH. Prefix every Go command with the devbox activation, e.g.
`devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestX -v'`
(run from the workspace root `/home/vexa/code/steel_compendium/workspace`).

**Reference spec:** `steel-etl/docs/superpowers/specs/2026-05-31-gods-projects-modeling-design.md`

---

## File Structure

- `steel-etl/internal/content/god.go` — MODIFY: add `domains` extraction + patron/nested path.
- `steel-etl/internal/content/god_test.go` — CREATE: god parser tests.
- `steel-etl/internal/content/project.go` — MODIFY: extract four project fields.
- `steel-etl/internal/content/project_test.go` — CREATE: project parser tests.
- `steel-etl/internal/content/helpers.go` — MODIFY: add `extractCostSuffix` + `findParentGodID`.
- `steel-etl/internal/content/helpers_test.go` — MODIFY: add tests for the two helpers.
- `steel-etl/internal/content/feature.go` — MODIFY: set `cost` from the heading suffix.
- `steel-etl/internal/content/feature_test.go` — CREATE: feature cost test.
- `steel-etl/input/heroes/Draw Steel Heroes.md` — MODIFY: add 27 `@type: god` annotations.
- `steel-etl/README.md`, workspace `CLAUDE.md`, `ARCHITECTURE.md` — MODIFY: freeze wording.

---

## Task 1: God `domains` field

**Files:**
- Modify: `steel-etl/internal/content/god.go`
- Create: `steel-etl/internal/content/god_test.go`

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/content/god_test.go`:

```go
package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestGodParser_Domains(t *testing.T) {
	p := &GodParser{}
	section := &parser.Section{
		Heading:      "Val",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "god", "id": "val"},
		BodySource:   "**Domains:** Creation, Knowledge, Life, Nature, Protection\n\nVal, the Noble Lord.",
	}

	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	domains, ok := result.Frontmatter["domains"].([]string)
	if !ok {
		t.Fatalf("domains not a []string: %T", result.Frontmatter["domains"])
	}
	want := []string{"Creation", "Knowledge", "Life", "Nature", "Protection"}
	if len(domains) != len(want) {
		t.Fatalf("domains = %v, want %v", domains, want)
	}
	for i := range want {
		if domains[i] != want[i] {
			t.Errorf("domains[%d] = %q, want %q", i, domains[i], want[i])
		}
	}
}

func TestGodParser_NoDomains(t *testing.T) {
	p := &GodParser{}
	section := &parser.Section{
		Heading:      "Nameless",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "god", "id": "nameless"},
		BodySource:   "A god with no listed domains.",
	}
	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if _, ok := result.Frontmatter["domains"]; ok {
		t.Error("domains should be absent when no Domains line is present")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestGodParser_Domains -v'`
Expected: FAIL — `domains not a []string` (field not yet set).

- [ ] **Step 3: Implement domains extraction in `god.go`**

Replace the body of `GodParser.Parse` so it adds domains. The full file becomes:

```go
package content

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// GodParser handles @type: god sections (deities in the Gods and Religion chapter).
type GodParser struct{}

func (p *GodParser) Type() string { return "god" }

func (p *GodParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	fm := map[string]any{
		"name": name,
		"type": "god",
	}

	body := section.FullBodySource()

	// Domains: "**Domains:** Creation, Life" -> ["Creation", "Life"]
	if raw := extractField(body, "Domains"); raw != "" {
		var domains []string
		for _, d := range strings.Split(raw, ",") {
			if d = strings.TrimSpace(d); d != "" {
				domains = append(domains, d)
			}
		}
		if len(domains) > 0 {
			fm["domains"] = domains
		}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"god"},
		ItemID:      id,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestGodParser -v'`
Expected: PASS (both `TestGodParser_Domains` and `TestGodParser_NoDomains`).

- [ ] **Step 5: Commit**

```bash
cd steel-etl && git add internal/content/god.go internal/content/god_test.go && \
git commit -m "feat: extract domains list in GodParser"
```

---

## Task 2: Patron resolution + nested SCC path for gods

**Files:**
- Modify: `steel-etl/internal/content/helpers.go` (add `findParentGodID`)
- Modify: `steel-etl/internal/content/god.go` (use it)
- Modify: `steel-etl/internal/content/god_test.go` (patron tests)

- [ ] **Step 1: Write the failing test**

Append to `steel-etl/internal/content/god_test.go`:

```go
func TestGodParser_PatronNestedPath(t *testing.T) {
	// Val (god, L3) -> "Heroes of the Elves" (unannotated container, L4)
	//   -> "A Sea of Suns" (god, L5)
	val := &parser.Section{
		Heading: "Val", HeadingLevel: 3,
		Annotation: map[string]string{"type": "god", "id": "val"},
	}
	container := &parser.Section{
		Heading: "Heroes of the Elves", HeadingLevel: 4,
		Parent: val,
	}
	saint := &parser.Section{
		Heading: "A Sea of Suns", HeadingLevel: 5,
		Annotation: map[string]string{"type": "god", "id": "a-sea-of-suns"},
		BodySource: "**Domains:** Creation, Life\n\nThe Composer.",
		Parent:     container,
	}

	p := &GodParser{}
	result, err := p.Parse(context.NewContextStack(nil), saint)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	wantPath := []string{"god", "val"}
	if len(result.TypePath) != 2 || result.TypePath[0] != wantPath[0] || result.TypePath[1] != wantPath[1] {
		t.Errorf("TypePath = %v, want %v", result.TypePath, wantPath)
	}
	if result.Frontmatter["patron"] != "val" {
		t.Errorf("patron = %v, want val", result.Frontmatter["patron"])
	}
	if result.ItemID != "a-sea-of-suns" {
		t.Errorf("ItemID = %q, want a-sea-of-suns", result.ItemID)
	}
}

func TestGodParser_NoPatronFlatPath(t *testing.T) {
	// Evil God under unannotated containers only -> flat path, no patron.
	humanGods := &parser.Section{Heading: "Human Gods of Vasloria", HeadingLevel: 3}
	evilGods := &parser.Section{Heading: "Evil Gods", HeadingLevel: 4, Parent: humanGods}
	nikros := &parser.Section{
		Heading: "Nikros the Tyrant", HeadingLevel: 5,
		Annotation: map[string]string{"type": "god", "id": "nikros"},
		BodySource: "**Domains:** Death, Fate, Storm, War\n\nThe Tyrant.",
		Parent:     evilGods,
	}

	p := &GodParser{}
	result, err := p.Parse(context.NewContextStack(nil), nikros)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(result.TypePath) != 1 || result.TypePath[0] != "god" {
		t.Errorf("TypePath = %v, want [god]", result.TypePath)
	}
	if _, ok := result.Frontmatter["patron"]; ok {
		t.Error("patron should be absent for a god with no ancestor god")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestGodParser_Patron -v'`
Expected: FAIL — compile error `undefined: findParentGodID` is not yet referenced, but the test fails because `TypePath` is `[god]` not `[god val]` and `patron` is absent.

- [ ] **Step 3: Add `findParentGodID` to `helpers.go`**

Add this function to `steel-etl/internal/content/helpers.go` (after `findAncestorID`), and add `"github.com/SteelCompendium/steel-etl/internal/parser"` to its imports:

```go
// findParentGodID walks the section tree upward via Parent pointers and returns
// the @id of the nearest ancestor whose @type is "god", or "" if none.
// Uses the real tree (not the context stack) so unannotated containers between a
// saint and its patron god are skipped without leaking stale stack entries.
func findParentGodID(section *parser.Section) string {
	for p := section.Parent; p != nil; p = p.Parent {
		if p.Type() == "god" {
			return p.ID()
		}
	}
	return ""
}
```

The import block of `helpers.go` becomes:

```go
import (
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)
```

- [ ] **Step 4: Use it in `god.go`**

In `god.go`, replace the return block's `TypePath` construction. Change the end of `Parse` from:

```go
	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"god"},
		ItemID:      id,
	}, nil
```

to:

```go
	typePath := []string{"god"}
	if patron := findParentGodID(section); patron != "" {
		typePath = append(typePath, patron)
		fm["patron"] = patron
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    typePath,
		ItemID:      id,
	}, nil
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestGodParser -v'`
Expected: PASS (all four god tests). The earlier `TestGodParser_Domains`/`NoDomains` sections have no Parent, so `findParentGodID` returns "" and they keep the flat `[god]` path.

- [ ] **Step 6: Commit**

```bash
cd steel-etl && git add internal/content/helpers.go internal/content/god.go internal/content/god_test.go && \
git commit -m "feat: resolve god patron via section tree, nest saint SCC paths"
```

---

## Task 3: Project structured fields

**Files:**
- Modify: `steel-etl/internal/content/project.go`
- Create: `steel-etl/internal/content/project_test.go`

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/content/project_test.go`:

```go
package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestProjectParser_Fields(t *testing.T) {
	p := &ProjectParser{}
	section := &parser.Section{
		Heading:      "Build Airship",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "project"},
		BodySource: "**Item Prerequisite:** Wind Crystal of Quintessence\n\n" +
			"**Project Source:** Texts or lore in Low Rhyvian\n\n" +
			"**Project Roll Characteristic:** Might, Reason, or Presence\n\n" +
			"**Project Goal:** 3,000\n\nWhen you start this project...",
	}

	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	checks := map[string]string{
		"prerequisite":        "Wind Crystal of Quintessence",
		"project_source":      "Texts or lore in Low Rhyvian",
		"roll_characteristic": "Might, Reason, or Presence",
		"project_goal":        "3,000",
	}
	for key, want := range checks {
		if got := result.Frontmatter[key]; got != want {
			t.Errorf("%s = %v, want %q", key, got, want)
		}
	}
}

func TestProjectParser_GoalVariesAndOmitted(t *testing.T) {
	p := &ProjectParser{}
	section := &parser.Section{
		Heading:      "Build or Repair Road",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "project"},
		BodySource:   "**Project Goal:** Varies\n\nWhen you start this project...",
	}
	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Frontmatter["project_goal"] != "Varies" {
		t.Errorf("project_goal = %v, want Varies", result.Frontmatter["project_goal"])
	}
	if _, ok := result.Frontmatter["prerequisite"]; ok {
		t.Error("prerequisite should be absent when not present in body")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestProjectParser -v'`
Expected: FAIL — fields are nil (not yet extracted).

- [ ] **Step 3: Implement field extraction in `project.go`**

Rewrite `ProjectParser.Parse` so the full file becomes:

```go
package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ProjectParser handles @type: project sections (downtime projects).
type ProjectParser struct{}

func (p *ProjectParser) Type() string { return "project" }

func (p *ProjectParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	fm := map[string]any{
		"name": name,
		"type": "project",
	}

	body := section.FullBodySource()

	// Regular bold-label fields. Each is set only when present.
	if v := extractField(body, "Item Prerequisite"); v != "" {
		fm["prerequisite"] = v
	}
	if v := extractField(body, "Project Source"); v != "" {
		fm["project_source"] = v
	}
	if v := extractField(body, "Project Roll Characteristic"); v != "" {
		fm["roll_characteristic"] = v
	}
	if v := extractField(body, "Project Goal"); v != "" {
		fm["project_goal"] = v
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"project"},
		ItemID:      id,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestProjectParser -v'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd steel-etl && git add internal/content/project.go internal/content/project_test.go && \
git commit -m "feat: extract structured fields in ProjectParser"
```

---

## Task 4: Ancestry purchased-trait point cost

**Files:**
- Modify: `steel-etl/internal/content/helpers.go` (add `extractCostSuffix`)
- Modify: `steel-etl/internal/content/helpers_test.go` (helper test)
- Modify: `steel-etl/internal/content/feature.go` (set `cost`)
- Create: `steel-etl/internal/content/feature_test.go` (feature cost test)

- [ ] **Step 1: Write the failing helper test**

Append to `steel-etl/internal/content/helpers_test.go`:

```go
func TestExtractCostSuffix(t *testing.T) {
	cases := map[string]string{
		"Barbed Tail (1 Point)":          "1 Point",
		"Impressive Horns (2 Points)":    "2 Points",
		"Alacrity of the Heart (11 Piety)": "11 Piety",
		"Growing Ferocity":               "",
		"No Suffix Here":                 "",
	}
	for heading, want := range cases {
		if got := extractCostSuffix(heading); got != want {
			t.Errorf("extractCostSuffix(%q) = %q, want %q", heading, got, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestExtractCostSuffix -v'`
Expected: FAIL — `undefined: extractCostSuffix`.

- [ ] **Step 3: Add `extractCostSuffix` to `helpers.go`**

The existing `costSuffixRe` (`\s*\(\d+\s+\w+\)\s*$`) matches the whole suffix. Add a capturing helper next to `CleanHeading`:

```go
// costSuffixInnerRe captures the inner text of a trailing "(N Unit)" cost suffix.
var costSuffixInnerRe = regexp.MustCompile(`\((\d+\s+\w+)\)\s*$`)

// extractCostSuffix returns the inner text of a trailing "(… cost …)" suffix on a
// heading, e.g. "Barbed Tail (1 Point)" -> "1 Point". Returns "" when absent.
func extractCostSuffix(s string) string {
	m := costSuffixInnerRe.FindStringSubmatch(strings.TrimSpace(s))
	if len(m) == 2 {
		return m[1]
	}
	return ""
}
```

- [ ] **Step 4: Run helper test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestExtractCostSuffix -v'`
Expected: PASS.

- [ ] **Step 5: Write the failing feature test**

Create `steel-etl/internal/content/feature_test.go`:

```go
package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestFeatureParser_PointCost(t *testing.T) {
	p := &FeatureParser{}
	section := &parser.Section{
		Heading:      "Barbed Tail (1 Point)",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "Your pointy tail allows you to punctuate all your actions.",
	}
	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Frontmatter["name"] != "Barbed Tail" {
		t.Errorf("name = %v, want Barbed Tail", result.Frontmatter["name"])
	}
	if result.Frontmatter["cost"] != "1 Point" {
		t.Errorf("cost = %v, want 1 Point", result.Frontmatter["cost"])
	}
}

func TestFeatureParser_NoCost(t *testing.T) {
	p := &FeatureParser{}
	section := &parser.Section{
		Heading:      "Growing Ferocity",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "You gain ferocity benefits.",
	}
	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if _, ok := result.Frontmatter["cost"]; ok {
		t.Error("cost should be absent when the heading has no cost suffix")
	}
}
```

- [ ] **Step 6: Run feature test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestFeatureParser_ -v'`
Expected: FAIL — `cost` is nil.

- [ ] **Step 7: Set `cost` in `feature.go`**

In `FeatureParser.Parse` (`feature.go`), the `fm` map is built starting around line 54 with `name`/`type`. Immediately after the `fm` literal is created, add the cost capture. Find:

```go
	fm := map[string]any{
		"name": cleanName,
		"type": "trait",
	}
```

and insert directly below it:

```go
	if cost := extractCostSuffix(section.Heading); cost != "" {
		fm["cost"] = cost
	}
```

(`cleanName` is already `CleanHeading(section.Heading)` from the top of the function, so the name stays suffix-free while `cost` captures the stripped value.)

- [ ] **Step 8: Run feature tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestFeatureParser_ -v'`
Expected: PASS (both).

- [ ] **Step 9: Commit**

```bash
cd steel-etl && git add internal/content/helpers.go internal/content/helpers_test.go internal/content/feature.go internal/content/feature_test.go && \
git commit -m "feat: capture (N Point) trait cost from heading suffix"
```

---

## Task 5: Annotate the 27 saints/heroes/evil-gods in source

**Files:**
- Modify: `steel-etl/input/heroes/Draw Steel Heroes.md`

For each figure below, insert a single line directly **above** its heading:
`<!-- @type: god | @id: <id> -->`

Match on the exact heading text (do not rely on line numbers — they shift as you
insert). All headings are unique. Headings already carry their `#####`/`####`
prefix; leave the heading text unchanged.

| Heading (exact)                          | `@id`                |
|------------------------------------------|----------------------|
| `##### A Sea of Suns`                    | `a-sea-of-suns`      |
| `##### The Taste of Morning`             | `taste-of-morning`   |
| `##### Ripples of Honey on a Shore of Gold` | `ripples-of-honey` |
| `##### Yllin Dyrvis`                     | `yllin-dyrvis`       |
| `##### Thyll Hylacae`                    | `thyll-hylacae`      |
| `##### Illwyv li Orchiax`                | `illwyv-li-orchiax`  |
| `##### Zarok the Law-Giver`              | `zarok`              |
| `##### Valak-koth the Seeker`            | `valak-koth`         |
| `##### Stakros the Engineer`             | `stakros`            |
| `##### Khorvath Who Slew a Thousand`     | `khorvath`           |
| `##### Grole the One-Handed`             | `grole`              |
| `##### Khravila Who Ran Forty Leagues`   | `khravila`           |
| `##### Mahsiti the Weaver`               | `mahsiti`            |
| `##### Prexaspes the Stargazer`          | `prexaspes`          |
| `##### Atossa the Shepherd`              | `atossa`             |
| `##### Uryal the Subtle`                 | `uryal`              |
| `##### Kuryalka the False Principle`     | `kuryalka`           |
| `##### Gaed the Confessor`               | `gaed`               |
| `##### Gryffyn the Stout`                | `gryffyn`            |
| `##### Llewellyn the Valiant`            | `llewellyn`          |
| `##### Gwenllian the Fell-Handed`        | `gwenllian`          |
| `##### Draighen the Warden`              | `draighen`           |
| `##### Eriarwen the Wroth`               | `eriarwen`           |
| `##### Nikros the Tyrant`                | `nikros`             |
| `##### Pentalion the Paladin`            | `pentalion`          |
| `##### Cyrvis`                           | `cyrvis`             |
| `##### Eseld of the Eye`                 | `eseld`              |

- [ ] **Step 1: Capture the baseline god-code set**

Run from the workspace root:

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl classify --config pipeline.yaml 2>/dev/null | grep "/god" | sort > /tmp/gods_before.txt'; wc -l /tmp/gods_before.txt
```

Expected: 9 lines (the existing gods). If `classify` needs different flags, run
`go run ./cmd/steel-etl classify --help` and adjust; the goal is a sorted list of
current `god` codes.

- [ ] **Step 2: Add the 27 annotation lines**

Edit `steel-etl/input/heroes/Draw Steel Heroes.md`. For each row in the table,
insert `<!-- @type: god | @id: <id> -->` on its own line immediately above the
matching heading. Insert a blank line before the comment if the preceding line is
not already blank (match the existing annotation style, e.g. the block above
`### Val`).

- [ ] **Step 3: Regenerate and diff the god codes**

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl classify --config pipeline.yaml 2>/dev/null | grep "/god" | sort > /tmp/gods_after.txt'
diff /tmp/gods_before.txt /tmp/gods_after.txt
```

Expected: exactly 27 additions (`>` lines), 0 removals (`<` lines). Spot-check
that nested and flat forms are correct, e.g.:

```
> mcdm.heroes.v1/god/val/a-sea-of-suns
> mcdm.heroes.v1/god/kul/mahsiti
> mcdm.heroes.v1/god/thellasko/uryal
> mcdm.heroes.v1/god/adun/gaed
> mcdm.heroes.v1/god/nikros
> mcdm.heroes.v1/god/pentalion
```

(`mahsiti` under `kul` confirms Heroes of the Hakaan resolves to patron Kul;
`nikros`/`pentalion` are flat, confirming Evil Gods have no patron.)

- [ ] **Step 4: Verify the full pipeline still builds and the registry only grew**

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml && go run ./cmd/steel-etl validate --config pipeline.yaml --scc-stable'
```

Expected: gen succeeds; validate reports no missing frozen codes (additions are
allowed). If `validate` flags missing codes, STOP — an existing code changed;
re-check the annotations did not alter any existing heading.

- [ ] **Step 5: Commit**

```bash
cd steel-etl && git add "input/heroes/Draw Steel Heroes.md" classification.json 2>/dev/null; \
git add "input/heroes/Draw Steel Heroes.md" && \
git commit -m "feat: annotate 27 saints/heroes/evil-gods as god entries"
```

(`classification.json` is gitignored per CLAUDE.md; the second `git add` of just
the `.md` is the authoritative staged change. Do not force-add the registry.)

---

## Task 6: Correct "frozen / immutable" SCC documentation

**Files:**
- Modify: `steel-etl/README.md`

The registry is additive-only: `Registry.ValidateAgainstFrozen` only checks that
existing codes are **not removed** — new codes may be added freely. Reword the
overstated "frozen / permanent / immutable" language to reflect that.

A pre-check confirmed the only doc that overstates this is `steel-etl/README.md`.
Workspace `CLAUDE.md` and `ARCHITECTURE.md` contain no frozen/immutable/permanent
SCC language. `steel-etl/CLAUDE.md` mentions "freeze enforcement" and the
`classification.freeze` setting — those accurately describe the mechanism and
must be left as-is (do not reword them).

- [ ] **Step 1: Reword `steel-etl/README.md`**

Make these edits (match each `old` exactly, including surrounding text):

1. Line ~5:
   - old: `SCC taxonomy frozen (1,432 codes).`
   - new: `SCC taxonomy stabilized (additive-only; existing codes are not removed or renamed).`
2. Line ~31:
   - old: `│   ├── scc/                       # SCC classifier and registry (frozen)`
   - new: `│   ├── scc/                       # SCC classifier and registry (additive-only)`
3. Line ~63:
   - old: `Every classified item gets a permanent SCC identifier:`
   - new: `Every classified item gets a stable SCC identifier:`
4. Line ~79:
   - old: `SCCs become permanent URLs (`steelcompendium.io/mcdm.heroes.v1/feature.ability.fury.level-1/brutal-slam`) and are immutable once frozen. See `plans/architecture-redesign/scc-taxonomy.md` for the full taxonomy.`
   - new: `SCCs become stable URLs (`steelcompendium.io/mcdm.heroes.v1/feature.ability.fury.level-1/brutal-slam`). Existing codes are not removed or renamed once published, but new codes may be added as content is modeled. See `plans/architecture-redesign/scc-taxonomy.md` for the full taxonomy.`
5. Line ~174:
   - old: `- **SCC Freeze** -- 1,432 codes frozen, validate/classify commands ✓`
   - new: `- **SCC stability** -- additive-only registry guard (no removals/renames), validate/classify commands ✓`

- [ ] **Step 2: Verify nothing in README still overstates it**

```bash
grep -niE "frozen|immutable|permanent" steel-etl/README.md
```

Expected: no remaining lines asserting SCC codes are frozen/immutable/permanent.
(The `pipeline.yaml` `freeze: true` config setting and the `internal/scc` code
comments are out of scope — this task is docs only.)

- [ ] **Step 3: Commit (README.md is in the steel-etl submodule)**

```bash
cd steel-etl && git add README.md && \
git commit -m "docs: SCC registry is additive-only, not frozen/immutable"
```

---

## Task 7: Full verification & integration

**Files:** none (verification only)

- [ ] **Step 1: Run the full content + pipeline test suite with race**

```bash
devbox run -- bash -c 'cd steel-etl && go test -race ./internal/content/ ./internal/pipeline/ ./internal/scc/'
```

Expected: `ok` for each package.

- [ ] **Step 2: Run the entire suite**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./...'
```

Expected: no failures. Investigate any regression before proceeding (the
project/god/feature output changes may touch golden/snapshot tests; update any
that legitimately reflect the new structured fields, but do NOT loosen
conformance tests for abilities/traits).

- [ ] **Step 3: Vet**

```bash
devbox run -- bash -c 'cd steel-etl && go vet ./...'
```

Expected: clean.

- [ ] **Step 4: Update FOLLOWUPS.md and finish the branch**

Remove the "Deeper modeling of gods and downtime projects" entry from the
workspace `FOLLOWUPS.md`. Then follow the `superpowers:finishing-a-development-branch`
skill to integrate (commit the submodule pointer bump in the workspace repo and
push both repos).

```bash
cd /home/vexa/code/steel_compendium/workspace && \
git add FOLLOWUPS.md steel-etl && \
git commit -m "chore: resolve gods/projects modeling follow-up"
```

---

## Notes for the implementer

- **Why patron uses the tree, not the context stack:** see the spec's "Why
  tree-walk" section. The `ContextStack.Push` clears only equal-or-deeper levels,
  so an unannotated container cannot clear a stale ancestor entry; the Evil Gods
  would inherit the previous god's id. `section.Parent` reflects true nesting.
- **Existing gods' body shrinks:** once saints are annotated, each parent god's
  `FullBodySource()` no longer folds in the saint text (annotated children are
  excluded). The god's *reading page* (`PageBody` via `RenderSubtree`) still shows
  saints inline in book order — only the structured `body` field changes. This is
  intended and does not affect any existing SCC code.
- **`extractField` is case- and whitespace-sensitive** to the label prefix; the
  project labels match the source exactly ("Item Prerequisite", "Project Source",
  "Project Roll Characteristic", "Project Goal").
