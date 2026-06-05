# Treasure Hierarchy Reorganization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure the "Treasures" output so individual treasures nest under their category as `treasure/<tier>/<category>/<item>` (e.g. `treasure/1st-echelon/consumable/black-ash-dart`), instead of the current broken flat structure where echelon *categories* are mis-tagged as treasure *items* and the real items are swallowed into category body text.

**Architecture:** Treasures use the same container→item context pattern that abilities already use (`feature-group` → `ability`). A new `treasure-group` container annotation on each category heading pushes `echelon` + `treasure-type` into the context stack; the `TreasureParser` reads them to build a nested `TypePath`. Category/echelon hierarchy in the Browse tree and SCC codes is driven entirely by that `TypePath` (directory structure + auto-generated index pages). Prose/rules sections currently mis-tagged as treasures are folded back into the chapter body. Both the heroes and beastheart books are updated.

**Tech Stack:** Go 1.26 (steel-etl ETL + site builder), annotated Markdown source, MkDocs Material (v2 site), Python (index transform). All Go/just/python commands run under **devbox** (`devbox run -- <cmd>`).

---

## Background / Context for the Engineer

You have no prior context on this repo. Read these first:

- `steel-etl/ANNOTATION-GUIDE.md` — how `<!-- @type: ... -->` annotations work. Annotations sit immediately before a heading and classify that heading's section. The pipeline pushes each annotated section's annotation map into a **context stack** keyed by heading level (`internal/pipeline/pipeline.go:126-128`), *before* checking whether a parser is registered — so a container type with no parser still contributes context. Child parsers read ancestor context via `ctx.Lookup(headingLevel, key)` (walks up to shallower levels).
- `ARCHITECTURE.md` (workspace root) — the pipeline: `steel-etl gen` (source MD → `data/data-rules/...` multi-format) then `steel-etl site` (→ `v2/docs/`). `gen` must be run with `--all` to process all books.
- The SCC code for an item is `Classify(source, TypePath, ItemID)` = `source/<TypePath joined by ".">/<ItemID>` (`internal/scc/classifier.go`). The data-rules output **directory** mirrors `TypePath` (e.g. `feature/ability/boren/bear-claws.md` for TypePath `[feature, ability, boren]`). Browse index pages are auto-generated per directory recursively (`internal/site/build.go:860 generateIndexesRecursive`).

### The bug (heroes book)

In `steel-etl/input/heroes/Draw Steel Heroes.md`, the Treasures chapter (`<!-- @type: chapter | @id: treasures -->`, ~line 23736) looks like:

```
#### 1st-Echelon Consumables      <-- annotated @type: treasure (WRONG: it's a category)
  ##### Black Ash Dart            <-- NO annotation → swallowed into the category's body
  ##### Blood Essence Vial        <-- NO annotation
  ...
```

So each `#### N-Echelon Consumables/Trinkets` and each `#### Leveled X Treasures` became a single `treasure/<slug>.md` whose body is a wall of items, and no individual treasures exist as entities.

### The beastheart book (already partly correct)

`steel-etl/input/beastheart/Draw Steel Beastheart.md` Rewards chapter (`<!-- @type: chapter | @id: rewards -->`, ~line 2537) already makes each item its own `@type: treasure` with an `@echelon: N` annotation, but the category headers are unannotated, so items currently produce **flat** `treasure/<id>` codes with no hierarchy.

### Target structure (decided with the user)

- **Hierarchy / SCC**: `treasure/<tier>/<category>/<item>` where `tier` ∈ {`1st-echelon`,`2nd-echelon`,`3rd-echelon`,`4th-echelon`,`leveled`} and `category` ∈ {`consumable`,`trinket`,`armor`,`implement`,`weapon`,`other`}. **Echelon comes first.** Examples:
  - `mcdm.heroes.v1/treasure.1st-echelon.consumable/black-ash-dart`
  - `mcdm.heroes.v1/treasure.1st-echelon.trinket/deadweight`
  - `mcdm.heroes.v1/treasure.leveled.weapon/displacer`
  - `mcdm.beastheart.v1/treasure.1st-echelon.trinket/precious-collar`
- **Prose sections** (What Does This Treasure Do?, Wearing Treasures, Wielding Treasures, Magic and Psionic Treasures, Stamina Bonuses and Damage Bonuses, Leveled Benefits, Carry Three Safely): **fold into the chapter body** — remove their `@type: treasure` annotations. The 5 inbound `scc:` links that point at now-removed/changed codes are repointed to the chapter.
- **Scope**: both heroes and beastheart books. `data-gen/` is a *separate legacy ETL* ("It's a mess") and is **out of scope** — do not modify it. All steel-etl outputs (data-rules, data-unified, json/yaml/dse) come from the single parser change.
- **SCC freeze is off** (`pipeline.yaml: classification.freeze: false`), so changing treasure codes is allowed.

### File map

| File | Responsibility | Change |
|------|----------------|--------|
| `steel-etl/internal/content/treasure.go` | `TreasureParser` + new `TreasureGroupParser` | Build nested `TypePath`; add container parser |
| `steel-etl/internal/content/treasure_test.go` | Parser tests | Create |
| `steel-etl/internal/content/registry.go` | Parser registry | Register `TreasureGroupParser` |
| `steel-etl/internal/site/cards.go` | Browse stat-cards | Detect treasure leaf dirs |
| `steel-etl/internal/site/cards_test.go` | Card tests | Create |
| `steel-etl/input/heroes/Draw Steel Heroes.md` | Source (heroes) | Re-annotate Treasures chapter + repoint links |
| `steel-etl/input/beastheart/Draw Steel Beastheart.md` | Source (beastheart) | Annotate category headers |
| `steel-etl/ANNOTATION-GUIDE.md` | Docs | Document `treasure-group` |
| `workspace CLAUDE.md` | Docs | Update SCC registry count note |

---

## Task 1: TreasureParser builds nested TypePath + TreasureGroupParser container

**Files:**
- Modify: `steel-etl/internal/content/treasure.go`
- Modify: `steel-etl/internal/content/registry.go:29` (add registration after `TreasureParser`)
- Test: `steel-etl/internal/content/treasure_test.go` (create)

The `TreasureParser` currently always emits `TypePath: []string{"treasure"}` (`treasure.go:89`). We change it to compute `tier` (echelon slug, or `leveled`) and `category` (`treasure-type`, already resolved into `fm["treasure_type"]` from annotation or context at `treasure.go:26-41`), and append them. We also add a no-op `TreasureGroupParser` (modeled on `FeatureGroupParser` in `feature.go:8-38`) so category headers are explicit, validate cleanly, and push context.

- [ ] **Step 1: Write the failing tests**

Create `steel-etl/internal/content/treasure_test.go`:

```go
package content

import (
	"reflect"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestTreasureParser_NestedTypePath_Echelon(t *testing.T) {
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)
	// Parent treasure-group at H4 supplies echelon + category.
	ctx.Push(4, context.Metadata{"type": "treasure-group", "echelon": "1", "treasure-type": "consumable"})

	section := &parser.Section{
		Heading:      "Black Ash Dart",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource:   "**Keywords:** Magic\n\nAs a maneuver, you make a ranged free strike.",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	want := []string{"treasure", "1st-echelon", "consumable"}
	if !reflect.DeepEqual(result.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", result.TypePath, want)
	}
	if result.ItemID != "black-ash-dart" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "black-ash-dart")
	}
	if result.Frontmatter["echelon"] != "1" {
		t.Errorf("echelon = %v, want 1", result.Frontmatter["echelon"])
	}
	if result.Frontmatter["treasure_type"] != "consumable" {
		t.Errorf("treasure_type = %v, want consumable", result.Frontmatter["treasure_type"])
	}
}

func TestTreasureParser_NestedTypePath_Leveled(t *testing.T) {
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)
	// Leveled treasures have no echelon → tier "leveled".
	ctx.Push(4, context.Metadata{"type": "treasure-group", "treasure-type": "weapon"})

	section := &parser.Section{
		Heading:      "Displacer",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource:   "A leveled weapon treasure.",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	want := []string{"treasure", "leveled", "weapon"}
	if !reflect.DeepEqual(result.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", result.TypePath, want)
	}
	if _, ok := result.Frontmatter["echelon"]; ok {
		t.Errorf("echelon should be unset for leveled treasures, got %v", result.Frontmatter["echelon"])
	}
}

func TestTreasureParser_ItemAnnotationOverridesContext(t *testing.T) {
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)
	ctx.Push(3, context.Metadata{"type": "treasure-group", "echelon": "2", "treasure-type": "trinket"})

	// Beastheart-style: item carries its own @echelon.
	section := &parser.Section{
		Heading:      "Werewolf Tooth Pendant",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure", "echelon": "2"},
		BodySource:   "A trinket.",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []string{"treasure", "2nd-echelon", "trinket"}
	if !reflect.DeepEqual(result.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", result.TypePath, want)
	}
}

func TestTreasureGroupParser_NoOutput(t *testing.T) {
	p := &TreasureGroupParser{}
	if p.Type() != "treasure-group" {
		t.Errorf("Type() = %q, want treasure-group", p.Type())
	}
	section := &parser.Section{
		Heading:      "1st-Echelon Consumables",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure-group", "echelon": "1", "treasure-type": "consumable"},
		BodySource:   "These are the most numerous treasures.",
	}
	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.TypePath != nil {
		t.Errorf("TypePath = %v, want nil (container emits no file)", result.TypePath)
	}
	if result.ItemID != "" {
		t.Errorf("ItemID = %q, want empty", result.ItemID)
	}
	if result.Frontmatter["treasure_type"] != "consumable" {
		t.Errorf("treasure_type = %v, want consumable", result.Frontmatter["treasure_type"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/scott/code/steelCompendium/workspace/steel-etl && devbox run -- go test ./internal/content/ -run 'TestTreasure' -v`
Expected: FAIL — `TestTreasureGroupParser_NoOutput` won't compile (`TreasureGroupParser` undefined); the nested-path tests fail (current TypePath is `[treasure]`).

- [ ] **Step 3: Implement the parser changes**

In `steel-etl/internal/content/treasure.go`, replace the final `return` block (currently lines 86-92):

```go
	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"treasure"},
		ItemID:      id,
	}, nil
}
```

with:

```go
	// Resolve echelon (item annotation → ancestor context) and record it.
	echelon := ""
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["echelon"]; ok {
			echelon = v
		}
	}
	if echelon == "" {
		if v, ok := ctx.Lookup(section.HeadingLevel, "echelon"); ok {
			echelon = v
		}
	}
	if echelon != "" {
		fm["echelon"] = echelon
	}

	// Category (consumable/trinket/armor/implement/weapon/other) was resolved
	// into fm["treasure_type"] above from annotation or ancestor context.
	category, _ := fm["treasure_type"].(string)

	// Nested type path: treasure/<tier>/<category>. tier is the echelon slug
	// (1st-echelon…4th-echelon) or "leveled" when the treasure has no echelon.
	typePath := []string{"treasure"}
	tier := echelonSlug(echelon)
	if tier == "" {
		tier = "leveled"
	}
	typePath = append(typePath, tier)
	if category != "" {
		typePath = append(typePath, category)
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}

// echelonSlug converts an echelon number ("1".."4") into its tier slug
// ("1st-echelon".."4th-echelon"). Any other value returns "".
func echelonSlug(echelon string) string {
	switch strings.TrimSpace(echelon) {
	case "1":
		return "1st-echelon"
	case "2":
		return "2nd-echelon"
	case "3":
		return "3rd-echelon"
	case "4":
		return "4th-echelon"
	default:
		return ""
	}
}

// TreasureGroupParser handles @type: treasure-group sections — structural
// category containers (e.g. "1st-Echelon Consumables", "Leveled Weapon
// Treasures") that provide echelon + treasure-type context to child treasures.
// Like FeatureGroupParser, it produces no standalone output file; the pipeline
// pushes its annotation into the context stack regardless.
type TreasureGroupParser struct{}

func (p *TreasureGroupParser) Type() string { return "treasure-group" }

func (p *TreasureGroupParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	fm := map[string]any{
		"name": section.Heading,
		"type": "treasure-group",
	}
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["echelon"]; ok {
			fm["echelon"] = v
		}
		if v, ok := ann["treasure-type"]; ok {
			fm["treasure_type"] = v
		}
	}
	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
	}, nil
}
```

(`strings` is already imported in `treasure.go`.)

- [ ] **Step 4: Register the container parser**

In `steel-etl/internal/content/registry.go`, after the `TreasureParser` registration (line 29 `r.Register(&TreasureParser{})`), add:

```go
	r.Register(&TreasureGroupParser{})
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/scott/code/steelCompendium/workspace/steel-etl && devbox run -- go test ./internal/content/ -run 'TestTreasure' -v`
Expected: PASS (all four tests).

- [ ] **Step 6: Run the full content package tests (no regressions)**

Run: `cd /home/scott/code/steelCompendium/workspace/steel-etl && devbox run -- go test ./internal/content/...`
Expected: PASS (`ok`).

- [ ] **Step 7: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add internal/content/treasure.go internal/content/treasure_test.go internal/content/registry.go
git commit -m "feat(treasure): nested type path + treasure-group container"
```

---

## Task 2: Browse stat-cards render treasure leaf directories

**Files:**
- Modify: `steel-etl/internal/site/cards.go:60-87` (`buildCardsContent`)
- Test: `steel-etl/internal/site/cards_test.go` (create)

`buildCardsContent` (`cards.go:60`) only renders rich treasure cards when the directory is literally named `treasure` and has no subdirs (`richCardTypes["treasure"]` + `len(subdirs)==0`). After nesting, the **leaf** dirs are `consumable`/`trinket`/`armor`/etc. (not `treasure`), so the nice cards would be lost. Fix: detect a leaf directory that lives under a `treasure/` ancestor and render it with the `treasure` card builder. Intermediate dirs (which have subdirs) keep falling back to the default expandable list.

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/site/cards_test.go`:

```go
package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCardsContent_TreasureLeaf(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, "treasure", "1st-echelon", "consumable")
	if err := os.MkdirAll(leaf, 0755); err != nil {
		t.Fatal(err)
	}
	item := "---\nname: Black Ash Dart\ntype: treasure\ntreasure_type: consumable\nechelon: \"1\"\nkeywords:\n  - Magic\n---\n\nAs a maneuver, you make a ranged free strike using a black ash dart.\n"
	if err := os.WriteFile(filepath.Join(leaf, "black-ash-dart.md"), []byte(item), 0644); err != nil {
		t.Fatal(err)
	}

	content, ok := buildCardsContent(leaf, "consumable", []string{"black-ash-dart.md"}, nil)
	if !ok {
		t.Fatalf("buildCardsContent ok=false, want true for treasure leaf dir")
	}
	if !strings.Contains(content, "sc-cards") {
		t.Errorf("expected sc-cards wrapper, got:\n%s", content)
	}
	if !strings.Contains(content, "Black Ash Dart") {
		t.Errorf("expected item name in cards, got:\n%s", content)
	}
	// Leaf title comes from the dirName.
	if !strings.Contains(content, "# Consumable") {
		t.Errorf("expected '# Consumable' title, got:\n%s", content)
	}
}

func TestBuildCardsContent_TreasureIntermediateFallsBack(t *testing.T) {
	root := t.TempDir()
	mid := filepath.Join(root, "treasure", "1st-echelon")
	if err := os.MkdirAll(filepath.Join(mid, "consumable"), 0755); err != nil {
		t.Fatal(err)
	}
	// Intermediate dir has a subdir → not a leaf → no cards.
	_, ok := buildCardsContent(mid, "1st-echelon", nil, []string{"consumable"})
	if ok {
		t.Errorf("buildCardsContent ok=true for intermediate dir, want false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/scott/code/steelCompendium/workspace/steel-etl && devbox run -- go test ./internal/site/ -run 'TestBuildCardsContent_Treasure' -v`
Expected: FAIL — `TestBuildCardsContent_TreasureLeaf` gets `ok=false` (leaf dirName `consumable` not in `richCardTypes`).

- [ ] **Step 3: Implement leaf detection**

In `steel-etl/internal/site/cards.go`, replace the opening of `buildCardsContent` (lines 60-63):

```go
func buildCardsContent(dir, dirName string, files, subdirs []string) (content string, ok bool) {
	if !richCardTypes[dirName] || len(files) == 0 || len(subdirs) > 0 {
		return "", false
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })
```

with:

```go
func buildCardsContent(dir, dirName string, files, subdirs []string) (content string, ok bool) {
	cardType := dirName
	if !richCardTypes[dirName] {
		// Treasure leaves are nested (treasure/<tier>/<category>); render their
		// items as treasure cards even though the leaf dirName isn't "treasure".
		if len(subdirs) == 0 && len(files) > 0 && pathHasSegment(dir, "treasure") {
			cardType = "treasure"
		} else {
			return "", false
		}
	}
	if len(files) == 0 || len(subdirs) > 0 {
		return "", false
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })
```

Then, in the same function body, change the two places that use `dirName` for the *card type* (not the title) to use `cardType`:

- The wrapper check (currently `if wideCardTypes[dirName] {`) → `if wideCardTypes[cardType] {`
- The per-card call (currently `sb.WriteString(cardFor(dirName, fm, body, f, name))`) → `sb.WriteString(cardFor(cardType, fm, body, f, name))`

Leave the title line (`dirToTitle(dirName)`) as-is so the leaf shows "Consumable"/"Trinket"/etc.

Add this helper at the end of `cards.go`:

```go
// pathHasSegment reports whether any path segment of dir equals seg.
func pathHasSegment(dir, seg string) bool {
	for _, p := range strings.Split(filepath.ToSlash(dir), "/") {
		if p == seg {
			return true
		}
	}
	return false
}
```

(`strings`, `filepath`, `sort`, `os` are already imported in `cards.go`.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/scott/code/steelCompendium/workspace/steel-etl && devbox run -- go test ./internal/site/ -run 'TestBuildCardsContent_Treasure' -v`
Expected: PASS (both).

- [ ] **Step 5: Run the full site package tests (no regressions)**

Run: `cd /home/scott/code/steelCompendium/workspace/steel-etl && devbox run -- go test ./internal/site/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add internal/site/cards.go internal/site/cards_test.go
git commit -m "feat(site): render nested treasure leaf dirs as stat-cards"
```

---

## Task 3: Re-annotate the heroes Treasures chapter

**Files:**
- Modify: `steel-etl/input/heroes/Draw Steel Heroes.md` (Treasures chapter only)

This is a deterministic source transform applied by a script (≈12 category headers reclassified, ≈94 items annotated, 7 prose sections un-annotated), then verified with `git diff` + `steel-etl validate`. **Do not hand-edit 94 items.**

- [ ] **Step 1: Snapshot the current Treasures heading structure (for diff sanity)**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
awk '/<!-- @type: chapter \| @id: treasures -->/{f=1} /<!-- @type: chapter \| @id: titles -->/{f=0} f' \
  "steel-etl/input/heroes/Draw Steel Heroes.md" | grep -cE '^##### '
```
Record the count (number of `#####` item headings in the chapter — expected ~94). You'll confirm the same count gets `@type: treasure` annotations.

- [ ] **Step 2: Write the transform script**

Create `steel-etl/scripts/reannotate_heroes_treasures.py`:

```python
#!/usr/bin/env python3
"""One-shot, idempotent re-annotation of the heroes Treasures chapter.

- Category headers (#### N-Echelon Consumables/Trinkets, #### Leveled X Treasures,
  #### Other Leveled Treasures) -> @type: treasure-group with echelon + treasure-type.
- Individual items (##### ...) -> @type: treasure (inherit echelon/category from group).
- Prose sections -> annotation removed (folded into chapter body).
"""
import re
import pathlib

PATH = pathlib.Path("steel-etl/input/heroes/Draw Steel Heroes.md")

GROUP_RE = re.compile(
    r"^#### (?:(\d)(?:st|nd|rd|th)-Echelon (Consumables|Trinkets)"
    r"|Leveled (Armor|Implement|Weapon) Treasures"
    r"|(Other) Leveled Treasures)\s*$"
)
CAT = {"Consumables": "consumable", "Trinkets": "trinket",
       "Armor": "armor", "Implement": "implement", "Weapon": "weapon"}

PROSE_HEADINGS = {
    "#### What Does This Treasure Do?",
    "#### Wearing Treasures",
    "#### Wielding Treasures",
    "#### Magic and Psionic Treasures",
    "#### Stamina Bonuses and Damage Bonuses",
    "#### Leveled Benefits",
    "#### Carry Three Safely",
}

lines = PATH.read_text().split("\n")
start = next(i for i, l in enumerate(lines)
             if l.strip() == "<!-- @type: chapter | @id: treasures -->")
end = next(i for i, l in enumerate(lines)
          if i > start and l.strip().startswith("<!-- @type: chapter"))

def drop_prev_treasure_ann(out):
    if out and out[-1].strip().startswith("<!-- @type: treasure"):
        out.pop()

out, i = [], 0
groups = items = prose = 0
while i < len(lines):
    line = lines[i]
    if start < i < end:
        if line.strip() in PROSE_HEADINGS:
            drop_prev_treasure_ann(out)
            out.append(line); prose += 1; i += 1; continue
        m = GROUP_RE.match(line)
        if m:
            drop_prev_treasure_ann(out)
            ech, kind, lvl_kind, other = m.groups()
            if ech:
                ann = f"<!-- @type: treasure-group | @echelon: {ech} | @treasure-type: {CAT[kind]} -->"
            elif lvl_kind:
                ann = f"<!-- @type: treasure-group | @treasure-type: {CAT[lvl_kind]} -->"
            else:
                ann = "<!-- @type: treasure-group | @treasure-type: other -->"
            out.append(ann); out.append(line); groups += 1; i += 1; continue
        if line.startswith("##### "):
            if not (out and out[-1].strip().startswith("<!-- @type: treasure")):
                out.append("<!-- @type: treasure -->"); items += 1
            out.append(line); i += 1; continue
    out.append(line); i += 1

PATH.write_text("\n".join(out))
print(f"groups={groups} items={items} prose={prose}")
```

- [ ] **Step 3: Run the script**

Run: `cd /home/scott/code/steelCompendium/workspace && devbox run -- python3 steel-etl/scripts/reannotate_heroes_treasures.py`
Expected output: `groups=12 items=<~94> prose=7` (items count matches Step 1).

- [ ] **Step 4: Review the diff**

Run: `cd /home/scott/code/steelCompendium/workspace && git -C steel-etl diff -- "input/heroes/Draw Steel Heroes.md" | head -120`
Expected: 12 `#### ... ` headers now preceded by `<!-- @type: treasure-group | ... -->`; every `##### Item` preceded by `<!-- @type: treasure -->`; the 7 prose `#### ` headers no longer preceded by a treasure annotation. Spot-check that `#### Leveled Benefits` and `#### Carry Three Safely` are **un**-annotated (folded), not treated as groups.

- [ ] **Step 5: Validate annotations**

Run: `cd /home/scott/code/steelCompendium/workspace/steel-etl && devbox run -- go run ./cmd/steel-etl validate "input/heroes/Draw Steel Heroes.md"`
Expected: no "Unknown @type" for `treasure-group` (it's registered), no annotation errors.

- [ ] **Step 6: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add scripts/reannotate_heroes_treasures.py "input/heroes/Draw Steel Heroes.md"
git commit -m "content(heroes): re-annotate Treasures into group/item hierarchy"
```

---

## Task 4: Annotate the beastheart category headers

**Files:**
- Modify: `steel-etl/input/beastheart/Draw Steel Beastheart.md` (Rewards chapter)

Beastheart items are already `@type: treasure` (trinkets carry `@echelon: N`). We only need to annotate the 6 category headers so items inherit `treasure-type` (and, for trinkets, `echelon`). Items are left unchanged.

- [ ] **Step 1: Annotate the four trinket category headers**

Make these four edits in `steel-etl/input/beastheart/Draw Steel Beastheart.md`. Each inserts an annotation line immediately before the existing header.

Edit 1 — before `### 1st-Echelon Trinkets`:
```
<!-- @type: treasure-group | @echelon: 1 | @treasure-type: trinket -->
### 1st-Echelon Trinkets
```
Edit 2 — before `### 2nd-Echelon Trinket`:
```
<!-- @type: treasure-group | @echelon: 2 | @treasure-type: trinket -->
### 2nd-Echelon Trinket
```
Edit 3 — before `### 3rd-Echelon Trinket`:
```
<!-- @type: treasure-group | @echelon: 3 | @treasure-type: trinket -->
### 3rd-Echelon Trinket
```
Edit 4 — before `### 4th-Echelon Trinket`:
```
<!-- @type: treasure-group | @echelon: 4 | @treasure-type: trinket -->
### 4th-Echelon Trinket
```

- [ ] **Step 2: Annotate the two leveled category headers**

Edit 5 — before `### Leveled Armor Treasures`:
```
<!-- @type: treasure-group | @treasure-type: armor -->
### Leveled Armor Treasures
```
Edit 6 — before `### Leveled Weapon Treasures`:
```
<!-- @type: treasure-group | @treasure-type: weapon -->
### Leveled Weapon Treasures
```

- [ ] **Step 3: Validate**

Run: `cd /home/scott/code/steelCompendium/workspace/steel-etl && devbox run -- go run ./cmd/steel-etl validate "input/beastheart/Draw Steel Beastheart.md"`
Expected: no annotation errors, no unknown types.

- [ ] **Step 4: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add "input/beastheart/Draw Steel Beastheart.md"
git commit -m "content(beastheart): annotate treasure category headers"
```

---

## Task 5: Repoint inbound links to removed/changed treasure codes

**Files:**
- Modify: `steel-etl/input/heroes/Draw Steel Heroes.md`

Five `scc:` links currently target codes that no longer exist after this change (3 leveled *category* codes that become container directories with no item code; 2 *prose* codes that were folded into the chapter). Repoint all five to the Treasures chapter, `scc:mcdm.heroes.v1/chapter/treasures` (chapter links are an established pattern, e.g. `scc:mcdm.heroes.v1/chapter/combat`).

- [ ] **Step 1: Confirm the current targets and find any other references**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
grep -rn -E "scc:mcdm\.heroes\.v1/treasure/(leveled-armor-treasures|leveled-implement-treasures|leveled-weapon-treasures|leveled-benefits|magic-and-psionic-treasures)" \
  steel-etl/input v2/static_content 2>/dev/null
```
Expected: 5 hits, all in `input/heroes/Draw Steel Heroes.md`. If any appear in `v2/static_content`, repoint those too in the same way.

- [ ] **Step 2: Repoint the five links**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- python3 - <<'PY'
import pathlib
p = pathlib.Path("steel-etl/input/heroes/Draw Steel Heroes.md")
t = p.read_text()
new = "scc:mcdm.heroes.v1/chapter/treasures"
for old in [
    "scc:mcdm.heroes.v1/treasure/leveled-armor-treasures",
    "scc:mcdm.heroes.v1/treasure/leveled-implement-treasures",
    "scc:mcdm.heroes.v1/treasure/leveled-weapon-treasures",
    "scc:mcdm.heroes.v1/treasure/leveled-benefits",
    "scc:mcdm.heroes.v1/treasure/magic-and-psionic-treasures",
]:
    n = t.count(old)
    t = t.replace(old, new)
    print(f"{old}: {n}")
p.write_text(t)
PY
```
Expected: each printed count is `1`.

- [ ] **Step 3: Verify no stale targets remain**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
grep -rcE "scc:mcdm\.heroes\.v1/treasure/(leveled-armor-treasures|leveled-implement-treasures|leveled-weapon-treasures|leveled-benefits|magic-and-psionic-treasures)" \
  "steel-etl/input/heroes/Draw Steel Heroes.md"
```
Expected: `0`.

- [ ] **Step 4: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add "input/heroes/Draw Steel Heroes.md"
git commit -m "content(heroes): repoint treasure category/prose links to chapter"
```

---

## Task 6: Regenerate all outputs and verify end-to-end

**Files:** none (generated output only; do not edit generated files).

This runs the real pipeline (`gen --all` + `site` + index transform) and verifies the new hierarchy. It does **not** push — publishing is `just deploy` once the user approves.

- [ ] **Step 1: Run gen for all books**

Run: `cd /home/scott/code/steelCompendium/workspace/steel-etl && devbox run -- go run ./cmd/steel-etl gen --config pipeline.yaml --all`
Expected: completes without `duplicate SCC` errors. If a duplicate SCC error appears (e.g. two items slugify to the same id like the three Color Cloaks), note the offending headings — they need distinct `@id` annotations added in Task 3's source (add `@id` to the item annotations and re-run). Verify the "Color Cloak (Blue/Red/Yellow)" trio produced `color-cloak-blue/red/yellow`.

- [ ] **Step 2: Verify the generated data-rules directory tree**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
find data/data-rules/en/md/treasure -type d | sort
```
Expected directories include:
```
data/data-rules/en/md/treasure
data/data-rules/en/md/treasure/1st-echelon
data/data-rules/en/md/treasure/1st-echelon/consumable
data/data-rules/en/md/treasure/1st-echelon/trinket
data/data-rules/en/md/treasure/2nd-echelon/consumable
... (3rd, 4th)
data/data-rules/en/md/treasure/leveled/armor
data/data-rules/en/md/treasure/leveled/implement
data/data-rules/en/md/treasure/leveled/weapon
data/data-rules/en/md/treasure/leveled/other
```
And spot-check an item + its SCC:
```bash
grep -m1 '^scc:' data/data-rules/en/md/treasure/1st-echelon/consumable/black-ash-dart.md
```
Expected: `scc: mcdm.heroes.v1/treasure.1st-echelon.consumable/black-ash-dart`

- [ ] **Step 3: Verify beastheart items nested too**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
grep -rm1 '^scc:' data/data-rules/en/md/treasure/1st-echelon/trinket/precious-collar.md
find data/data-rules/en/md/treasure/leveled/weapon -name 'longclaw.md'
```
Expected: `scc: mcdm.beastheart.v1/treasure.1st-echelon.trinket/precious-collar` and the longclaw file exists under `leveled/weapon/`.

- [ ] **Step 4: Confirm the prose sections are gone as items but present in the chapter**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
ls data/data-rules/en/md/treasure/**/magic-and-psionic-treasures.md 2>/dev/null; echo "exit:$?"
grep -c "Magic and Psionic Treasures" data/data-rules/en/md/chapter/treasures.md
```
Expected: no `magic-and-psionic-treasures.md` item file; the phrase still appears in the rendered chapter body (count ≥ 1).

- [ ] **Step 5: Build the site + transform indexes**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
devbox run -- go run ./cmd/steel-etl site --config "$(cd .. && pwd)/v2/site.yaml"
cd /home/scott/code/steelCompendium/workspace/v2
devbox run -- python3 scripts/transform_indexes.py docs/Browse || true
```
(The `transform_indexes.py` argument mirrors the `deploy-v2` recipe; if it errors on args, run it as the recipe does — see `workspace/justfile` `deploy-v2`.)
Expected: site build reports treasure pages written; no errors.

- [ ] **Step 6: Verify the Browse hierarchy + index pages**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
find v2/docs/Browse/treasure -name index.md | sort
sed -n '1,40p' v2/docs/Browse/treasure/1st-echelon/consumable/index.md
```
Expected: index.md at `treasure/`, `treasure/1st-echelon/`, `treasure/1st-echelon/consumable/`, … `treasure/leveled/weapon/`, etc. The leaf `consumable/index.md` renders `# Consumable` with `sc-cards` containing the individual treasures (e.g. "Black Ash Dart"). The `treasure/index.md` and `treasure/1st-echelon/index.md` render expandable browse lists of their subdirectories.

- [ ] **Step 7: Verify SCC permalink stubs + repointed links resolve**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
ls v2/docs/scc/mcdm.heroes.v1/treasure.1st-echelon.consumable/black-ash-dart/index.html
grep -rc "scc:mcdm.heroes.v1/chapter/treasures" data/data-rules/en/md-linked/ | grep -v ':0' | head
```
Expected: the permalink stub exists for a nested treasure; the repointed chapter links are present (resolved) in the linked output. Confirm there are **no** unresolved `scc:` link warnings for the old codes in the gen output from Step 1.

- [ ] **Step 8: Commit generated output**

```bash
cd /home/scott/code/steelCompendium/workspace
git add data/ steel-etl/output/ 2>/dev/null; true
# The data-* dirs are separate git repos; commit each that changed:
for d in data/data-rules data/data-unified data/data-rules-clean data/data-bestiary steelCompendium.github.io; do
  if [ -d "$d/.git" ]; then git -C "$d" add -A && git -C "$d" commit -m "chore: treasure hierarchy regen" || true; fi
done
```
(If the data repos are managed by `just deploy` instead, skip manual commits here and note that the user should run `just deploy` to publish — see Task 7.)

---

## Task 7: Update docs

**Files:**
- Modify: `steel-etl/ANNOTATION-GUIDE.md`
- Modify: `workspace/CLAUDE.md` (SCC registry count note)

- [ ] **Step 1: Document the treasure hierarchy in the annotation guide**

In `steel-etl/ANNOTATION-GUIDE.md`, under the "Rewards" content-types table (around the `treasure` row, line ~120), add a `treasure-group` entry and a short pattern block. Insert after the Rewards table:

```markdown
### Treasure hierarchy

Treasures nest as `treasure/<tier>/<category>/<item>`. Category headings are
containers; individual treasures are items that inherit echelon + category from
the container via context (same pattern as `feature-group` → `ability`).

```markdown
<!-- @type: treasure-group | @echelon: 1 | @treasure-type: consumable -->
#### 1st-Echelon Consumables

<!-- @type: treasure -->
##### Black Ash Dart
...

<!-- @type: treasure-group | @treasure-type: weapon -->
#### Leveled Weapon Treasures

<!-- @type: treasure -->
##### Displacer
...
```

- `@echelon`: `1`–`4` for echelon-tiered treasures; omit for leveled treasures
  (tier becomes `leveled`).
- `@treasure-type`: `consumable` | `trinket` | `armor` | `implement` | `weapon` | `other`.
- An item may set its own `@echelon`/`@treasure-type` to override the container
  (beastheart trinkets carry per-item `@echelon`).
```

Also add `treasure-group` to the structural-types list (the table near line 87) as a container type that "provides echelon + treasure-type context to child treasures; emits no file" (mirroring `feature-group`).

- [ ] **Step 2: Update the SCC registry count note**

In `workspace/CLAUDE.md`, the "SCC (Steel Compendium Classification)" section states "Registry: 1,754 codes across 17 types". Re-run the count and update both numbers (a new `treasure-group` is a container, not a code-producing type, so type count is unchanged at 17; the code count rises because ~120 treasure items are now individually classified):

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
devbox run -- go run ./cmd/steel-etl classify --config pipeline.yaml --all 2>/dev/null | tail -5
```
Update the sentence to the new total code count reported. (If the exact phrasing/flags differ, use the "Sections classified" total from the Task 6 Step 1 gen output.)

- [ ] **Step 3: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add ANNOTATION-GUIDE.md
git -C /home/scott/code/steelCompendium/workspace add CLAUDE.md
git commit -m "docs: document treasure hierarchy + treasure-group annotation"
git -C /home/scott/code/steelCompendium/workspace commit -m "docs: refresh SCC registry count for treasure hierarchy"
```

- [ ] **Step 4: Hand off publishing**

Tell the user the change is complete and verified locally. To publish: `just deploy` from the workspace root (runs `gen --all` + site build + index transform and commits/pushes the v2 site and SCC API). Note that `just deploy` pushes to GitHub Pages — only run when ready.

---

## Self-Review Notes (verification against the spec)

- **Hierarchy = echelon-first** `treasure/<tier>/<category>` ✓ (Task 1, verified Task 6 Step 2).
- **Both books** ✓ (Task 3 heroes, Task 4 beastheart; verified Task 6 Steps 2-3).
- **Prose folded into chapter** ✓ (Task 3 prose handling; verified Task 6 Step 4).
- **Inbound links repointed** ✓ (Task 5; verified Task 6 Step 7).
- **Browse cards preserved at leaves** ✓ (Task 2; verified Task 6 Step 6).
- **`data-gen` untouched** ✓ (explicitly out of scope).
- **Type consistency:** annotation key `treasure-type` (hyphen) read by both `TreasureParser` (existing `treasure.go:38` context lookup) and new `TreasureGroupParser`; `echelon` key read by both. `echelonSlug` maps `1→1st-echelon` … `4→4th-echelon`, empty→`leveled`. `pathHasSegment` used only by `buildCardsContent`. `TreasureGroupParser` returns nil `TypePath` (no file) consistent with `FeatureGroupParser`.
- **Risks to watch:** (a) duplicate-SCC from slugify collisions (Color Cloak trio, apostrophe names) — caught at Task 6 Step 1, fix by adding `@id`. (b) `transform_indexes.py` invocation args — mirror the `deploy-v2` recipe exactly if the bare call errors. (c) SDK (`data-sdk-npm`) consumers walking `treasure/` may assume a flat layout — out of scope here, but flag to the user if the SDK build later complains.
