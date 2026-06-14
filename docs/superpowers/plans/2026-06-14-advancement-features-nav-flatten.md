# Advancement-features nav flatten + paired index cards — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In the v2 Browse tree, flatten each `advancement-features/` page to sit beside its base entity (beastheart companions + summoner fixtures) and render the group index as paired single-link stat-cards — leaving SCC codes, `/scc/` permalinks, and the `data/data-*` repos untouched.

**Architecture:** Two additions to the steel-etl **site builder** (`internal/site/`), mirroring the existing `hoistStatblockPath` "code≠path" pattern: (1) a pure path transform `flattenAdvancementFeaturesPath` wired into the same two places the hoist is; (2) a new index builder `buildAdvancementPairContent` that emits a 2-column `.sc-cards` grid of base+advancement pairs, plus one small CSS modifier in v2.

**Tech Stack:** Go (steel-etl), `go test -race`; CSS (v2 MkDocs Material). Run all Go commands through devbox: `devbox run -- <cmd>` from the workspace root.

**Spec:** `steel-etl/docs/superpowers/specs/2026-06-14-advancement-features-nav-flatten-design.md`

---

## File Structure

| File | Responsibility | Action |
|------|----------------|--------|
| `steel-etl/internal/site/build.go` | `flattenAdvancementFeaturesPath` (new); two wiring call-sites; one dispatch line in `buildIndexContent` | Modify |
| `steel-etl/internal/site/build_test.go` | Unit test for the path transform; integration test for wiring | Modify |
| `steel-etl/internal/site/advancement_pairs.go` | `buildAdvancementPairContent` index builder | Create |
| `steel-etl/internal/site/advancement_pairs_test.go` | Unit test for the index builder | Create |
| `v2/docs/stylesheets/steel-redesign.css` | `.sc-cards--pairs` 2-column grid modifier | Modify |
| `steel-etl/docs/site-builder.md`, `steel-etl/CLAUDE.md` | Document the new path transform + paired index | Modify |

All Go work is in package `site`. Helpers reused (already exist, do not redefine): `bestiaryGroupParents` (map), `card(file, icon, typeLabel, name, inner string) string`, `pathHasSegment(dir, seg string) bool`, `dirToTitle`, `fileToTitle`, `naturalLess`, `readFrontmatterName`, `hoistStatblockPath`. Valid `crestSVG` icon keys include `paw` and `skull`.

---

## Task 1: `flattenAdvancementFeaturesPath` path transform

**Files:**
- Modify: `steel-etl/internal/site/build.go` (add function after `hoistStatblockPath`, ~line 556)
- Test: `steel-etl/internal/site/build_test.go`

- [ ] **Step 1: Write the failing test**

Add to `build_test.go`:

```go
func TestFlattenAdvancementFeaturesPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// companion advancement page → sibling of its base species page
		{"monster/companion/beastheart/advancement-features/wolf.md",
			"monster/companion/beastheart/wolf-advancement-features.md"},
		// fixture advancement page → sibling of its base fixture page
		{"monster/fixture/demon/advancement-features/the-boil.md",
			"monster/fixture/demon/the-boil-advancement-features.md"},
		// base companion page is untouched
		{"monster/companion/beastheart/wolf.md",
			"monster/companion/beastheart/wolf.md"},
		// non-bestiary path untouched
		{"feature/ability/fury/level-1/gouge.md",
			"feature/ability/fury/level-1/gouge.md"},
		// no advancement-features segment → unchanged
		{"monster/goblins/cutter.md", "monster/goblins/cutter.md"},
	}
	for _, c := range cases {
		if got := flattenAdvancementFeaturesPath(c.in); got != c.want {
			t.Errorf("flattenAdvancementFeaturesPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/site/ -run TestFlattenAdvancementFeaturesPath"`
Expected: FAIL — `undefined: flattenAdvancementFeaturesPath`

- [ ] **Step 3: Write minimal implementation**

In `build.go`, immediately after the `hoistStatblockPath` function (after its closing `}` near line 556), add:

```go
// flattenAdvancementFeaturesPath collapses a non-leaf "advancement-features"
// folder in the bestiary tree, folding its name into the leaf filename
// (<id>.md → <id>-advancement-features.md) so the page sits beside its base
// entity instead of in an advancement-features/ sub-folder. Used by beastheart
// companions and summoner fixtures. Like hoistStatblockPath this is a deliberate
// code≠path divergence: the SCC CODE keeps its `.advancement-features` segment;
// only the Browse URL/sidebar changes. The slug deliberately echoes the SCC
// segment so the URL keeps a breadcrumb back to the code. Non-matching paths
// (no advancement-features segment, or outside a bestiary type root) are
// returned unchanged.
func flattenAdvancementFeaturesPath(relPath string) string {
	rel := filepath.ToSlash(relPath)
	parts := strings.Split(rel, "/")
	if len(parts) < 3 || !bestiaryGroupParents[parts[0]] {
		return relPath
	}
	for i, p := range parts {
		if i < len(parts)-1 && p == "advancement-features" {
			leaf := parts[len(parts)-1] // always <id>.md (the segment's only child)
			id := strings.TrimSuffix(leaf, ".md")
			out := append(parts[:i:i], id+"-advancement-features.md")
			return strings.Join(out, "/")
		}
	}
	return relPath
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/site/ -run TestFlattenAdvancementFeaturesPath -v"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add internal/site/build.go internal/site/build_test.go
git commit -m "feat(site): flattenAdvancementFeaturesPath transform (code≠path)"
```

---

## Task 2: Wire the transform into dest-path + link rewriting

**Files:**
- Modify: `steel-etl/internal/site/build.go` (two call-sites: ~line 221 and ~line 966)
- Test: `steel-etl/internal/site/build_test.go`

- [ ] **Step 1: Write the failing integration test**

Add to `build_test.go` (models `TestBuild_Groups`; `checkExists`/`checkNotExists` already exist in this file):

```go
func TestBuild_FlattensAdvancementFeatures(t *testing.T) {
	srcDir := t.TempDir()
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	files := map[string]string{
		"monster/companion/beastheart/statblock/wolf.md":            "---\nname: Wolf\ntype: feature-group\n---\n\nWolf statblock.",
		"monster/companion/beastheart/advancement-features/wolf.md": "---\nname: Wolf Advancement Features\ntype: featureblock\n---\n\nSee [the wolf](../statblock/wolf.md).",
	}
	for rel, content := range files {
		path := filepath.Join(srcDir, rel)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}

	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Sections:  []SectionConfig{{Name: "Browse", Include: []string{"monster/"}}},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	for _, e := range result.Errors {
		t.Errorf("Error: %s", e)
	}

	// Advancement page is flattened to a sibling of the (hoisted) base page.
	checkExists(t, docsDir, "Browse/monster/companion/beastheart/wolf-advancement-features.md")
	checkNotExists(t, docsDir, "Browse/monster/companion/beastheart/advancement-features/wolf.md")
	// Base page keeps its hoisted location.
	checkExists(t, docsDir, "Browse/monster/companion/beastheart/wolf.md")

	// The link from the advancement page resolves to the hoisted base sibling.
	data, _ := os.ReadFile(filepath.Join(docsDir, "Browse/monster/companion/beastheart/wolf-advancement-features.md"))
	if !strings.Contains(string(data), "(wolf.md)") {
		t.Errorf("expected link rewritten to sibling (wolf.md), got:\n%s", data)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/site/ -run TestBuild_FlattensAdvancementFeatures -v"`
Expected: FAIL — advancement file still at `advancement-features/wolf.md`, flattened path missing.

- [ ] **Step 3: Wire in the dest-path call-site**

In `build.go`, find (~line 221):

```go
		// Collapse the redundant statblock/ folder out of the site URL (code≠path).
		destRel = hoistStatblockPath(destRel)
```

Replace with:

```go
		// Collapse the redundant statblock/ folder out of the site URL (code≠path).
		destRel = hoistStatblockPath(destRel)
		// Flatten advancement-features/<id> beside its base entity (code≠path).
		destRel = flattenAdvancementFeaturesPath(destRel)
```

- [ ] **Step 4: Wire in the link-rewrite call-site**

In `build.go`, find (~line 966, inside `rewriteSectionLinks`):

```go
			relTarget = hoistStatblockPath(relTarget)
			targetFull = filepath.ToSlash(filepath.Join(targetSection, relTarget))
```

Replace with:

```go
			relTarget = hoistStatblockPath(relTarget)
			relTarget = flattenAdvancementFeaturesPath(relTarget)
			targetFull = filepath.ToSlash(filepath.Join(targetSection, relTarget))
```

- [ ] **Step 5: Run test to verify it passes**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/site/ -run TestBuild_FlattensAdvancementFeatures -v"`
Expected: PASS

- [ ] **Step 6: Run the full site package to check no regressions**

Run: `devbox run -- bash -c "cd steel-etl && go test -race ./internal/site/"`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
cd steel-etl
git add internal/site/build.go internal/site/build_test.go
git commit -m "feat(site): wire advancement-features flatten into dest path + link rewrite"
```

---

## Task 3: Paired index-card builder

**Files:**
- Create: `steel-etl/internal/site/advancement_pairs.go`
- Create: `steel-etl/internal/site/advancement_pairs_test.go`
- Modify: `steel-etl/internal/site/build.go` (`buildIndexContent` dispatch, ~line 1186)

- [ ] **Step 1: Write the failing test**

Create `advancement_pairs_test.go`:

```go
package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAdvancementPairContent(t *testing.T) {
	dir := t.TempDir()
	write := func(name, fm string) {
		os.WriteFile(filepath.Join(dir, name), []byte("---\n"+fm+"\n---\n\nbody"), 0644)
	}
	write("wolf.md", "name: Wolf\ntype: feature-group")
	write("wolf-advancement-features.md", "name: Wolf Advancement Features\ntype: featureblock")
	write("boar.md", "name: Boar\ntype: feature-group")
	write("boar-advancement-features.md", "name: Boar Advancement Features\ntype: featureblock")

	files := []string{"boar.md", "boar-advancement-features.md", "wolf.md", "wolf-advancement-features.md"}
	out, ok := buildAdvancementPairContent(filepath.Join("monster/companion/beastheart"), "beastheart", files, nil)
	if !ok {
		t.Fatal("expected ok=true for a dir with advancement pairs")
	}

	// 2-column pair grid wrapper.
	if !strings.Contains(out, `class="sc-cards sc-cards--pairs"`) {
		t.Errorf("missing sc-cards--pairs wrapper:\n%s", out)
	}
	// Each base is immediately followed by its advancement (base-first ordering).
	wolfBase := strings.Index(out, `href="wolf/"`)
	wolfAdv := strings.Index(out, `href="wolf-advancement-features/"`)
	if wolfBase < 0 || wolfAdv < 0 || wolfBase > wolfAdv {
		t.Errorf("expected base wolf card before its advancement card; base=%d adv=%d", wolfBase, wolfAdv)
	}
	// Companion crest + distinguishing eyebrows.
	if !strings.Contains(out, ">Companion<") || !strings.Contains(out, ">Advancement Features<") {
		t.Errorf("expected Companion and Advancement Features eyebrows:\n%s", out)
	}
}

func TestBuildAdvancementPairContent_NoPairs(t *testing.T) {
	if _, ok := buildAdvancementPairContent("monster/goblins", "goblins", []string{"cutter.md"}, nil); ok {
		t.Error("expected ok=false when no advancement-features leaves are present")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/site/ -run TestBuildAdvancementPairContent"`
Expected: FAIL — `undefined: buildAdvancementPairContent`

- [ ] **Step 3: Write the builder**

Create `advancement_pairs.go`:

```go
package site

// Paired index cards for the flattened beastheart-companion and summoner-fixture
// group dirs. After flattenAdvancementFeaturesPath (build.go) runs in buildSection,
// each group dir holds <id>.md + <id>-advancement-features.md as flat siblings.
// This builder pairs them into a 2-up .sc-cards grid (base card immediately
// followed by its advancement card) so a pair shares a row. SITE-ONLY: it reads
// the generated md-linked pages' frontmatter; the shared data repos are untouched.
// Styled by docs/stylesheets/steel-redesign.css (.sc-cards--pairs).

import (
	"path/filepath"
	"sort"
	"strings"
)

const advFeatSuffix = "-advancement-features"

// buildAdvancementPairContent renders a group dir whose leaves come in
// <id>.md + <id>-advancement-features.md pairs as a 2-column pair grid.
// ok=false → caller falls through to the default index builders.
func buildAdvancementPairContent(dir, dirName string, files, subdirs []string) (string, bool) {
	advByBase := map[string]string{}
	var bases []string
	for _, f := range files {
		id := strings.TrimSuffix(f, ".md")
		if strings.HasSuffix(id, advFeatSuffix) {
			advByBase[strings.TrimSuffix(id, advFeatSuffix)] = f
		} else {
			bases = append(bases, f)
		}
	}
	if len(advByBase) == 0 {
		return "", false
	}
	sort.Slice(bases, func(i, j int) bool { return naturalLess(bases[i], bases[j]) })

	baseEyebrow, icon := "Companion", "paw"
	if pathHasSegment(dir, "fixture") {
		baseEyebrow, icon = "Fixture", "skull"
	}

	cardName := func(file string) string {
		if n := readFrontmatterName(filepath.Join(dir, file)); n != "" {
			return n
		}
		return fileToTitle(file)
	}

	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n")
	sb.WriteString("<div class=\"sc-cards sc-cards--pairs\">\n")

	seen := map[string]bool{}
	for _, bf := range bases {
		id := strings.TrimSuffix(bf, ".md")
		seen[id] = true
		name := cardName(bf)
		sb.WriteString(card(bf, icon, baseEyebrow, name, ""))
		if af, ok := advByBase[id]; ok {
			// Advancement card shares its base's name; the eyebrow distinguishes it.
			sb.WriteString(card(af, icon, "Advancement Features", name, ""))
		}
	}
	// Defensive: an advancement page with no base sibling renders on its own.
	var orphans []string
	for base, af := range advByBase {
		if !seen[base] {
			orphans = append(orphans, af)
		}
	}
	sort.Slice(orphans, func(i, j int) bool { return naturalLess(orphans[i], orphans[j]) })
	for _, af := range orphans {
		sb.WriteString(card(af, icon, "Advancement Features", cardName(af), ""))
	}

	sb.WriteString("</div>\n")
	return sb.String(), true
}
```

- [ ] **Step 4: Wire the builder into the dispatch**

In `build.go`, find the top of `buildIndexContent` (~line 1186):

```go
func buildIndexContent(dir, dirName string, files, subdirs []string) string {
	// Rich stat-cards for supported index types (kit, …); falls back below.
	if cards, ok := buildCardsContent(dir, dirName, files, subdirs); ok {
		return cards
	}
```

Insert the pair builder as the first check:

```go
func buildIndexContent(dir, dirName string, files, subdirs []string) string {
	// Flattened companion/fixture group dirs: base + advancement-features pairs.
	if pairs, ok := buildAdvancementPairContent(dir, dirName, files, subdirs); ok {
		return pairs
	}
	// Rich stat-cards for supported index types (kit, …); falls back below.
	if cards, ok := buildCardsContent(dir, dirName, files, subdirs); ok {
		return cards
	}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `devbox run -- bash -c "cd steel-etl && go test -race ./internal/site/ -run 'TestBuildAdvancementPairContent|TestBuild_'"`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd steel-etl
git add internal/site/advancement_pairs.go internal/site/advancement_pairs_test.go internal/site/build.go
git commit -m "feat(site): paired base+advancement index cards for flattened group dirs"
```

---

## Task 4: 2-column CSS modifier in v2

**Files:**
- Modify: `v2/docs/stylesheets/steel-redesign.css` (after the `.md-typeset .sc-cards` rule, ~line 146)

Not TDD (CSS); verified by build + visual check in Task 5.

- [ ] **Step 1: Add the modifier rule**

In `steel-redesign.css`, immediately after the `.md-typeset .sc-cards { … }` block (the rule ending with `gap: 1rem; }`, ~line 146), add:

```css
/* Paired base + advancement-features cards (beastheart companions, summoner
   fixtures): force exactly two columns so each base/advancement pair shares a
   row. Collapses to one column on phones. Emitted by steel-etl
   buildAdvancementPairContent as <div class="sc-cards sc-cards--pairs">. */
.md-typeset .sc-cards--pairs {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
@media (max-width: 44.9375em) {
  .md-typeset .sc-cards--pairs {
    grid-template-columns: 1fr;
  }
}
```

- [ ] **Step 2: Commit**

```bash
cd v2
git add docs/stylesheets/steel-redesign.css
git commit -m "style: 2-column sc-cards--pairs grid for paired advancement cards"
```

---

## Task 5: Full build verification

**Files:** none (verification only). Run from the workspace root.

- [ ] **Step 1: Full Go test + vet**

Run: `devbox run -- bash -c "cd steel-etl && go test -race ./... && go vet ./..."`
Expected: all PASS, no vet findings.

- [ ] **Step 2: Build the site and inspect the flattened tree**

Run:
```bash
devbox run -- bash -c "cd steel-etl && go run ./cmd/steel-etl gen --all --config pipeline.yaml && go run ./cmd/steel-etl site --config ../v2/site.yaml"
```
Then:
```bash
find v2/docs/Browse/monster/companion/beastheart -maxdepth 1 -name '*advancement-features.md' | sort
find v2/docs/Browse/monster/fixture -name '*advancement-features.md' | sort
test -d v2/docs/Browse/monster/companion/beastheart/advancement-features && echo "STILL NESTED (bad)" || echo "flattened (good)"
```
Expected: 14 companion `*-advancement-features.md` siblings, 4 fixture ones; "flattened (good)".

- [ ] **Step 3: Confirm the paired index + permalink survival**

Run:
```bash
grep -c 'sc-cards--pairs' v2/docs/Browse/monster/companion/beastheart/index.md
grep -c 'wolf-advancement-features/' v2/docs/Browse/monster/companion/beastheart/index.md
ls v2/docs/scc/mcdm.beastheart.v1/ | grep advancement-features | head
```
Expected: index has the `sc-cards--pairs` grid and links the flattened slug; the `/scc/…advancement-features…` permalink stub still exists (SCC code unchanged).

- [ ] **Step 4: Visual spot-check (optional but recommended)**

Run: `devbox run -- mkdocs build -f v2/mkdocs.yml` then open `v2/site/Browse/monster/companion/beastheart/index.html` — confirm each companion + its advancement card share a row, base-first, with paw/skull crests.

- [ ] **Step 5: Commit any regenerated output if the repo tracks it**

`data/` is gitignored build output; `v2/docs/Browse/**` is generated. Only commit generated files if your deploy flow tracks them (it normally does not — deploy regenerates). No commit expected here.

---

## Task 6: Documentation sync

**Files:**
- Modify: `steel-etl/docs/site-builder.md` (path-transform section)
- Modify: `steel-etl/CLAUDE.md` (Statblocks gotchas — note the flatten alongside `hoistStatblockPath`)

- [ ] **Step 1: Document the path transform in site-builder.md**

In `steel-etl/docs/site-builder.md`, in the section describing `hoistStatblockPath` / path relocations, add a short paragraph:

```markdown
### Advancement-features flatten

`flattenAdvancementFeaturesPath` (build.go) collapses a non-leaf
`advancement-features/` folder in the bestiary tree, folding its name into the
leaf (`…/advancement-features/<id>.md` → `…/<id>-advancement-features.md`) so a
beastheart-companion or summoner-fixture advancement page sits beside its base
entity. Like `hoistStatblockPath` it is a deliberate **code≠path** divergence
(the SCC code keeps `.advancement-features`) and is applied in the same two
places — the dest-path computation in `buildSection` and the inbound-link mirror
in `rewriteSectionLinks`. The flattened group dir is then rendered by
`buildAdvancementPairContent` (advancement_pairs.go) as a 2-up `.sc-cards
--pairs` grid pairing each base with its advancement card.
```

- [ ] **Step 2: Note it in CLAUDE.md Statblocks gotchas**

In `steel-etl/CLAUDE.md`, in the Statblocks "Code≠path" bullet, append a sentence:

```markdown
Advancement-features pages additionally flatten beside their base entity
(`flattenAdvancementFeaturesPath`: `…/advancement-features/<id>` →
`…/<id>-advancement-features`) for beastheart companions and summoner fixtures —
nav-only, SCC code unchanged — and the group index pairs them via
`buildAdvancementPairContent`.
```

- [ ] **Step 3: Update the workspace SCC current-state note**

In the workspace `CLAUDE.md` SCC section, the Companions and Fixtures bullets state where pages sit in Browse. Append to each a clause noting the advancement page now flattens to `…/<id>-advancement-features` (nav-only; SCC code unchanged). Keep it to one clause per bullet — this file is a router, not a log.

- [ ] **Step 4: Append a dated entry to the SCC log**

This is a Browse-nav change (not an SCC scheme change), but the companion/fixture browse layout is tracked in `workspace/docs/scc-log.md`. Add a dated `2026-06-14` entry: advancement-features flattened in Browse for companions + fixtures; SCC codes/permalinks unchanged; link the spec + this plan.

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add docs/site-builder.md CLAUDE.md
git commit -m "docs: advancement-features nav flatten + paired index"
cd ..
git add CLAUDE.md docs/scc-log.md
git commit -m "docs: advancement-features Browse flatten (companions + fixtures)"
```

---

## Self-Review

- **Spec coverage:** Part 1 path flatten → Tasks 1–2 (both call-sites wired, matching the spec's two-place requirement). Part 2 paired separate single-link cards, base-first ordering, 2-column guarantee → Tasks 3–4. Both families (companion + fixture) → covered by the `bestiaryGroupParents`/`monster/` guard (Task 1) and the `pathHasSegment(dir, "fixture")` branch (Task 3). Non-goals (no SCC/data/page-content change) → verified in Task 5 Step 3. Docs sync → Task 6.
- **Placeholder scan:** none — every code/test step has complete code; commands have expected output.
- **Type consistency:** `flattenAdvancementFeaturesPath(string) string`, `buildAdvancementPairContent(dir, dirName string, files, subdirs []string) (string, bool)`, and `card(file, icon, typeLabel, name, inner string) string` are used identically across tasks. Wrapper class `sc-cards--pairs` matches between the Go emitter (Task 3) and the CSS (Task 4). Suffix constant `advFeatSuffix = "-advancement-features"` matches the slug produced by Task 1.
