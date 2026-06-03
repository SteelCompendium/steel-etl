# Read Tab By Book Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize the v2 site's **Read** tab so each source book is its own folder with its own index, chapters are ordered by their position in the source book (not alphabetically), and the Beastheart book shows its full content (class, companions, rewards, perks) as proper chapters instead of only the fiction intro.

**Architecture:** Four coordinated changes. (1) **Content:** re-annotate `steel-etl/input/beastheart/Draw Steel Beastheart.md` so its rules nest under four chapters (Fiction, The Beastheart Class, Rewards, Perks) — fixing the flat-hierarchy bug that currently hides them from Read and makes the Browse class page swallow treasures/perks. (2) **Pipeline:** persist a per-book `order` integer into chapter frontmatter (document order). (3) **Site builder:** group the Read section into per-book subfolders, generate per-book indexes + a Read landing index, and emit explicit source-ordered `.nav.yml` nav lists. (4) **Config:** declare books (key→folder/label/order) in `v2/site.yaml`.

**Tech Stack:** Go 1.26 (steel-etl, `internal/pipeline` + `internal/site`), annotated markdown (Beastheart source), MkDocs Material + `awesome-nav` plugin (v2), `just` recipes, devbox toolchain.

---

## ⚠️ Read this first: environment + ground rules

- **Devbox is mandatory.** Go/just/node are NOT on PATH. Prefix every Go/just command with `devbox run --`, e.g. `devbox run -- go test ./...`. Run from the workspace root `/home/vexa/code/steel_compendium/workspace` (which contains `devbox.json`). For steel-etl Go commands, `cd steel-etl` after activating, or use `devbox run -- bash -c 'cd steel-etl && go test ./...'`.
- **Never hand-edit generated dirs:** `data/data-rules/`, `data/data-beastheart/`, `data/data-unified/`, `data/data-rules-clean/`, `v2/docs/Browse/`, `v2/docs/Read/`, `v2/docs/scc/`. They are wiped/regenerated. All content changes go in `steel-etl/input/...`; all behavior changes go in Go source or `v2/site.yaml` / `v2/static_content/`.
- **SCC stability:** SCC codes are annotation-driven (`@type` + `@id`), NOT heading-level-driven. The Beastheart re-annotation below preserves every existing `@id`, so no SCC code changes. After regen, verify with `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate --scc-stable --config pipeline.yaml'` (adjust path to the beastheart pipeline config — see Task 1.1).
- **The user must run deploys.** `just deploy-v2` / `just deploy-api` are the user's to run. This plan ends by asking the user to deploy; do not run deploys yourself.

## Design decisions (review these — they are judgment calls)

These were chosen to satisfy the four requirements with minimal risk. Flag during plan review if any should change:

1. **Beastheart "The Beastheart Class" becomes a chapter that *contains* the class.** The Read tab includes only `chapter/`-typed pages. To show the class in Read while keeping the Browse `class/beastheart` page, the class is wrapped in a chapter:
   - Chapter heading (Read): **"The Beastheart Class"** — `@type: chapter`, `@id: the-beastheart-class`
   - Nested class heading (Browse): **"Beastheart"** — `@type: class`, `@id: beastheart` (unchanged id ⇒ unchanged SCC `mcdm.beastheart.v1/class/beastheart`)
   - **Consequence:** the Browse class page's display name (frontmatter `name`/H1) changes from "The Beastheart Class" to **"Beastheart"**, consistent with the other class names (Censor, Conduit, …). The SCC code and URL are unchanged.
2. **Book identity = `scc:` prefix** (substring before the first `/`). Generalizes to all future books without per-file tagging.
3. **Per-book folder layout** under Read: `Read/<folder>/<chapter>.md` + `Read/<folder>/index.md`, plus a generated `Read/index.md` landing page listing books. Folder slug + display label come from `v2/site.yaml`.
4. **Chapter order = document order**, persisted as an `order:` integer in chapter frontmatter by the pipeline, then emitted as an explicit `nav:` list in each book's `.nav.yml`. Books themselves are ordered by `order` in `site.yaml`.
5. **Beastheart internal nesting** follows the book's table of contents where heading depth permits (max H6). The class subtree is demoted by exactly one level (deepest existing H5 → H6, which fits). Rewards/Perks get explicit per-heading level normalization.

## File structure

| File | Responsibility | Change |
|------|----------------|--------|
| `steel-etl/input/beastheart/Draw Steel Beastheart.md` | Beastheart source of truth | Re-annotate: 4 chapters, nest class, promote Rewards/Perks, normalize heading levels |
| `steel-etl/internal/pipeline/pipeline.go` | Section walk + frontmatter assembly | Inject per-book `order:` into chapter frontmatter |
| `steel-etl/internal/pipeline/pipeline_test.go` (or new `pipeline_order_test.go`) | Pipeline tests | Test chapter order assignment |
| `steel-etl/internal/site/config.go` | Site config types | Add `Books []BookConfig`; add `GroupByBook` to `SectionConfig`; resolve book lookup |
| `steel-etl/internal/site/config_test.go` | Config tests | Test book parsing + lookup |
| `steel-etl/internal/site/build.go` | Site builder | Per-book dest mapping, per-book `.nav.yml` (ordered) + indexes, Read landing index |
| `steel-etl/internal/site/build_test.go` | Builder tests | Test per-book grouping, ordering, indexes |
| `v2/site.yaml` | Site builder config | Add `books:` list; set `group_by_book: true` on Read section |
| `steel-etl/CLAUDE.md` / `ARCHITECTURE.md` | Docs | Note per-book Read layout + chapter ordering |

---

## Phase 1 — Beastheart source re-annotation (content)

**Why first:** It is independent of the Go changes and immediately fixes the Browse class page swallowing treasures/perks. After this phase, the Beastheart book will contribute 4 chapters (still flat in Read until Phase 3).

**Target top-level structure (4 chapters):**

```
# The Beastheart & The Faeries        [H1, @type: chapter, @id: the-beastheart-and-the-faeries]   (fiction — UNCHANGED)
# The Beastheart Class                 [H1, @type: chapter, @id: the-beastheart-class]              (NEW wrapper)
  ## Beastheart                        [H2, @type: class,   @id: beastheart]                        (was H1 "The Beastheart Class")
    ### Basics ... ### 10th-Level Features  (entire class body, demoted +1 level)
# Rewards                              [H1, @type: chapter, @id: rewards]                            (was "## Rewards")
  ## Trinkets / ## Leveled Treasures   (+ normalized echelon/category subheadings)
# Perks                                [H1, @type: chapter, @id: perks]                              (was "## Perks")
  ## Exploration / Intrigue / Interpersonal Perks
```

> **Line numbers below are the *current* source positions** (from the pre-edit file). Because edits shift line numbers, each task re-derives positions with `grep -n` right before editing rather than trusting stale numbers. Always re-grep.

### Task 1.0: Find the Beastheart pipeline config

The pipeline must be run per book. Confirm how the Beastheart book is generated before changing its source.

- [ ] **Step 1: Locate the beastheart pipeline config and gen command**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
ls steel-etl/*.yaml steel-etl/**/*.yaml 2>/dev/null | grep -i beast || true
grep -rn "data-beastheart\|beastheart" steel-etl/*.yaml steel-etl/justfile justfile 2>/dev/null | head
cat justfile | sed -n '1,80p'
```
Expected: identify the config (e.g. `steel-etl/pipeline-beastheart.yaml` or a `--book` flag) and the exact `gen` invocation the `just deploy` recipes use for the beastheart book. Record it; later tasks call it as `<BEASTHEART_GEN_CMD>`.

- [ ] **Step 2: Capture a baseline of the current Browse class page (to prove the bug + the fix)**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
grep -c -E '^#### (Precious Collar|Cavalry Armor|Born Tracker)|^## (Rewards|Perks)' data/data-beastheart/en/md-linked/class/beastheart.md
```
Expected: a non-zero count (currently the class page wrongly contains Rewards/treasures/Perks). After Phase 1 regen this must become `0`.

### Task 1.1: Add the "The Beastheart Class" chapter wrapper + nest the class

**Files:**
- Modify: `steel-etl/input/beastheart/Draw Steel Beastheart.md`

- [ ] **Step 1: Re-grep current anchors**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
grep -nE '^# |@type: class \| @id: beastheart|@type: chapter' "steel-etl/input/beastheart/Draw Steel Beastheart.md"
grep -nE '^## (Basics|Rewards|Perks)' "steel-etl/input/beastheart/Draw Steel Beastheart.md"
```
Expected (current): `# The Beastheart & The Faeries` (~L8), class annotation + `# The Beastheart Class` (~L266-267), `## Basics` (~L274), `## Rewards` (~L2560), `## Perks` (~L2852).

- [ ] **Step 2: Convert the class H1 into a chapter wrapper + nested class heading**

The current block is:
```markdown
<!-- @type: class | @id: beastheart -->
# The Beastheart Class

A beastheart never fights alone! You travel with a ferocious beast by your side...
```
Replace it (via Edit) with:
```markdown
<!-- @type: chapter | @id: the-beastheart-class -->
# The Beastheart Class

<!-- @type: class | @id: beastheart -->
## Beastheart

A beastheart never fights alone! You travel with a ferocious beast by your side...
```
(Keep the full existing intro prose — only the heading line above it is added; match enough of the first prose sentence in `old_string` to anchor uniquely.)

- [ ] **Step 3: Demote the entire class body by one heading level**

Everything from `## Basics` (the first class feature) up to **but not including** `## Rewards` must gain one `#`. Re-derive the exact line range first:
```bash
cd /home/vexa/code/steel_compendium/workspace
F="steel-etl/input/beastheart/Draw Steel Beastheart.md"
START=$(grep -nE '^## Basics' "$F" | head -1 | cut -d: -f1)
END=$(( $(grep -nE '^## Rewards' "$F" | head -1 | cut -d: -f1) - 1 ))
echo "Demoting headings in lines $START..$END"
# Add one leading '#' to every ATX heading line in the range (no fenced code blocks exist in this file — verified).
perl -i -pe "if (\$. >= $START && \$. <= $END && /^#{1,6} /) { s/^#/##/ }" "$F"
```
Expected: `## Basics`→`### Basics`, the `## Nth-Level Features` group headers→`###`, the per-feature `##`→`###`, companion `### Basilisk`→`####`, abilities `#### Petrify`→`#####`, advancement features `##### Foes Forever Frozen`→`######` (H6, fits).

- [ ] **Step 4: Verify the class subtree is now bounded and no heading exceeds H6**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
F="steel-etl/input/beastheart/Draw Steel Beastheart.md"
echo "--- H1s (must be exactly 4 after Tasks 1.1-1.3; right now: fiction, class-chapter, + still ## Rewards/## Perks) ---"
grep -nE '^# ' "$F"
echo "--- any H7+ (must be empty) ---"
grep -nE '^#{7,} ' "$F" || echo "OK: no heading deeper than H6"
echo "--- class heading present at H2 ---"
grep -nE '^## Beastheart$' "$F"
```
Expected: no `#######`; `## The Beastheart` present; the only `# ` lines are the fiction chapter and the new "The Beastheart Class" chapter (Rewards/Perks promoted in 1.2/1.3).

### Task 1.2: Promote "Rewards" to a chapter and normalize treasure subheadings

**Files:**
- Modify: `steel-etl/input/beastheart/Draw Steel Beastheart.md`

Target structure (per the book TOC):
```
# Rewards                          [H1, @type: chapter, @id: rewards]
  ## Trinkets                      [H2]
    ### 1st-Echelon Trinkets       [H3]
      #### Precious Collar         [H4, @type: treasure]   (already H4)
    ### 2nd/3rd/4th-Echelon Trinket[H3]
  ## Leveled Treasures             [H2]
    ### Leveled Armor Treasures    [H3]
      #### Cavalry Armor           [H4, @type: treasure]
    ### Leveled Weapon Treasures   [H3]
```

- [ ] **Step 1: Re-grep the Rewards block**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
F="steel-etl/input/beastheart/Draw Steel Beastheart.md"
awk 'NR>=2 && /^## Rewards/{p=1} p && /^## Perks/{p=0} p' "$F" | grep -nE '^#{2,4} ' | head -60
grep -nE '^## (Rewards|Trinkets|Leveled Treasures|Leveled Armor Treasures|Leveled Weapon Treasures|[0-9].*Echelon Trinket)' "$F"
```
Record current line numbers for each heading.

- [ ] **Step 2: Make Rewards a chapter (H1)**

Edit the `## Rewards` line. The current line is:
```markdown
## Rewards
```
Replace with:
```markdown
<!-- @type: chapter | @id: rewards -->
# Rewards
```

- [ ] **Step 3: Demote the echelon/category sub-group headings from H2 to H3**

These five headings are currently `##` and must become `###` so they nest under Trinkets / Leveled Treasures (which stay `##`). Edit each line individually:
- `## 1st-Echelon Trinkets` → `### 1st-Echelon Trinkets`
- `## 2nd-Echelon Trinket` → `### 2nd-Echelon Trinket`
- `## 3rd-Echelon Trinket` → `### 3rd-Echelon Trinket`
- `## 4th-Echelon Trinket` → `### 4th-Echelon Trinket`
- `## Leveled Armor Treasures` → `### Leveled Armor Treasures`
- `## Leveled Weapon Treasures` → `### Leveled Weapon Treasures`

Leave `## Trinkets` and `## Leveled Treasures` as H2. Leave each `#### <treasure name>` as H4 (treasure entries already H4).

- [ ] **Step 4: Verify Rewards nesting**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
F="steel-etl/input/beastheart/Draw Steel Beastheart.md"
awk '/^# Rewards/{p=1} p && /^# Perks/{p=0} p' "$F" | grep -E '^#{1,6} '
```
Expected order/levels: `# Rewards`, `## Trinkets`, `### 1st-Echelon Trinkets`, `#### Precious Collar`, ..., `## Leveled Treasures`, `### Leveled Armor Treasures`, `#### Cavalry Armor`, `### Leveled Weapon Treasures`, ...

### Task 1.3: Promote "Perks" to a chapter and normalize perk subheadings

**Files:**
- Modify: `steel-etl/input/beastheart/Draw Steel Beastheart.md`

Target:
```
# Perks                        [H1, @type: chapter, @id: perks]
  ## Exploration Perks         [H2]
    ### Born Tracker           [H3, @type: perk]
    ### Ride Along             [H3, @type: perk]
      #### Ride Along          [H4 ability statblock — already present, now nests fine]
  ## Intrigue Perks            [H2]
  ## Interpersonal Perks       [H2]
```

- [ ] **Step 1: Re-grep the Perks block**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
F="steel-etl/input/beastheart/Draw Steel Beastheart.md"
awk '/^## Perks/{p=1} p' "$F" | grep -nE '^#{2,4} |@type: perk'
```

- [ ] **Step 2: Make Perks a chapter (H1)**

Current:
```markdown
## Perks
```
Replace with:
```markdown
<!-- @type: chapter | @id: perks -->
# Perks
```

- [ ] **Step 3: Demote each perk *entry* heading from H2 to H3**

The category headers `## Exploration Perks`, `## Intrigue Perks`, `## Interpersonal Perks` stay H2. Each individual perk title (the line immediately after a `<!-- @type: perk ... -->` annotation) is currently `## <Name>` and becomes `### <Name>`. The eight perk titles:
`Born Tracker`, `Ride Along`, `Wild Rumpus`, `Wilds Explorer`, `Trained Thief`, `People Sense`, `Voice of the Wild`, `You Can Pet Them, They're Friendly`.

For each, change `## <Name>` → `### <Name>`. (The inner `#### <Name>` ability statblocks under Ride Along / Wild Rumpus stay H4 and now nest correctly under their H3 perk.)

- [ ] **Step 4: Verify Perks nesting + final H1 count**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
F="steel-etl/input/beastheart/Draw Steel Beastheart.md"
echo "--- final H1 chapters (expect exactly 4) ---"
grep -nE '^# ' "$F"
echo "--- perks block ---"
awk '/^# Perks/{p=1} p' "$F" | grep -E '^#{1,6} '
echo "--- no H7+ ---"
grep -nE '^#{7,} ' "$F" || echo OK
```
Expected: exactly 4 `# ` headings: `The Beastheart & The Faeries`, `The Beastheart Class`, `Rewards`, `Perks`. Perk entries at `###`. No H7+.

### Task 1.4: Regenerate and verify the Beastheart book

- [ ] **Step 1: Run the beastheart pipeline**

Run (substitute the command found in Task 1.0):
```bash
cd /home/vexa/code/steel_compendium/workspace
devbox run -- bash -c 'cd steel-etl && <BEASTHEART_GEN_CMD>'
```
Expected: success, no duplicate-SCC errors.

- [ ] **Step 2: Verify SCC stability (no codes changed)**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate --scc-stable --config <BEASTHEART_PIPELINE_CONFIG>'
```
Expected: PASS / no changed codes. (The class id `beastheart`, all treasure ids, all perk ids, all feature/ability ids are unchanged.)

- [ ] **Step 3: Verify the Browse class page no longer swallows treasures/perks**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
grep -c -E '^#### (Precious Collar|Cavalry Armor|Born Tracker)|^## (Rewards|Perks)' data/data-beastheart/en/md-linked/class/beastheart.md
```
Expected: `0` (was non-zero in Task 1.0 Step 2).

- [ ] **Step 4: Verify the four chapters now exist with full bodies**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
ls -la data/data-beastheart/en/md-linked/chapter/
for f in the-beastheart-and-the-faeries the-beastheart-class rewards perks; do
  echo "--- $f ---"; head -6 "data/data-beastheart/en/md-linked/chapter/$f.md"; wc -l "data/data-beastheart/en/md-linked/chapter/$f.md"
done
echo "--- the-beastheart-class chapter should contain Basics + a companion + level features ---"
grep -cE 'Basics|Wild Nature|Companion|10th-Level Features' data/data-beastheart/en/md-linked/chapter/the-beastheart-class.md
echo "--- rewards chapter should contain a treasure; perks chapter a perk ---"
grep -c 'Precious Collar' data/data-beastheart/en/md-linked/chapter/rewards.md
grep -c 'Born Tracker' data/data-beastheart/en/md-linked/chapter/perks.md
```
Expected: four chapter files exist; `the-beastheart-class.md` is large and contains the class content (no longer ends at "Continued in *Between Sun & Shadow*."); `rewards.md` contains treasures; `perks.md` contains perks.

- [ ] **Step 5: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git -C steel-etl add "input/beastheart/Draw Steel Beastheart.md"
git -C steel-etl commit -m "content: restructure Beastheart into 4 chapters (class wrapped, rewards/perks promoted)

Fixes flat-hierarchy bug where Rewards/Perks were H2 trapped under the class H1,
causing the Browse class page to swallow treasures+perks and hiding the rules from
the Read tab. Class is now nested under a 'The Beastheart Class' chapter; Rewards
and Perks are top-level chapters. All @id values preserved (no SCC changes)."
```

---

## Phase 2 — Persist chapter source order in the pipeline (Go)

**Why:** The site builder walks the filesystem (alphabetical) and loses document order. Persist a per-book `order:` integer onto each chapter so the builder can sort by it.

### Task 2.1: Assign `order` to chapter frontmatter in document order

**Files:**
- Modify: `steel-etl/internal/pipeline/pipeline.go` (the `walk` closure in `RunWithConfig`, around lines 107-173)
- Test: `steel-etl/internal/pipeline/pipeline_order_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/pipeline/pipeline_order_test.go`. Mirror the existing tests' setup style in `pipeline_test.go` (read it first to reuse helpers/fixtures and the correct `RunWithConfig`/`Run` signature). The test feeds a small annotated doc with three chapters interleaved with non-chapter content and asserts chapters receive `order` 0,1,2 in document order:

```go
package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChapterOrderIsDocumentOrder(t *testing.T) {
	src := strings.Join([]string{
		"---", "book: mcdm.test.v1", "---", "",
		"<!-- @type: chapter | @id: alpha -->", "# Alpha", "", "Intro alpha.", "",
		"<!-- @type: chapter | @id: bravo -->", "# Bravo", "", "Intro bravo.", "",
		"<!-- @type: chapter | @id: charlie -->", "# Charlie", "", "Intro charlie.", "",
	}, "\n")

	dir := t.TempDir()
	in := filepath.Join(dir, "in.md")
	if err := os.WriteFile(in, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	reg := filepath.Join(dir, "registry.json")

	// Use the same entry point as pipeline_test.go (adjust if the helper differs).
	if _, err := Run(in, out, reg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// md output carries frontmatter; assert order values.
	for name, want := range map[string]string{"alpha": "order: 0", "bravo": "order: 1", "charlie": "order: 2"} {
		// Path shape: <out>/.../chapter/<name>.md — find it.
		var found string
		filepath.Walk(out, func(p string, _ os.FileInfo, _ error) error {
			if strings.HasSuffix(p, "chapter/"+name+".md") {
				found = p
			}
			return nil
		})
		if found == "" {
			t.Fatalf("chapter %s.md not generated", name)
		}
		data, _ := os.ReadFile(found)
		if !strings.Contains(string(data), want) {
			t.Errorf("%s: expected %q in frontmatter, got:\n%s", name, want, string(data))
		}
	}
}
```

> If `Run`'s signature or output path shape differs from the assumption, adjust to match `pipeline_test.go`. Read that file before running.

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/pipeline/ -run TestChapterOrderIsDocumentOrder -v'`
Expected: FAIL — `order: N` not found (field not yet emitted).

- [ ] **Step 3: Implement order assignment**

In `steel-etl/internal/pipeline/pipeline.go`, declare a counter before the `walk` closure and inject it for chapter-typed sections. Add immediately after `result.ParsedSections++` (line ~130), before the `parsed.PageBody = ...` line:

```go
		// Chapters get a per-book document-order index so the site builder can
		// present them in book order rather than alphabetically.
		if typeName == "chapter" {
			parsed.Frontmatter["order"] = chapterOrder
			chapterOrder++
		}
```

And declare the counter just before `var walk func(...)` (line ~107):

```go
	chapterOrder := 0
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/pipeline/ -run TestChapterOrderIsDocumentOrder -v'`
Expected: PASS.

- [ ] **Step 5: Run the full pipeline package tests; fix golden/conformance fallout**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/pipeline/... ./internal/output/...'`
Expected: PASS. If any golden/conformance test now fails because chapter frontmatter gained an `order:` field, update the expected fixtures to include `order:` (this is the intended new behavior, not a regression). Note which fixtures changed in the commit message.

- [ ] **Step 6: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git -C steel-etl add internal/pipeline/ internal/output/
git -C steel-etl commit -m "feat(pipeline): persist per-book chapter document order in frontmatter"
```

---

## Phase 3 — Per-book Read separation in the site builder (Go)

**Why:** Group Read by book, order chapters by `order`, and generate per-book + landing indexes.

### Task 3.1: Add book config + section flag

**Files:**
- Modify: `steel-etl/internal/site/config.go`
- Test: `steel-etl/internal/site/config_test.go`

- [ ] **Step 1: Write the failing test**

Append to `steel-etl/internal/site/config_test.go`:

```go
func TestLoadBooksAndGroupByBook(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "site.yaml")
	yaml := `
source_dirs: [./src]
docs_dir: ./docs
books:
  - key: mcdm.heroes.v1
    folder: heroes
    label: Draw Steel Heroes
    order: 1
  - key: mcdm.beastheart.v1
    folder: beastheart
    label: "Draw Steel: Beastheart"
    order: 2
sections:
  - name: Read
    include: [chapter/]
    group_by_book: true
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadSiteConfig(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Books) != 2 {
		t.Fatalf("expected 2 books, got %d", len(cfg.Books))
	}
	b, ok := cfg.BookByKey("mcdm.beastheart.v1")
	if !ok || b.Folder != "beastheart" || b.Label != "Draw Steel: Beastheart" || b.Order != 2 {
		t.Errorf("BookByKey beastheart wrong: %+v ok=%v", b, ok)
	}
	if !cfg.Sections[0].GroupByBook {
		t.Errorf("expected GroupByBook=true on Read section")
	}
}
```
(Ensure `os`, `path/filepath`, `testing` are imported.)

- [ ] **Step 2: Run to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestLoadBooksAndGroupByBook -v'`
Expected: FAIL — `cfg.Books` / `BookByKey` / `GroupByBook` undefined.

- [ ] **Step 3: Implement config types**

In `steel-etl/internal/site/config.go`, add to `Config`:
```go
	// Books maps a book's SCC prefix to a display folder/label/order for
	// per-book section grouping (used by sections with GroupByBook=true).
	Books []BookConfig `yaml:"books,omitempty"`
```
Add to `SectionConfig`:
```go
	// GroupByBook places each page into a per-book subfolder (derived from the
	// page's scc prefix via Config.Books) instead of its SCC type path, and
	// emits source-ordered nav + per-book index pages.
	GroupByBook bool `yaml:"group_by_book,omitempty"`
```
Add the type + lookup after `GroupConfig`:
```go
// BookConfig maps a book's SCC prefix (substring before the first '/') to a
// display folder slug, human label, and sort order for the Read tab.
type BookConfig struct {
	Key    string `yaml:"key"`
	Folder string `yaml:"folder"`
	Label  string `yaml:"label"`
	Order  int    `yaml:"order"`
}

// BookByKey returns the BookConfig whose Key matches, and whether it was found.
func (c *Config) BookByKey(key string) (BookConfig, bool) {
	for _, b := range c.Books {
		if b.Key == key {
			return b, true
		}
	}
	return BookConfig{}, false
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestLoadBooksAndGroupByBook -v'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git -C steel-etl add internal/site/config.go internal/site/config_test.go
git -C steel-etl commit -m "feat(site): add per-book config (Books, group_by_book)"
```

### Task 3.2: Add frontmatter helpers for book key + order

**Files:**
- Modify: `steel-etl/internal/site/build.go`
- Test: `steel-etl/internal/site/build_test.go`

- [ ] **Step 1: Write the failing test**

Append to `steel-etl/internal/site/build_test.go`:
```go
func TestBookKeyFromSCC(t *testing.T) {
	cases := map[string]string{
		"mcdm.heroes.v1/chapter/introduction": "mcdm.heroes.v1",
		"mcdm.beastheart.v1/chapter/rewards":  "mcdm.beastheart.v1",
		"":                                    "",
		"noslash":                             "noslash",
	}
	for in, want := range cases {
		if got := bookKeyFromSCC(in); got != want {
			t.Errorf("bookKeyFromSCC(%q)=%q want %q", in, got, want)
		}
	}
}

func TestParseFrontmatterOrder(t *testing.T) {
	fm := "name: Rewards\nscc: mcdm.beastheart.v1/chapter/rewards\ntype: chapter\norder: 3\n"
	if got := parseFrontmatterInt(fm, "order", -1); got != 3 {
		t.Errorf("order=%d want 3", got)
	}
	if got := parseFrontmatterInt("name: x\n", "order", 99); got != 99 {
		t.Errorf("missing order default=%d want 99", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run "TestBookKeyFromSCC|TestParseFrontmatterOrder" -v'`
Expected: FAIL — undefined `bookKeyFromSCC` / `parseFrontmatterInt`.

- [ ] **Step 3: Implement helpers**

Add to `steel-etl/internal/site/build.go` (near `parseFrontmatterField`, ~line 552). Add `strconv` to imports:
```go
// bookKeyFromSCC returns the book prefix of an SCC code (substring before the
// first '/'); returns the input unchanged when there is no '/'.
func bookKeyFromSCC(scc string) string {
	if i := strings.Index(scc, "/"); i >= 0 {
		return scc[:i]
	}
	return scc
}

// parseFrontmatterInt extracts an integer scalar from frontmatter, or def if
// absent/unparseable.
func parseFrontmatterInt(fm, key string, def int) int {
	v := parseFrontmatterField(fm, key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run "TestBookKeyFromSCC|TestParseFrontmatterOrder" -v'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git -C steel-etl add internal/site/build.go internal/site/build_test.go
git -C steel-etl commit -m "feat(site): add bookKeyFromSCC + parseFrontmatterInt helpers"
```

### Task 3.3: Route GroupByBook sections into per-book folders

**Files:**
- Modify: `steel-etl/internal/site/build.go` (`buildSection`, ~lines 134-179; `Build`, ~lines 51-65)
- Test: `steel-etl/internal/site/build_test.go`

This replaces the SCC-type destination (`chapter/x.md`) with a per-book destination (`<folder>/x.md`) for GroupByBook sections, and skips the default `.nav.yml` writer for them (Task 3.4 writes ordered nav instead).

- [ ] **Step 1: Write the failing test**

Append to `build_test.go` a test that lays down two fake md-linked source dirs (one per book), each with a `chapter/` file carrying `scc` + `order` frontmatter, runs `Build`, and asserts files land under `Read/<folder>/` in the right place:

```go
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildGroupsReadByBook(t *testing.T) {
	dir := t.TempDir()
	heroesSrc := filepath.Join(dir, "src-heroes")
	beastSrc := filepath.Join(dir, "src-beast")
	docs := filepath.Join(dir, "docs")

	writeFile(t, filepath.Join(heroesSrc, "chapter", "introduction.md"),
		"---\nname: Introduction\nscc: mcdm.heroes.v1/chapter/introduction\ntype: chapter\norder: 0\n---\n\nHero intro.\n")
	writeFile(t, filepath.Join(heroesSrc, "chapter", "classes.md"),
		"---\nname: Classes\nscc: mcdm.heroes.v1/chapter/classes\ntype: chapter\norder: 7\n---\n\nClasses.\n")
	writeFile(t, filepath.Join(beastSrc, "chapter", "rewards.md"),
		"---\nname: Rewards\nscc: mcdm.beastheart.v1/chapter/rewards\ntype: chapter\norder: 2\n---\n\nRewards.\n")

	cfg := &Config{
		SourceDirs: []string{heroesSrc, beastSrc},
		DocsDir:    docs,
		Books: []BookConfig{
			{Key: "mcdm.heroes.v1", Folder: "heroes", Label: "Draw Steel Heroes", Order: 1},
			{Key: "mcdm.beastheart.v1", Folder: "beastheart", Label: "Draw Steel: Beastheart", Order: 2},
		},
		Sections: []SectionConfig{{Name: "Read", Include: []string{"chapter/"}, GroupByBook: true}},
	}

	if _, err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, p := range []string{
		filepath.Join(docs, "Read", "heroes", "introduction.md"),
		filepath.Join(docs, "Read", "heroes", "classes.md"),
		filepath.Join(docs, "Read", "beastheart", "rewards.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
	// Must NOT keep the SCC type folder.
	if _, err := os.Stat(filepath.Join(docs, "Read", "chapter")); err == nil {
		t.Errorf("Read/chapter should not exist under group_by_book")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildGroupsReadByBook -v'`
Expected: FAIL — files land under `Read/chapter/`, not `Read/heroes/`.

- [ ] **Step 3: Implement per-book destination in `buildSection`**

In `buildSection` (build.go), after reading `data` (the file bytes) and BEFORE `rewriteSectionLinks`, compute the destination. Replace the block that computes `destRel`/`destPath` so that, for GroupByBook sections, the destination is `<folder>/<basename>`:

```go
	for _, entry := range entries {
		if !matchesSection(entry.relPath, section) {
			continue
		}

		data, err := os.ReadFile(entry.absPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("read %s: %v", entry.absPath, err))
			continue
		}

		var destRel, parentName string
		if section.GroupByBook {
			fm, _ := splitFrontmatter(string(data))
			key := bookKeyFromSCC(parseFrontmatterField(fm, "scc"))
			book, ok := cfg.BookByKey(key)
			if !ok {
				errs = append(errs, fmt.Sprintf("no book config for scc prefix %q (%s)", key, entry.relPath))
				continue
			}
			destRel = filepath.ToSlash(filepath.Join(book.Folder, filepath.Base(entry.relPath)))
		} else {
			destRel, parentName = applyGroups(entry.relPath, section.Groups, entry.sourceDir)
		}
		destPath := filepath.Join(sectionDir, destRel)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			errs = append(errs, fmt.Sprintf("mkdir %s: %v", destPath, err))
			continue
		}

		data = []byte(rewriteSectionLinks(string(data), entry.relPath, destRel, section.Name, cfg.Sections))

		if parentName != "" {
			data = combineFrontmatterName(data, parentName)
		}
		data = injectH1(data)

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", destPath, err))
			continue
		}
		count++
	}
```
(This restructures the existing loop body — the `os.ReadFile` moves up before destination computation. Keep the rest identical.)

- [ ] **Step 4: Skip the default `.nav.yml` writer for GroupByBook sections**

In `Build` (build.go ~line 59-65), guard the per-section nav writer so GroupByBook sections are handled by Task 3.4 instead:
```go
	for _, section := range cfg.Sections {
		if section.GroupByBook {
			continue // ordered per-book nav is written by writeBookNav
		}
		if err := writeNavYaml(cfg.DocsDir, section); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("nav %s: %v", section.Name, err))
		} else {
			result.NavFiles++
		}
	}
```

- [ ] **Step 5: Run to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildGroupsReadByBook -v'`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git -C steel-etl add internal/site/build.go internal/site/build_test.go
git -C steel-etl commit -m "feat(site): route group_by_book sections into per-book folders"
```

### Task 3.4: Emit source-ordered per-book nav + book/landing indexes

**Files:**
- Modify: `steel-etl/internal/site/build.go` (call from `Build` after `buildSection` loop; new `writeBookNavAndIndexes`)
- Test: `steel-etl/internal/site/build_test.go`

- [ ] **Step 1: Write the failing test**

Append to `build_test.go` (reuses `TestBuildGroupsReadByBook`'s setup; factor a helper or duplicate the three `writeFile` calls + cfg). Assert nav ordering + indexes:

```go
func TestBuildBookNavAndIndexes(t *testing.T) {
	dir := t.TempDir()
	heroesSrc := filepath.Join(dir, "src-heroes")
	beastSrc := filepath.Join(dir, "src-beast")
	docs := filepath.Join(dir, "docs")
	writeFile(t, filepath.Join(heroesSrc, "chapter", "introduction.md"),
		"---\nname: Introduction\nscc: mcdm.heroes.v1/chapter/introduction\ntype: chapter\norder: 0\n---\n\nHero intro.\n")
	writeFile(t, filepath.Join(heroesSrc, "chapter", "classes.md"),
		"---\nname: Classes\nscc: mcdm.heroes.v1/chapter/classes\ntype: chapter\norder: 7\n---\n\nClasses.\n")
	writeFile(t, filepath.Join(beastSrc, "chapter", "rewards.md"),
		"---\nname: Rewards\nscc: mcdm.beastheart.v1/chapter/rewards\ntype: chapter\norder: 2\n---\n\nRewards.\n")
	writeFile(t, filepath.Join(beastSrc, "chapter", "the-beastheart-and-the-faeries.md"),
		"---\nname: The Beastheart & The Faeries\nscc: mcdm.beastheart.v1/chapter/the-beastheart-and-the-faeries\ntype: chapter\norder: 0\n---\n\nFiction.\n")
	cfg := &Config{
		SourceDirs: []string{heroesSrc, beastSrc},
		DocsDir:    docs,
		Books: []BookConfig{
			{Key: "mcdm.heroes.v1", Folder: "heroes", Label: "Draw Steel Heroes", Order: 1},
			{Key: "mcdm.beastheart.v1", Folder: "beastheart", Label: "Draw Steel: Beastheart", Order: 2},
		},
		Sections: []SectionConfig{{Name: "Read", Title: "Rulebook Chapters", Include: []string{"chapter/"}, GroupByBook: true}},
	}
	if _, err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}

	// Per-book nav: ordered by `order`, with book label as title.
	heroesNav, _ := os.ReadFile(filepath.Join(docs, "Read", "heroes", ".nav.yml"))
	if !strings.Contains(string(heroesNav), "Draw Steel Heroes") {
		t.Errorf("heroes nav missing label:\n%s", heroesNav)
	}
	// introduction (order 0) must appear before classes (order 7).
	if i, c := strings.Index(string(heroesNav), "introduction.md"), strings.Index(string(heroesNav), "classes.md"); i < 0 || c < 0 || i > c {
		t.Errorf("heroes nav not in source order:\n%s", heroesNav)
	}
	// beastheart: fiction (order 0) before rewards (order 2).
	beastNav, _ := os.ReadFile(filepath.Join(docs, "Read", "beastheart", ".nav.yml"))
	if f, r := strings.Index(string(beastNav), "the-beastheart-and-the-faeries.md"), strings.Index(string(beastNav), "rewards.md"); f < 0 || r < 0 || f > r {
		t.Errorf("beastheart nav not in source order:\n%s", beastNav)
	}

	// Top-level Read nav orders books by Book.Order.
	readNav, _ := os.ReadFile(filepath.Join(docs, "Read", ".nav.yml"))
	if h, b := strings.Index(string(readNav), "heroes"), strings.Index(string(readNav), "beastheart"); h < 0 || b < 0 || h > b {
		t.Errorf("Read nav not in book order:\n%s", readNav)
	}

	// Per-book index + landing index exist.
	for _, p := range []string{
		filepath.Join(docs, "Read", "heroes", "index.md"),
		filepath.Join(docs, "Read", "beastheart", "index.md"),
		filepath.Join(docs, "Read", "index.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s: %v", p, err)
		}
	}
	// Book index lists chapters by friendly name in source order.
	beastIdx, _ := os.ReadFile(filepath.Join(docs, "Read", "beastheart", "index.md"))
	if f, r := strings.Index(string(beastIdx), "The Beastheart & The Faeries"), strings.Index(string(beastIdx), "Rewards"); f < 0 || r < 0 || f > r {
		t.Errorf("beastheart index not in source order:\n%s", beastIdx)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildBookNavAndIndexes -v'`
Expected: FAIL — `.nav.yml`/index files not produced for per-book layout.

- [ ] **Step 3: Implement nav + index generation**

In `build.go`, call a new function from `Build` after the `buildSection` loop and before the generic `generateIndexPages` call (so the generic index generator can be told to skip GroupByBook sections — see Step 4). Add to `Build`:
```go
	// Per-book ordered nav + indexes for GroupByBook sections.
	for _, section := range cfg.Sections {
		if !section.GroupByBook {
			continue
		}
		n, errs := writeBookNavAndIndexes(cfg, section)
		result.NavFiles += n
		result.Errors = append(result.Errors, errs...)
	}
```

Add the implementation:
```go
// chapterRef is a chapter file with its display name and source order.
type chapterRef struct {
	file  string // basename, e.g. "rewards.md"
	name  string // frontmatter name, e.g. "Rewards"
	order int
}

// writeBookNavAndIndexes emits, for a GroupByBook section: one ordered .nav.yml
// + index.md per book folder, and a top-level section .nav.yml + index.md that
// lists the books in Book.Order.
func writeBookNavAndIndexes(cfg *Config, section SectionConfig) (int, []string) {
	sectionDir := filepath.Join(cfg.DocsDir, section.Name)
	var errs []string
	navCount := 0

	// Books that actually produced a folder, in Book.Order.
	books := append([]BookConfig(nil), cfg.Books...)
	sort.SliceStable(books, func(i, j int) bool { return books[i].Order < books[j].Order })

	var present []BookConfig
	for _, b := range books {
		bookDir := filepath.Join(sectionDir, b.Folder)
		if _, err := os.Stat(bookDir); err != nil {
			continue // no chapters for this book
		}
		present = append(present, b)

		// Collect chapter files (skip index.md) with name + order.
		var chapters []chapterRef
		dirEntries, _ := os.ReadDir(bookDir)
		for _, e := range dirEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "index.md" {
				continue
			}
			fm, _ := splitFrontmatter(readFile(filepath.Join(bookDir, e.Name())))
			name := parseFrontmatterField(fm, "name")
			if name == "" {
				name = fileToTitle(e.Name())
			}
			chapters = append(chapters, chapterRef{
				file:  e.Name(),
				name:  name,
				order: parseFrontmatterInt(fm, "order", 1<<30),
			})
		}
		sort.SliceStable(chapters, func(i, j int) bool {
			if chapters[i].order != chapters[j].order {
				return chapters[i].order < chapters[j].order
			}
			return naturalLess(chapters[i].file, chapters[j].file)
		})

		// Per-book .nav.yml: explicit ordered list (index first).
		var nb strings.Builder
		nb.WriteString("title: " + yamlScalar(b.Label) + "\n")
		nb.WriteString("nav:\n")
		nb.WriteString("  - index.md\n")
		for _, c := range chapters {
			nb.WriteString("  - " + c.file + "\n")
		}
		if err := os.WriteFile(filepath.Join(bookDir, ".nav.yml"), []byte(nb.String()), 0644); err != nil {
			errs = append(errs, fmt.Sprintf("book nav %s: %v", b.Folder, err))
		} else {
			navCount++
		}

		// Per-book index.md (ordered list of chapters).
		var ib strings.Builder
		ib.WriteString("# " + b.Label + "\n\n---\n\n<div class=\"browse-index\" markdown>\n\n")
		for _, c := range chapters {
			ib.WriteString("- [" + c.name + "](" + c.file + ")\n")
		}
		ib.WriteString("\n</div>\n")
		if err := os.WriteFile(filepath.Join(bookDir, "index.md"), []byte(ib.String()), 0644); err != nil {
			errs = append(errs, fmt.Sprintf("book index %s: %v", b.Folder, err))
		}
	}

	// Section-level .nav.yml: title + ordered book folders (index first).
	title := section.Title
	if title == "" {
		title = section.Name
	}
	var sb strings.Builder
	sb.WriteString("title: " + yamlScalar(title) + "\n")
	sb.WriteString("nav:\n")
	sb.WriteString("  - index.md\n")
	for _, b := range present {
		sb.WriteString("  - " + b.Folder + "\n")
	}
	if err := os.WriteFile(filepath.Join(sectionDir, ".nav.yml"), []byte(sb.String()), 0644); err != nil {
		errs = append(errs, fmt.Sprintf("section nav %s: %v", section.Name, err))
	} else {
		navCount++
	}

	// Section landing index.md: lists the books.
	var lb strings.Builder
	lb.WriteString("---\nsearch:\n  exclude: true\n---\n\n# " + title + "\n\n---\n\n<div class=\"browse-index\" markdown>\n\n")
	for _, b := range present {
		lb.WriteString("- [" + b.Label + "](" + b.Folder + "/)\n")
	}
	lb.WriteString("\n</div>\n")
	if err := os.WriteFile(filepath.Join(sectionDir, "index.md"), []byte(lb.String()), 0644); err != nil {
		errs = append(errs, fmt.Sprintf("section index %s: %v", section.Name, err))
	}

	return navCount, errs
}

// readFile reads a file, returning "" on error (used for best-effort frontmatter reads).
func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// yamlScalar quotes a YAML scalar if it contains characters that need quoting.
func yamlScalar(s string) string {
	if strings.ContainsAny(s, ":#\"'{}[],&*?|<>=!%@`") {
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	}
	return s
}
```

- [ ] **Step 4: Tell the generic index generator to skip GroupByBook sections**

The generic `generateIndexPages` would otherwise overwrite per-book indexes with alphabetical ones. In `Build`, change the call so GroupByBook sections are excluded, and in `generateIndexPages` accept the filtered list. Simplest: filter before calling:
```go
	// Generate index pages for type directories (skip GroupByBook sections —
	// those get ordered indexes from writeBookNavAndIndexes).
	var genericSections []SectionConfig
	for _, s := range cfg.Sections {
		if !s.GroupByBook {
			genericSections = append(genericSections, s)
		}
	}
	indexCount, indexErrs := generateIndexPages(cfg.DocsDir, genericSections)
```
Ensure `writeBookNavAndIndexes` runs AFTER `buildSection` but the `generateIndexPages` call uses `genericSections`. Order in `Build`: buildSection loop → per-book nav/index loop (Step 3) → generic index pages (filtered) → search exclusion → static content → SCC stubs.

- [ ] **Step 5: Run to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildBookNavAndIndexes -v'`
Expected: PASS.

- [ ] **Step 6: Run the whole site package + vet**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/... && go vet ./internal/site/...'`
Expected: PASS. Fix any pre-existing test that asserted the old flat `Read/chapter/` layout (update it to the per-book layout — intended change).

- [ ] **Step 7: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git -C steel-etl add internal/site/
git -C steel-etl commit -m "feat(site): source-ordered per-book Read nav + book/landing indexes"
```

---

## Phase 4 — Wire up v2 config + end-to-end verification

### Task 4.1: Configure books + group_by_book in site.yaml

**Files:**
- Modify: `v2/site.yaml`

- [ ] **Step 1: Add the books list and enable grouping on Read**

Edit `v2/site.yaml`. Add a top-level `books:` block (e.g. after `docs_dir:`):
```yaml
# Books for per-book Read grouping. `key` matches the SCC prefix (substring
# before the first '/'); `folder` is the URL slug; `label` is the display title;
# `order` sets book ordering in the Read tab.
books:
  - key: mcdm.heroes.v1
    folder: heroes
    label: Draw Steel Heroes
    order: 1
  - key: mcdm.beastheart.v1
    folder: beastheart
    label: "Draw Steel: Beastheart"
    order: 2
```
And set `group_by_book: true` on the Read section (replacing/augmenting its `sort: natural`):
```yaml
  - name: Read
    title: Rulebook Chapters
    include:
      - chapter/
    group_by_book: true
```
(Leave `search_exclude: [Read]` as-is.)

- [ ] **Step 2: Verify the config loads**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml'`
Expected: builds without "no book config for scc prefix" errors. If a prefix error appears, a chapter has an scc prefix not listed in `books:` — add it.

- [ ] **Step 3: Inspect the generated Read tree**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
find v2/docs/Read -maxdepth 2 -type f | sort
echo "--- Read landing ---"; cat v2/docs/Read/index.md
echo "--- heroes nav ---"; cat v2/docs/Read/heroes/.nav.yml
echo "--- beastheart nav ---"; cat v2/docs/Read/beastheart/.nav.yml
echo "--- beastheart index ---"; cat v2/docs/Read/beastheart/index.md
```
Expected: `Read/index.md`, `Read/.nav.yml`, `Read/heroes/{.nav.yml,index.md,*.md}`, `Read/beastheart/{.nav.yml,index.md,the-beastheart-and-the-faeries.md,the-beastheart-class.md,rewards.md,perks.md}`. Heroes chapters in book order (Introduction → … → For the Director); beastheart in source order (Fiction → The Beastheart Class → Rewards → Perks).

- [ ] **Step 4: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git -C v2 add site.yaml
git -C v2 commit -m "feat(site): group Read tab by book with source ordering"
```

### Task 4.2: Build the MkDocs site and visually verify

- [ ] **Step 1: Run the full v2 build**

Run: `devbox run -- bash -c 'cd v2 && mkdocs build'`
Expected: builds clean. Watch for awesome-nav warnings about `.nav.yml` (e.g. files listed in `nav:` that don't exist, or vice versa). Resolve any.

- [ ] **Step 2: Spot-check rendered HTML**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
ls v2/site/Read/ v2/site/Read/beastheart/ 2>/dev/null
grep -l "Continued in" v2/site/Read/beastheart/*/index.html 2>/dev/null || true
echo "--- the-beastheart-class page should be large (full class) ---"
wc -c v2/site/Read/beastheart/the-beastheart-class/index.html
```
Expected: beastheart Read folder has 4 chapters; `the-beastheart-class` page is large (full class content), no longer ending at the fiction cliff-hanger; the fiction page still legitimately ends with "Continued in *Between Sun & Shadow*." (that is the real end of the fiction only).

- [ ] **Step 3: Confirm Browse class page is clean**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
grep -c -iE 'Precious Collar|Born Tracker|Cavalry Armor' v2/site/Browse/class/beastheart/index.html 2>/dev/null || \
  grep -rc -iE 'Precious Collar|Born Tracker' v2/docs/Browse/class/ 2>/dev/null | head
```
Expected: `0` — the Browse class page no longer contains treasures/perks.

### Task 4.3: Update docs

**Files:**
- Modify: `steel-etl/CLAUDE.md` (site builder section), `ARCHITECTURE.md` (Read section mapping)

- [ ] **Step 1: Document the per-book Read layout + chapter ordering**

In `steel-etl/CLAUDE.md` under "Site builder", add a bullet:
> - **Per-book Read grouping**: when a section sets `group_by_book: true`, pages are placed under `Read/<book-folder>/` (folder/label/order from the `books:` list in `site.yaml`, keyed by SCC prefix). Each book gets a source-ordered `.nav.yml` + `index.md`, and the Read tab gets a landing `index.md` listing books. Chapter order comes from the `order:` frontmatter field the pipeline assigns in document order.

In `ARCHITECTURE.md`, update the "section mapping (Browse, Read)" line to note Read is grouped by book and ordered by source position.

- [ ] **Step 2: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace
git -C steel-etl add CLAUDE.md
git add ARCHITECTURE.md
git -C steel-etl commit -m "docs: per-book Read grouping + chapter ordering"
git commit -m "docs: per-book Read grouping in ARCHITECTURE"
```

### Task 4.4: Full regression + hand off deploy

- [ ] **Step 1: Run all steel-etl tests with race**

Run: `devbox run -- bash -c 'cd steel-etl && go test -race ./...'`
Expected: PASS.

- [ ] **Step 2: Summarize for the user and request deploy**

The user runs deploys. Report: what changed, the new Read layout, the Browse class-page fix, and ask them to run `just deploy-v2` (and `just deploy-api` if SCC API output is desired) to publish. Note SCC codes are unchanged (verified in Task 1.4 Step 2).

---

## Self-review

- **Spec coverage:**
  - Bullet 1 (separate Read by book) → Phase 3 (per-book folders) + Task 4.1 (config). ✓
  - Bullet 2 (only one beastheart chapter) → Phase 1 (4 chapters now generated). ✓
  - Bullet 3 (chapter ends prematurely) → Phase 1 (class content now in its own chapter; fiction's "Continued in…" is correctly only the fiction's end). ✓
  - Bullet 4 (sort by source order) → Phase 2 (`order` frontmatter) + Task 3.4 (ordered nav/index) + Task 4.1 (book order). ✓
- **Type/name consistency:** `BookConfig{Key,Folder,Label,Order}`, `Config.Books`, `Config.BookByKey`, `SectionConfig.GroupByBook`, helpers `bookKeyFromSCC` / `parseFrontmatterInt` / `writeBookNavAndIndexes` / `chapterRef` / `readFile` / `yamlScalar` are used consistently across Tasks 3.1–3.4 and 4.1. Frontmatter key `order` is written by Phase 2 and read by Task 3.4. ✓
- **Known approximations to confirm during execution (not placeholders):**
  - Task 1.0 resolves the exact Beastheart `gen` command + pipeline config path (referenced later as `<BEASTHEART_GEN_CMD>` / `<BEASTHEART_PIPELINE_CONFIG>`).
  - Task 2.1 Step 1 verifies the pipeline entry point signature against `pipeline_test.go` before relying on `Run(in, out, reg)`.
  - Task 3.3 Step 3 restructures an existing loop body — diff against the current `buildSection` to keep unrelated behavior identical.
- **Risk notes:** heading bulk-demotion (Task 1.1 Step 3) relies on there being no fenced code blocks in the Beastheart source (verified: `grep` found none). The class-name change (design decision #1) is the one user-facing surprise — called out for review.
