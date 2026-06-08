# Card ⇄ Data Field Parity (steel-etl) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote the fields the index-page cards currently scrape from the rendered page **body** (`flavor` for every card type, plus treasure `project_goal` / `project_roll_characteristic` / `echelon`) into structured frontmatter so they flow into the JSON/YAML data outputs and the schemas, with the **parser as the single source of truth** and the cards reading the frontmatter field.

**Architecture:** The passthrough content parsers (`class`, `ancestry`, `career`, `treasure`, `culture`, `perk`, `title`, `complication`, `kit`) already emit frontmatter that `transformPassthrough`/`transformKit` copy verbatim into JSON/YAML via `copyFrontmatter`. So adding `fm["flavor"] = …` in a parser automatically lands the field in the data outputs — the remaining work is (1) extract the field once in the parser, (2) declare it in **both** hand-synced schema copies, (3) update the schema-validation allowlist, and (4) refactor `cards.go` to read the frontmatter field (with a body fallback so older/edge pages never render blank). This plan covers steel-etl + both schema copies only. The `data-sdk-npm` consumer side (DTOs, model classes, IO readers/writers, TS tests) is a **separate plan**: `2026-06-08-card-data-field-parity-sdk.md`.

**Tech Stack:** Go 1.26 (devbox), `go test` table-driven tests, JSON Schema draft 2019-09 (`unevaluatedProperties: false`).

---

## Design & Decisions

These were settled during planning (2026-06-08):

1. **`flavor` is the first prose paragraph of the body, markdown-stripped.** This matches the existing ability `flavor` convention (a clean descriptor, asterisks/links removed) and exactly reproduces what every card already *displays* (the cards run `firstProse` → `stripMD` today). Rich markdown remains available to consumers via the `content` field.
2. **Parser is the single source.** Extraction logic moves into one shared `content` helper (`firstFlavorParagraph`). `cards.go` reads `fm["flavor"]` and keeps its existing body-scrape (`firstProse` / `careerFlavor` / `complicationFlavor`) only as a **fallback** when the field is absent — zero output regression, field is primary.
3. **`class` card is unchanged.** The class card renders the *full multi-paragraph intro* (`classIntro`), which is richer than a scalar `flavor`. Class still **gains a `flavor` data field** (first paragraph) for cross-type consistency, but `classCard()` keeps rendering `classIntro` from the body.
4. **`perk` gains a `flavor` field** (first prose paragraph) for schema symmetry, even though the perk card shows the full prose block. The perk card is left unchanged; the field is additive in the data only.
5. **Treasure gains three structured fields:** `flavor` (italic descriptor), `project_goal`, and `project_roll_characteristic` (both already declared in the schema but never populated), plus `echelon` is **added to the schema** (the parser already emits it — a reverse gap where data was ahead of the contract).
6. **Body-prose detection for the data field excludes bold `**Label:**` stat lines** (e.g. `**Level:**`, `**Benefit:**`). These look like prose to the card's looser `isProse` but are structured data; excluding them makes the parser-side extraction more correct and removes the need for per-type skip lists.

### Field/schema matrix (what changes where)

| Type | New `fm` field(s) emitted by parser | Schema already has it? | Schema edit needed | Allowlist edit needed | Card refactor |
|------|-------------------------------------|------------------------|--------------------|-----------------------|---------------|
| class | `flavor` | yes | — | — | none (keeps `classIntro`) |
| ancestry | `flavor` | yes | — | — | read field |
| career | `flavor` | yes | — | — | read field |
| culture | `flavor` | **no** | add `flavor` (×2) | add `flavor` | read field |
| title | `flavor` | yes | — | — | read field |
| complication | `flavor` | yes | — | — | read field |
| perk | `flavor` | **no** | add `flavor` (×2) | add `flavor` | none (additive) |
| kit | `flavor` | yes | — | — | read field |
| treasure | `flavor`, `project_goal`, `project_roll_characteristic` | `flavor`/`project_*` yes; `echelon` **no** | add `echelon` (×2) | add `echelon` | read fields |

"×2" = both `steel-etl/schemas/*.schema.json` **and** `data-sdk-npm/src/schema/*.schema.json` (hand-synced copies — see workspace `ARCHITECTURE.md` → "Schemas: two hand-synced copies"). `project_goal` / `project_roll_characteristic` are already in the treasure allowlist (`schema_validation_test.go`).

---

## File Structure

**Create:**
- `steel-etl/internal/content/flavor.go` — the single `firstFlavorParagraph` extractor + its `isFlavorProse` / `stripInlineMarkdown` helpers.
- `steel-etl/internal/content/flavor_test.go` — unit tests for the extractor + per-parser flavor-emission tests.
- `steel-etl/docs/card-data-parity.md` — the standing rule + checklist for future card/parser changes.

**Modify (parsers — add `fm["flavor"]`, treasure also project fields):**
- `steel-etl/internal/content/class.go`, `ancestry.go`, `career.go`, `culture.go`, `title.go`, `complication.go`, `perk.go`, `kit.go`, `treasure.go`

**Modify (schemas — both copies):**
- `steel-etl/schemas/culture.schema.json` + `data-sdk-npm/src/schema/culture.schema.json` (add `flavor`)
- `steel-etl/schemas/perk.schema.json` + `data-sdk-npm/src/schema/perk.schema.json` (add `flavor`)
- `steel-etl/schemas/treasure.schema.json` + `data-sdk-npm/src/schema/treasure.schema.json` (add `echelon`)

**Modify (validation + cards + docs):**
- `steel-etl/internal/output/schema_validation_test.go` (allowlists + new assertion cases)
- `steel-etl/internal/site/cards.go` (read `fm["flavor"]`; treasure project fields) + `steel-etl/internal/site/cards_test.go`
- `steel-etl/CLAUDE.md`, workspace `ARCHITECTURE.md`

> **Run commands from the workspace root** (`/home/vexa/code/steel_compendium/workspace`). Go is under devbox; the pattern is:
> `devbox run -- bash -c "cd steel-etl && go test ./internal/content/..."`

---

## Task 1: The shared flavor extractor

**Files:**
- Create: `steel-etl/internal/content/flavor.go`
- Test: `steel-etl/internal/content/flavor_test.go`

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/content/flavor_test.go`:

```go
package content

import "testing"

func TestFirstFlavorParagraph(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			"plain prose first paragraph",
			"You were born to the road, never staying in one place.\n\n**Skills:** Nature",
			"You were born to the road, never staying in one place.",
		},
		{
			"italic treasure descriptor",
			"*A worn leather bag that holds far more than its size suggests.*\n\n**Keywords:** Magic",
			"A worn leather bag that holds far more than its size suggests.",
		},
		{
			"skips heading then returns prose",
			"#### Flavor\n\nAn ancient order of knights.",
			"An ancient order of knights.",
		},
		{
			"skips bold stat line",
			"**Level:** 3\n\nThis blade hums with power.",
			"This blade hums with power.",
		},
		{
			"strips links and emphasis",
			"You can become [frightened](rule.combat/frightened.md) by **nothing**.",
			"You can become frightened by nothing.",
		},
		{
			"skips blockquote, table, list, rule",
			"---\n\n> a quote\n\n| a | b |\n\n- item\n\nReal flavor here.",
			"Real flavor here.",
		},
		{"empty body", "", ""},
		{"no prose at all", "#### Heading\n\n**Level:** 1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstFlavorParagraph(tt.body); got != tt.want {
				t.Errorf("firstFlavorParagraph() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/content/ -run TestFirstFlavorParagraph"`
Expected: FAIL — `undefined: firstFlavorParagraph`.

- [ ] **Step 3: Write the implementation**

Create `steel-etl/internal/content/flavor.go`:

```go
package content

import (
	"regexp"
	"strings"
)

// contentMdLinkRe matches a markdown link [text](target) for stripping.
var contentMdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

// firstFlavorParagraph returns the first prose paragraph of a section body,
// markdown-stripped, for use as the structured `flavor` field. It mirrors what
// the index-page cards display (first prose paragraph, links/emphasis removed)
// so the data field and the card stay in lockstep — the parser is the single
// source of truth and cards.go reads this value back from frontmatter.
//
// "Prose" excludes headings, tables, blockquotes, list items, horizontal
// rules, and bold "**Label:** value" stat lines (which look like prose but are
// structured data the parser lifts into their own fields).
func firstFlavorParagraph(body string) string {
	for _, raw := range strings.Split(body, "\n") {
		t := strings.TrimSpace(raw)
		if !isFlavorProse(t) {
			continue
		}
		if s := stripInlineMarkdown(t); s != "" {
			return s
		}
	}
	return ""
}

// isFlavorProse reports whether a trimmed line is flavor prose (not a heading,
// table, blockquote, list item, horizontal rule, or bold stat line).
func isFlavorProse(t string) bool {
	if t == "" || t == "---" {
		return false
	}
	if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "|") ||
		strings.HasPrefix(t, ">") || strings.HasPrefix(t, "- ") ||
		strings.HasPrefix(t, "**") {
		return false
	}
	return true
}

// stripInlineMarkdown removes link syntax, bold/italic markers, and inline code
// backticks, returning clean descriptor text.
func stripInlineMarkdown(s string) string {
	s = contentMdLinkRe.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "`", "")
	return strings.TrimSpace(s)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/content/ -run TestFirstFlavorParagraph"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add internal/content/flavor.go internal/content/flavor_test.go
git commit -m "feat(content): add firstFlavorParagraph extractor for structured flavor field"
```

---

## Task 2: Emit `flavor` from the simple passthrough parsers (ancestry, career, culture, title, complication, perk, class, kit)

**Files:**
- Modify: `steel-etl/internal/content/ancestry.go`, `career.go`, `culture.go`, `title.go`, `complication.go`, `perk.go`, `class.go`, `kit.go`
- Test: `steel-etl/internal/content/flavor_test.go`

- [ ] **Step 1: Write the failing test**

Append to `steel-etl/internal/content/flavor_test.go`:

```go
import_marker_for_parser_tests:

// (add these to the existing flavor_test.go file, alongside the imports)
```

Add the following test (and extend the import block to include `context`, `parser`):

```go
func parseSection(t *testing.T, p interface {
	Parse(*ctxstack, *parser.Section) (*ParsedContent, error)
}) {}
```

Instead of the placeholder above, write a concrete test that drives each parser through a minimal `*parser.Section`. Replace the file's import block with:

```go
import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)
```

and add:

```go
// newLeafSection builds a minimal annotated leaf section whose FullBodySource()
// returns body, for driving a parser in isolation.
func newLeafSection(heading, body string, ann map[string]string) *parser.Section {
	return &parser.Section{
		Heading:      heading,
		HeadingLevel: 4,
		Annotation:   ann,
		BodySource:   body,
	}
}

func TestParsers_EmitFlavor(t *testing.T) {
	body := "An ancient bloodline of stone-skinned giants.\n\n**Signature Trait:** Mighty\n"
	tests := []struct {
		name    string
		parser  interface {
			Parse(*context.ContextStack, *parser.Section) (*ParsedContent, error)
		}
		heading string
		want    string
	}{
		{"ancestry", &AncestryParser{}, "Hakaan", "An ancient bloodline of stone-skinned giants."},
		{"culture", &CultureParser{}, "Nomadic", "An ancient bloodline of stone-skinned giants."},
		{"title", &TitleParser{}, "Demonslayer", "An ancient bloodline of stone-skinned giants."},
		{"perk", &PerkParser{}, "Alert", "An ancient bloodline of stone-skinned giants."},
		{"class", &ClassParser{}, "Fury", "An ancient bloodline of stone-skinned giants."},
		{"kit", &KitParser{}, "Panther", "An ancient bloodline of stone-skinned giants."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec := newLeafSection(tt.heading, body, nil)
			got, err := tt.parser.Parse(context.NewContextStack(nil), sec)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got.Frontmatter["flavor"] != tt.want {
				t.Errorf("flavor = %v, want %q", got.Frontmatter["flavor"], tt.want)
			}
		})
	}
}

func TestCareerParser_FlavorStripsPrompt(t *testing.T) {
	body := "You worked as a spy for a powerful noble. In defining your career, think about the following questions:\n\n**Renown:** 1\n"
	sec := newLeafSection("Spy", body, nil)
	got, err := (&CareerParser{}).Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Frontmatter["flavor"] != "You worked as a spy for a powerful noble." {
		t.Errorf("flavor = %v, want stripped lead-in", got.Frontmatter["flavor"])
	}
}

func TestComplicationParser_FlavorSkipsBenefit(t *testing.T) {
	body := "A debt you can never seem to repay.\n\n**Benefit:** You know lenders.\n\n**Drawback:** They know you.\n"
	sec := newLeafSection("Crushing Debt", body, nil)
	got, err := (&ComplicationParser{}).Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Frontmatter["flavor"] != "A debt you can never seem to repay." {
		t.Errorf("flavor = %v, want flavor above benefit/drawback", got.Frontmatter["flavor"])
	}
}
```

> **Note on `parser.Section`:** This test assumes `parser.Section` has an exported `BodySource` field (or equivalent) that `FullBodySource()` returns. Before writing the test, open `steel-etl/internal/parser/section.go`, confirm the field name used to back `FullBodySource()`, and adjust `newLeafSection` to set it. If `FullBodySource()` is computed from child nodes rather than a settable field, instead build the section via the package's existing test constructor (grep `internal/content` test files — e.g. `treasure_test.go`, `kit_test.go` — for how they build `*parser.Section`, and reuse that exact pattern).

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/content/ -run 'TestParsers_EmitFlavor|TestCareerParser_FlavorStripsPrompt|TestComplicationParser_FlavorSkipsBenefit'"`
Expected: FAIL — `flavor` is nil for every parser.

- [ ] **Step 3: Add `fm["flavor"]` to each parser**

In `ancestry.go`, `culture.go`, `title.go`, `perk.go`, `class.go`, `kit.go`, immediately after `body := section.FullBodySource()` (for `kit.go`, after the existing `extractKitEquipmentText(body, fm)` call), add:

```go
	if f := firstFlavorParagraph(body); f != "" {
		fm["flavor"] = f
	}
```

In `career.go`, after `body := section.FullBodySource()`, add the prompt-stripping variant:

```go
	if f := firstFlavorParagraph(body); f != "" {
		if i := strings.Index(f, "In defining your career"); i >= 0 {
			f = strings.TrimSpace(f[:i])
		}
		if f != "" {
			fm["flavor"] = f
		}
	}
```

In `complication.go`, after `body := section.FullBodySource()`, add:

```go
	if f := firstFlavorParagraph(body); f != "" {
		fm["flavor"] = f
	}
```

(`isFlavorProse` already skips the `**Benefit:**` / `**Drawback:**` bold lines, so the first prose paragraph is the flavor above them.)

> `career.go` already imports `strings`; the others (`ancestry.go`, `class.go`, `culture.go`, `kit.go`) do too. `title.go`, `perk.go`, `complication.go`: confirm `strings` is imported only where the snippet uses it — the plain `firstFlavorParagraph` snippet needs no new import.

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/content/ -run 'TestParsers_EmitFlavor|TestCareerParser_FlavorStripsPrompt|TestComplicationParser_FlavorSkipsBenefit'"`
Expected: PASS.

- [ ] **Step 5: Run the full content package to check for regressions**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/content/..."`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add internal/content/ancestry.go internal/content/career.go internal/content/culture.go \
        internal/content/title.go internal/content/complication.go internal/content/perk.go \
        internal/content/class.go internal/content/kit.go internal/content/flavor_test.go
git commit -m "feat(content): parsers emit structured flavor field for index-card types"
```

---

## Task 3: Treasure — emit `flavor`, `project_goal`, `project_roll_characteristic`

**Files:**
- Modify: `steel-etl/internal/content/treasure.go`
- Test: `steel-etl/internal/content/treasure_test.go`

- [ ] **Step 1: Write the failing test**

Append to `steel-etl/internal/content/treasure_test.go` (match the existing test style in that file for building the `*parser.Section` — reuse its helper/constructor):

```go
func TestTreasureParser_FlavorAndProjectFields(t *testing.T) {
	body := "" +
		"*A bag that holds far more than its size suggests.*\n\n" +
		"**Keywords:** Magic\n\n" +
		"**Project Goal:** 45\n\n" +
		"**Project Roll Characteristic:** Reason\n"
	// Build the section the same way the other tests in this file do, with
	// heading "Bag of Holding" and the body above.
	parsed := parseTreasureForTest(t, "Bag of Holding", body) // see note below

	if parsed.Frontmatter["flavor"] != "A bag that holds far more than its size suggests." {
		t.Errorf("flavor = %v", parsed.Frontmatter["flavor"])
	}
	if parsed.Frontmatter["project_goal"] != "45" {
		t.Errorf("project_goal = %v, want 45", parsed.Frontmatter["project_goal"])
	}
	if parsed.Frontmatter["project_roll_characteristic"] != "Reason" {
		t.Errorf("project_roll_characteristic = %v, want Reason", parsed.Frontmatter["project_roll_characteristic"])
	}
}
```

> Replace `parseTreasureForTest` with whatever construction the existing `treasure_test.go` tests use (grep the file for `TreasureParser{}` and copy the section-building lines inline if there is no helper).

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/content/ -run TestTreasureParser_FlavorAndProjectFields"`
Expected: FAIL — `project_goal` / `project_roll_characteristic` / `flavor` are nil.

- [ ] **Step 3: Implement**

In `treasure.go`, inside `TreasureParser.Parse`, after the existing `effect` extraction block (after line ~68, `fm["effect"] = v`), add:

```go
	if v := extractField(body, "Project Goal"); v != "" {
		fm["project_goal"] = v
	}
	if v := extractField(body, "Project Roll Characteristic"); v != "" {
		fm["project_roll_characteristic"] = v
	}
	if f := firstFlavorParagraph(body); f != "" {
		fm["flavor"] = f
	}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/content/ -run TestTreasureParser_FlavorAndProjectFields"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add internal/content/treasure.go internal/content/treasure_test.go
git commit -m "feat(content): treasure parser emits flavor, project_goal, project_roll_characteristic"
```

---

## Task 4: Schema updates — add `flavor` (culture, perk) and `echelon` (treasure) in BOTH copies

**Files:**
- Modify: `steel-etl/schemas/culture.schema.json` + `data-sdk-npm/src/schema/culture.schema.json`
- Modify: `steel-etl/schemas/perk.schema.json` + `data-sdk-npm/src/schema/perk.schema.json`
- Modify: `steel-etl/schemas/treasure.schema.json` + `data-sdk-npm/src/schema/treasure.schema.json`

- [ ] **Step 1: Add `flavor` to both culture schema copies**

In each of `steel-etl/schemas/culture.schema.json` and `data-sdk-npm/src/schema/culture.schema.json`, add this property to the `properties` object (place it right after the `type` property so it reads naturally):

```json
    "flavor": {
      "type": "string",
      "description": "Narrative flavor text describing the culture (first prose paragraph)."
    },
```

- [ ] **Step 2: Add `flavor` to both perk schema copies**

In each of `steel-etl/schemas/perk.schema.json` and `data-sdk-npm/src/schema/perk.schema.json`, add to `properties`:

```json
    "flavor": {
      "type": "string",
      "description": "Narrative flavor / lead descriptor for the perk (first prose paragraph)."
    },
```

- [ ] **Step 3: Add `echelon` to both treasure schema copies**

In each of `steel-etl/schemas/treasure.schema.json` and `data-sdk-npm/src/schema/treasure.schema.json`, add to `properties`:

```json
    "echelon": {
      "type": "string",
      "description": "Echelon tier the treasure belongs to (\"1\"–\"4\"), when applicable."
    },
```

- [ ] **Step 4: Verify the two copies are byte-identical per type**

Run:

```bash
cd /home/vexa/code/steel_compendium/workspace
for s in culture perk treasure; do
  diff -q steel-etl/schemas/$s.schema.json data-sdk-npm/src/schema/$s.schema.json \
    && echo "$s: IN SYNC" || echo "$s: DRIFT — fix before commit"
done
```

Expected: `culture: IN SYNC`, `perk: IN SYNC`, `treasure: IN SYNC`.

- [ ] **Step 5: Validate the JSON is well-formed**

Run: `devbox run -- bash -c "for f in steel-etl/schemas/culture.schema.json steel-etl/schemas/perk.schema.json steel-etl/schemas/treasure.schema.json data-sdk-npm/src/schema/culture.schema.json data-sdk-npm/src/schema/perk.schema.json data-sdk-npm/src/schema/treasure.schema.json; do jq empty \$f && echo \"\$f ok\"; done"`
Expected: every file prints `ok`.

- [ ] **Step 6: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git add steel-etl/schemas/culture.schema.json steel-etl/schemas/perk.schema.json steel-etl/schemas/treasure.schema.json
git -C data-sdk-npm add src/schema/culture.schema.json src/schema/perk.schema.json src/schema/treasure.schema.json
git commit -m "feat(schema): add culture/perk flavor and treasure echelon (steel-etl copy)"
git -C data-sdk-npm commit -m "feat(schema): add culture/perk flavor and treasure echelon (sdk copy)"
```

> The two repos commit independently (separate git roots). Keep the messages parallel so the sync is auditable.

---

## Task 5: Update the schema-validation allowlist + add coverage

**Files:**
- Modify: `steel-etl/internal/output/schema_validation_test.go`

- [ ] **Step 1: Extend the allowlist (the test that fails on unexpected fields)**

In `schemaAllowedFields`, add the new keys:

- `culture`: add `"flavor": true,`
- `perk`: add `"flavor": true,`
- `treasure`: add `"echelon": true,` (note: `project_goal` and `project_roll_characteristic` are already present)

- [ ] **Step 2: Add the new fields to `TestSchema_NoUnevaluatedProperties` cases**

In `TestSchema_NoUnevaluatedProperties`, extend the `culture`, `perk`, and `treasure` test fixtures to include the new fields so the allowlist is exercised:

- culture fixture `fm`: add `"flavor": "A wandering people of the high steppes",`
- perk fixture `fm`: add `"flavor": "You always keep one eye on the door",`
- treasure fixture `fm`: add `"echelon": "3", "project_goal": "45", "project_roll_characteristic": "Reason",`

- [ ] **Step 3: Run the schema-validation tests to verify they pass**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/output/ -run TestSchema"`
Expected: PASS (the new fields are now allowed; absence would have produced `unexpected field` errors).

- [ ] **Step 4: Run the full output package (includes conformance against the real Heroes doc)**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/output/..."`
Expected: PASS. (Confirms the real-document transform still conforms now that parsers emit the new fields.)

- [ ] **Step 5: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add internal/output/schema_validation_test.go
git commit -m "test(output): allowlist culture/perk flavor and treasure echelon/project fields"
```

---

## Task 6: Refactor `cards.go` to read the frontmatter field (parser single source)

**Files:**
- Modify: `steel-etl/internal/site/cards.go`
- Test: `steel-etl/internal/site/cards_test.go`

- [ ] **Step 1: Write the failing test**

Append to `steel-etl/internal/site/cards_test.go`:

```go
func TestCardFlavor_PrefersFrontmatter(t *testing.T) {
	fm := "flavor: From the frontmatter field\n"
	body := "From the body prose paragraph.\n"
	if got := cardFlavor(fm, body); got != "From the frontmatter field" {
		t.Errorf("cardFlavor = %q, want frontmatter value", got)
	}
}

func TestCardFlavor_FallsBackToBody(t *testing.T) {
	fm := "name: Thing\n"
	body := "From the body prose paragraph.\n"
	if got := cardFlavor(fm, body); got != "From the body prose paragraph." {
		t.Errorf("cardFlavor = %q, want body fallback", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/site/ -run TestCardFlavor"`
Expected: FAIL — `undefined: cardFlavor`.

- [ ] **Step 3: Add the `cardFlavor` helper**

In `cards.go`, add near the other frontmatter/body helpers (e.g. just above `firstProse`):

```go
// cardFlavor returns the structured `flavor` frontmatter field (the parser is
// the single source of truth), falling back to the first body prose paragraph
// for pages produced before the field existed. Keeps card output stable while
// making the data field authoritative.
func cardFlavor(fm, body string) string {
	if f := strings.TrimSpace(parseFrontmatterField(fm, "flavor")); f != "" {
		return f
	}
	return firstProse(body)
}
```

- [ ] **Step 4: Route the cards through `cardFlavor` / the new treasure fields**

Make these edits in `cards.go`:

- **`ancestryCard`** — replace `if f := firstProse(body); f != "" {` with `if f := cardFlavor(fm, body); f != "" {`.
- **`titleCard`** — replace `if f := firstProse(body); f != "" {` with `if f := cardFlavor(fm, body); f != "" {`.
- **`cultureCard`** — replace `if f := firstProse(body); f != "" {` with `if f := cardFlavor(fm, body); f != "" {`.
- **`kitCard`** — replace `if desc := firstProse(body); desc != "" {` with `if desc := cardFlavor(fm, body); desc != "" {`.
- **`careerCard`** — replace `if f := careerFlavor(body); f != "" {` with:
  ```go
  	if f := strings.TrimSpace(parseFrontmatterField(fm, "flavor")); f != "" {
  		inner += flavorDiv(f, 200)
  	} else if f := careerFlavor(body); f != "" {
  		inner += flavorDiv(f, 200)
  	}
  ```
- **`complicationCard`** — replace the opening lines that compute `flavor`:
  ```go
  	flavor := complicationFlavor(body)
  	if flavor == "" {
  		flavor = stripMD(firstField(fm, "benefit", "drawback"))
  	}
  ```
  with:
  ```go
  	flavor := strings.TrimSpace(parseFrontmatterField(fm, "flavor"))
  	if flavor == "" {
  		flavor = complicationFlavor(body)
  	}
  	if flavor == "" {
  		flavor = stripMD(firstField(fm, "benefit", "drawback"))
  	}
  ```
- **`treasureCard`** — replace the `firstProse(body)` flavor line and the two `bodyLabeledLine` project lookups:
  - flavor: change `if f := firstProse(body); f != "" {` to `if f := cardFlavor(fm, body); f != "" {`.
  - Project Goal: change
    ```go
    	if v := bodyLabeledLine(body, "Project Goal"); v != "" {
    ```
    to
    ```go
    	if v := firstNonEmpty(parseFrontmatterField(fm, "project_goal"), bodyLabeledLine(body, "Project Goal")); v != "" {
    ```
  - Project Roll Characteristic: change
    ```go
    	if v := bodyLabeledLine(body, "Project Roll Characteristic"); v != "" {
    ```
    to
    ```go
    	if v := firstNonEmpty(parseFrontmatterField(fm, "project_roll_characteristic"), bodyLabeledLine(body, "Project Roll Characteristic")); v != "" {
    ```

Add the small helper used above (near `cardFlavor`):

```go
// firstNonEmpty returns the first trimmed-non-empty value among the args.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
```

> `classCard` is intentionally **not** changed — it renders the full `classIntro`, which is richer than the scalar `flavor` field (see Design decision 3).

- [ ] **Step 5: Run the site package tests**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/site/..."`
Expected: PASS — including the new `TestCardFlavor_*` and the existing card golden/snapshot tests (output is unchanged because the field carries the same stripped text the body scrape produced, and fallbacks preserve behavior where the field is absent).

- [ ] **Step 6: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add internal/site/cards.go internal/site/cards_test.go
git commit -m "refactor(site): cards read structured flavor + treasure project fields (parser single source)"
```

---

## Task 7: Full build, end-to-end regen sanity check

**Files:** none (verification only)

- [ ] **Step 1: Build + vet + full test suite with race detector**

Run: `devbox run -- bash -c "cd steel-etl && go build ./... && go vet ./... && go test -race ./..."`
Expected: PASS across all packages.

- [ ] **Step 2: Regenerate the data outputs and spot-check a flavor field landed in JSON**

Run:

```bash
cd /home/vexa/code/steel_compendium/workspace
devbox run -- bash -c "cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all"
```

Then spot-check that a treasure JSON now carries the new fields and a culture JSON carries `flavor`:

```bash
cd /home/vexa/code/steel_compendium/workspace
grep -rl '"project_goal"' data/ | head -1 | xargs -I{} sh -c 'echo {}; jq "{name, flavor, project_goal, project_roll_characteristic, echelon}" {}'
grep -rl '"type":"culture"' data/ 2>/dev/null | head -1 | xargs -I{} sh -c 'echo {}; jq "{name, flavor}" {}'
```

Expected: the treasure object shows populated `flavor` / `project_goal` / `project_roll_characteristic` (and `echelon` where applicable); the culture object shows a populated `flavor`. (`data/` is gitignored build output — nothing to commit here; this step only proves the field flows end-to-end.)

> If `jq` reports the JSON is an array/object wrapper, adjust the filter to the file's actual top-level shape — the point is to eyeball that the fields are present and non-empty.

---

## Task 8: Documentation — make future card fields land in the data formats

**Files:**
- Create: `steel-etl/docs/card-data-parity.md`
- Modify: `steel-etl/CLAUDE.md`, workspace `ARCHITECTURE.md`

- [ ] **Step 1: Write the standing rule + checklist**

Create `steel-etl/docs/card-data-parity.md`:

```markdown
# Card ⇄ Data Field Parity

**Rule:** When a parser is upgraded to surface a field on an index card (or any
other card / page), that field MUST also be promoted into the structured data
formats — unless it is purely presentational (truncation, icon choice, layout).

The index/preview cards (`internal/site/cards.go`, `feature_index.go`,
`ability_cards.go`, `trait_cards.go`) are *site-only* and may read either
frontmatter or the page body. The **page body is not a data contract** — only
frontmatter flows into the JSON/YAML outputs (via `transformPassthrough` /
`copyFrontmatter`) and is governed by the schemas. So a card that scrapes a
value out of the body is a parity gap: the website shows data the data repos
lack.

## Checklist for adding a card-surfaced field

1. **Extract once, in the parser.** Add `fm["<field>"] = …` in the relevant
   `internal/content/<type>.go` parser so the value lands in frontmatter and
   flows automatically into JSON/YAML. Share extraction helpers (e.g.
   `firstFlavorParagraph`) so the card and the data never diverge.
2. **Declare it in BOTH schema copies.** `steel-etl/schemas/<type>.schema.json`
   AND `data-sdk-npm/src/schema/<type>.schema.json`. They are hand-synced and
   use `unevaluatedProperties: false`, so an undeclared field is invalid.
   Verify with `diff -q` between the two copies.
3. **Update the allowlist test.** Add the key to `schemaAllowedFields` in
   `internal/output/schema_validation_test.go` and exercise it in
   `TestSchema_NoUnevaluatedProperties`.
4. **Make the card read the frontmatter field** (with a body fallback only as a
   safety net), so the parser is the single source of truth.
5. **The data-sdk-npm consumer side is its own effort.** Adding the field to the
   TS SDK (DTOs, model classes, markdown/json/yaml readers+writers, tests) is
   tracked separately — see `docs/superpowers/plans/2026-06-08-card-data-field-parity-sdk.md`.

## Precedent

The first sweep (2026-06-08) promoted `flavor` for every card type plus treasure
`project_goal` / `project_roll_characteristic` / `echelon`. See
`docs/superpowers/plans/2026-06-08-card-data-field-parity.md`.
```

- [ ] **Step 2: Cross-reference from steel-etl/CLAUDE.md**

In `steel-etl/CLAUDE.md`, in the "Content embedding patterns" section (or directly after the `cards.go` row in the Key files table), add a short note:

```markdown
## Card ⇄ data field parity

Index/preview cards are site-only and may read the page **body**, but the body
is **not** a data contract — only frontmatter flows into JSON/YAML + the schemas.
When you upgrade a parser to surface a new field on a card, promote it into the
data formats too: emit `fm["<field>"]` in the parser, declare it in BOTH schema
copies, update the `schema_validation_test.go` allowlist, and have the card read
the field. Full checklist: `docs/card-data-parity.md`.
```

- [ ] **Step 3: Note it in the workspace ARCHITECTURE.md**

In `ARCHITECTURE.md`, in (or adjacent to) the "Schemas: two hand-synced copies" section, add one line:

```markdown
- **Card ⇄ data parity:** index-card fields scraped from the page body must also
  be promoted into frontmatter + both schema copies, or the site shows data the
  data repos lack. See `steel-etl/docs/card-data-parity.md`.
```

- [ ] **Step 4: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git add steel-etl/docs/card-data-parity.md steel-etl/CLAUDE.md ARCHITECTURE.md
git commit -m "docs: card<->data field parity rule + checklist for future parser upgrades"
```

---

## Self-Review notes

- **Spec coverage:** flavor for all card types (Tasks 2–3), treasure project fields + echelon (Tasks 3–4), schema in both copies (Task 4), allowlist (Task 5), cards read field (Task 6), docs for future parity (Task 8), data-sdk-npm split out (separate plan). ✓
- **`parser.Section` construction** is the one unknown — every parser test (Task 2/3) depends on building a `*parser.Section` whose `FullBodySource()` returns a chosen body. Resolve it once by copying the construction from the existing `treasure_test.go` / `kit_test.go` before writing Task 2's test (called out inline).
- **No output regression:** cards keep their body-scrape as a fallback and the field stores the same stripped text, so existing card snapshot tests stay green.
- **`echelon` reverse-gap:** the parser already emitted `echelon`; Task 4 only catches the schema up — there is no data change for treasures, just contract correctness.
```
