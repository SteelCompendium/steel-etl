# Book-Faithful Aggregate Pages Implementation Plan

> **STATUS: ✅ COMPLETED 2026-05-29** via subagent-driven-development (all 7 tasks; per-task + final holistic review passed). Shipped on branch `book-faithful-pages` across three repos. PRs: steel-etl#1 (core), v2#3 (`site.yaml`+docs), workspace#2 (docs) — merge steel-etl#1 first. All checkboxes below are retained as an execution record. Known unrelated pre-existing failure left as-is: `TestBuild_GeneratesIndexPages`. Deferred items captured in workspace `FOLLOWUPS.md`.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every site page (`Browse/*`, `Read/chapter/*`) render its source section exactly as it appears in the book — the section's own content plus every nested child inline, in document order — instead of the current disassemble-then-regroup composite that drops or duplicates nested content.

**Architecture:** Add a generic `RenderSubtree` serializer that walks a parsed `Section` tree in document order, emitting each node's body with headings normalized (page title = H1, descendants nest by source depth) and ability statblocks un-blockquoted. The pipeline stores this as a new `ParsedContent.PageBody`, populated for every section. Only the `md-linked` generator (the site builder's input) emits `PageBody`; all structured formats (`md`, `json`, `yaml`, `dse`, etc.) are unchanged. The site builder's `composites:` reassembly is removed — class/ancestry/chapter pages now come from plain section-mapping of the now-complete `md-linked` files.

**Tech Stack:** Go 1.26 (via devbox), `go test`, the steel-etl pipeline + site builder, MkDocs Material (v2).

---

## Background / Why

Current behavior (the bug being fixed):

- `gen` splits each annotated source section into its own file. A section's file contains its own body + unannotated children, but **annotated** children (traits, abilities) become separate files. So container pages (`class/censor.md`, `chapter/classes.md`, `feature/trait/censor/.../censor-abilities.md`) were stubs missing their nested content.
- The site builder reassembled class pages via `composites:` by collecting the class's trait files and ability files and **grouping them by type** ("N-Level Features", "N-Level Abilities"). Because traits also embed their abilities, every ability appeared **twice** on the class page (e.g. `Judgment` and all of `Censor Abilities`).
- The user wants `Browse/class/censor` (and every page) to read like the book with surrounding sections removed: nested, in source order, no duplication.

Key facts discovered during research (do not re-derive):

- `ParsedContent` is defined in `steel-etl/internal/content/parser.go:18`.
- `parser.Section` (`steel-etl/internal/parser/section.go:6`) has: `Heading`, `HeadingLevel`, `Annotation`, `BodySource` (immediate body between this heading and the first child), `Children`, `Parent`. `Type()` returns the `@type` annotation or `""`.
- `Section.BodySource` is the **immediate** body only; children (annotated or not) live in `Section.Children`.
- Ability statblocks are blockquoted in source (`> ###### Name`). `AbilityParser` (`steel-etl/internal/content/ability.go:26`) calls `stripBlockquotePrefix(section.FullBodySource())` to un-blockquote them. **Genuine flavor quotes** (e.g. the Censor class "We FIGHT!" blockquote) are NOT ability sections and must stay blockquoted. So blockquote-stripping must be keyed to `Type()=="ability"`.
- Helpers available in the `content` package: `CleanHeading(s)` (`helpers.go:33`, strips cost suffixes like `(11 Piety)`), `stripBlockquotePrefix(s)` (`ability.go:114`), `Slugify(s)` (`helpers.go:41`).
- Link resolution: `scc.Resolver.ResolveLinks(content, relativeTo, mode)` (`steel-etl/internal/scc/resolver.go:46`) resolves all `scc:` links in `content` relative to the `relativeTo` SCC code. `LinkAll` is the default mode (`resolver.go:16`). So a full-subtree body with raw `scc:` links resolves correctly when passed the container's own SCC code.
- `md-linked` is written by `LinkedGenerator.WriteSection` (`steel-etl/internal/output/linked.go:22`), which builds a `content.ParsedContent` copy with `Body: g.Resolver.ResolveLinks(parsed.Body, sccCode, g.LinkMode)` then calls `BuildMarkdownFile`.
- `BuildMarkdownFile` (`steel-etl/internal/output/generator.go:72`) writes frontmatter + `parsed.Body`.
- Pipeline walk: `steel-etl/internal/pipeline/pipeline.go:97-158`. For each section it calls `p.Parse(contextStack, section)` to get `parsed`, classifies it, then `for _, gen := range generators { gen.WriteSection(sccCode, parsed) }`. The live `*parser.Section` is in scope at line 114 — this is where `PageBody` gets populated.
- Site builder: `steel-etl/internal/site/build.go`. Composite assembly is `assembleComposites`/`assembleComposite` (`build.go:498-677`). H1 injection is `injectH1` (`build.go:275`). The composite config comes from `v2/site.yaml` `composites:` (class + ancestry).
- `v2/site.yaml` composites: `base: class` (include `feature/trait/{name}` + `feature/ability/{name}`) and `base: ancestry` (include `feature/trait/{name}-traits`, `remove_sources: true`). Section `Browse` excludes `feature/trait/ancestry-traits`; group `kit` flattens `feature/ability` → `Kits/`.

Interim state to be aware of (uncommitted working-tree changes from the session that discovered this bug):
- `steel-etl/internal/content/feature.go` — changed to embed all abilities into a trait's `Body` and only set `Children["ability"]` when exactly one ability child.
- `steel-etl/internal/parser/section.go` — added `FullBodySourceWithAbilities()`.
- `steel-etl/internal/content/fullbody_test.go` — added `TestFeatureParser_MultiAbilityContainerEmbedsAllInline` and `TestFeatureParser_SingleAbilityTraitStillEmbeds`.

Task 0 reconciles this interim state. The generic `PageBody` mechanism supersedes the interim `Body` rendering change, but the "embed `Children["ability"]` only when exactly one ability child" rule is kept (it is correct for the SDK trait schema, which has a singular `ability` field).

## File Structure

| File | Responsibility | Action |
|------|----------------|--------|
| `steel-etl/internal/content/render_subtree.go` | `RenderSubtree(section)` + helpers: book-order, heading-normalized, blockquote-aware serialization of a section subtree | Create |
| `steel-etl/internal/content/render_subtree_test.go` | Unit tests for `RenderSubtree` | Create |
| `steel-etl/internal/content/parser.go` | Add `PageBody string` field to `ParsedContent` | Modify (`:18`) |
| `steel-etl/internal/pipeline/pipeline.go` | Populate `parsed.PageBody = content.RenderSubtree(section)` in the walk | Modify (`:114-121`) |
| `steel-etl/internal/output/linked.go` | Emit `PageBody` (link-resolved) instead of `Body`, falling back to `Body` when `PageBody` is empty | Modify (`:22-48`) |
| `steel-etl/internal/output/linked_test.go` | Test that `md-linked` uses `PageBody` when present | Modify |
| `steel-etl/internal/content/feature.go` | Keep single-ability `Children` embed (len==1 only); restore `Body` to structured `FullBodySource` (no inline ability append) | Modify |
| `steel-etl/internal/parser/section.go` | Remove the interim `FullBodySourceWithAbilities()` (superseded by `RenderSubtree`) | Modify |
| `steel-etl/internal/content/fullbody_test.go` | Update interim tests to match the structured-`Body` contract | Modify |
| `v2/site.yaml` | Remove `composites:` for class + ancestry | Modify |
| `steel-etl/internal/site/build.go` | Remove now-dead composite assembly code; keep section mapping, exclusions, groups | Modify |
| `steel-etl/internal/site/build_test.go` | Remove/adjust composite tests | Modify |
| `ARCHITECTURE.md`, `steel-etl/CLAUDE.md`, `v2/CLAUDE.md` | Document the new rendering model | Modify |

---

## Task 0: Reconcile the interim hotfix to a clean baseline

**Files:**
- Modify: `steel-etl/internal/content/feature.go`
- Modify: `steel-etl/internal/parser/section.go`
- Modify: `steel-etl/internal/content/fullbody_test.go`

- [x] **Step 1: Restore `feature.go` `Body` to structured form, keep len==1 `Children` embed**

In `steel-etl/internal/content/feature.go`, the embed block should read (replacing the interim version that set `result.Body = section.FullBodySourceWithAbilities()`):

```go
	result := &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    typePath,
		ItemID:      id,
	}

	// Embed a single child ability as a structured nested object for the SDK
	// trait schema (which has a singular `ability` field). This only applies to
	// single-ability traits (e.g. "Faithful Friend"). Multi-ability containers
	// (e.g. "Censor Abilities") do NOT get a singular embed; their abilities are
	// rendered on the page via PageBody/RenderSubtree, not the structured Body.
	abilityChildren := collectAbilityChildren(section)
	if len(abilityChildren) == 1 {
		abilityParser := &AbilityParser{}
		parsed, err := abilityParser.Parse(context.NewContextStack(nil), abilityChildren[0])
		if err == nil {
			result.Children = map[string]*ParsedContent{
				"ability": parsed,
			}
		}
	}

	return result, nil
}
```

Keep the `collectAbilityChildren` helper that already exists below `Parse`. Remove the now-unused `strings` import if present (it was removed in the interim change; confirm the import block has only `context` and `parser`).

- [x] **Step 2: Remove the interim `FullBodySourceWithAbilities` from `section.go`**

In `steel-etl/internal/parser/section.go`, delete the `FullBodySourceWithAbilities` method entirely (the block beginning `// FullBodySourceWithAbilities behaves like FullBodySource...`). `FullBodySource` stays.

- [x] **Step 3: Update the interim tests in `fullbody_test.go`**

`TestFeatureParser_SingleAbilityTraitStillEmbeds` stays as-is (still valid: single-ability trait sets `Children["ability"]`), but its body assertions must not require the ability inline. Replace its two body assertions with structured-`Body` expectations:

```go
	if result.Children == nil {
		t.Fatal("single-ability trait should embed its ability in Children")
	}
	if _, ok := result.Children["ability"]; !ok {
		t.Error("single-ability trait should set Children[\"ability\"]")
	}
	if !strings.Contains(result.Body, "You have the following ability.") {
		t.Error("single-ability trait body should keep its intro prose")
	}
	// Body is structured (own content only); the ability is rendered via PageBody,
	// not appended to Body.
	if strings.Contains(result.Body, "An animal spirit is drawn to you.") {
		t.Error("structured Body should NOT inline the ability (PageBody handles rendering)")
	}
```

Replace `TestFeatureParser_MultiAbilityContainerEmbedsAllInline` with a test asserting the structured-`Body` contract (rendering is covered by `RenderSubtree` tests in Task 1):

```go
func TestFeatureParser_MultiAbilityContainerNoSingularEmbed(t *testing.T) {
	section := &parser.Section{
		Heading:      "Censor Abilities",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "You use a blend of martial techniques and divine magic.",
		Children: []*parser.Section{
			{
				Heading:      "Signature Ability",
				HeadingLevel: 5,
				BodySource:   "Choose one signature ability.",
				Children: []*parser.Section{
					{Heading: "Back Blasphemer!", HeadingLevel: 6, Annotation: map[string]string{"type": "ability"}, BodySource: "> *desc1*"},
					{Heading: "Halt Miscreant!", HeadingLevel: 6, Annotation: map[string]string{"type": "ability"}, BodySource: "> *desc2*"},
				},
			},
		},
	}
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "censor"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "level": "1"})

	result, err := (&FeatureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	// Multi-ability container: no singular structured embed.
	if result.Children != nil {
		if _, ok := result.Children["ability"]; ok {
			t.Error("multi-ability container must not set Children[\"ability\"]")
		}
	}
	// Structured Body is own content only.
	if strings.Contains(result.Body, "desc1") || strings.Contains(result.Body, "desc2") {
		t.Error("structured Body must not inline ability statblocks")
	}
}
```

- [x] **Step 4: Build and run content + parser tests**

Run: `devbox run -- go -C steel-etl test ./internal/content/ ./internal/parser/`
Expected: PASS (build clean, no unused imports).

- [x] **Step 5: Commit**

```bash
git -C steel-etl add internal/content/feature.go internal/parser/section.go internal/content/fullbody_test.go
git -C steel-etl commit -m "refactor: structured trait Body, keep single-ability embed only"
```

---

## Task 1: Add the `RenderSubtree` serializer

**Files:**
- Create: `steel-etl/internal/content/render_subtree.go`
- Test: `steel-etl/internal/content/render_subtree_test.go`

- [x] **Step 1: Write the failing tests**

Create `steel-etl/internal/content/render_subtree_test.go`:

```go
package content

import (
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestRenderSubtree_NormalizesHeadingsAndOrder(t *testing.T) {
	// Class (H2) -> feature-group (H3) -> trait (H4) -> subheading (H5) -> ability (H6)
	class := &parser.Section{
		Heading:      "Censor",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "censor"},
		BodySource:   "Demons fear you.\n\n> \"We FIGHT!\"\n> **Sir V**",
		Children: []*parser.Section{
			{
				Heading:      "1st-Level Features",
				HeadingLevel: 3,
				Annotation:   map[string]string{"type": "feature-group", "level": "1"},
				BodySource:   "As a 1st-level censor...",
				Children: []*parser.Section{
					{
						Heading:      "Censor Abilities",
						HeadingLevel: 4,
						Annotation:   map[string]string{"type": "feature"},
						BodySource:   "You use a blend.",
						Children: []*parser.Section{
							{
								Heading:      "Signature Ability",
								HeadingLevel: 5,
								BodySource:   "Choose one.",
								Children: []*parser.Section{
									{Heading: "Back Blasphemer!", HeadingLevel: 6, Annotation: map[string]string{"type": "ability"}, BodySource: "> *flavor1*\n> \n> | t | t |"},
									{Heading: "Halt Miscreant!", HeadingLevel: 6, Annotation: map[string]string{"type": "ability"}, BodySource: "> *flavor2*"},
								},
							},
						},
					},
				},
			},
		},
	}

	got := RenderSubtree(class)

	// Own body present, flavor quote stays blockquoted
	if !strings.Contains(got, "Demons fear you.") {
		t.Error("own body missing")
	}
	if !strings.Contains(got, "> \"We FIGHT!\"") {
		t.Error("genuine flavor blockquote must stay blockquoted")
	}
	// Heading normalization: class is the H1 (added by site), so depth-1 child = H2
	if !strings.Contains(got, "## 1st-Level Features") {
		t.Error("feature-group should normalize to H2")
	}
	if !strings.Contains(got, "### Censor Abilities") {
		t.Error("trait should normalize to H3")
	}
	if !strings.Contains(got, "#### Signature Ability") {
		t.Error("subheading should normalize to H4")
	}
	if !strings.Contains(got, "##### Back Blasphemer!") {
		t.Error("ability should normalize to H5")
	}
	// Ability statblocks un-blockquoted
	if strings.Contains(got, "> *flavor1*") {
		t.Error("ability body must be un-blockquoted")
	}
	if !strings.Contains(got, "*flavor1*") {
		t.Error("ability flavor text should still be present (un-blockquoted)")
	}
	// Document order
	iBack := strings.Index(got, "Back Blasphemer!")
	iHalt := strings.Index(got, "Halt Miscreant!")
	if !(iBack >= 0 && iBack < iHalt) {
		t.Errorf("abilities out of order: back=%d halt=%d", iBack, iHalt)
	}
}

func TestRenderSubtree_LeafEqualsOwnBody(t *testing.T) {
	// A leaf ability: PageBody equals its un-blockquoted body, no headings added.
	ability := &parser.Section{
		Heading:      "Back Blasphemer!",
		HeadingLevel: 6,
		Annotation:   map[string]string{"type": "ability"},
		BodySource:   "> *You channel power.*\n> \n> **Power Roll + Presence:**",
	}
	got := RenderSubtree(ability)
	if strings.Contains(got, "> *You channel") {
		t.Error("leaf ability should be un-blockquoted")
	}
	if !strings.Contains(got, "*You channel power.*") {
		t.Error("leaf ability content missing")
	}
	if strings.Contains(got, "#") {
		t.Error("leaf with no children should add no headings")
	}
}

func TestRenderSubtree_ChapterPreservesSourceLevels(t *testing.T) {
	// Chapter is H1, so a class child stays H2 (book-faithful).
	chapter := &parser.Section{
		Heading:      "Classes",
		HeadingLevel: 1,
		Annotation:   map[string]string{"type": "chapter", "id": "classes"},
		BodySource:   "How classes work.",
		Children: []*parser.Section{
			{Heading: "Censor", HeadingLevel: 2, Annotation: map[string]string{"type": "class", "id": "censor"}, BodySource: "Demons fear you."},
		},
	}
	got := RenderSubtree(chapter)
	if !strings.Contains(got, "## Censor") {
		t.Error("class under chapter should be H2")
	}
	if !strings.Contains(got, "How classes work.") {
		t.Error("chapter own body missing")
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `devbox run -- go -C steel-etl test ./internal/content/ -run TestRenderSubtree`
Expected: FAIL with "undefined: RenderSubtree".

- [x] **Step 3: Implement `RenderSubtree`**

Create `steel-etl/internal/content/render_subtree.go`:

```go
package content

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// RenderSubtree serializes a section's entire subtree as book-order markdown:
// the section's own immediate body, followed by every descendant (annotated or
// not) inline in document order. Heading levels are normalized so the section
// itself occupies the page's H1 (added separately by the site builder via H1
// injection) and descendants nest by their source depth. Ability statblocks
// (sections with @type: ability), which are blockquoted in source, are
// un-blockquoted to match how standalone ability pages render; genuine flavor
// blockquotes (which are not ability sections) are preserved.
//
// scc: links are left in their raw form; the md-linked generator resolves them
// relative to the page's own SCC code.
func RenderSubtree(section *parser.Section) string {
	return renderSubtree(section, section.HeadingLevel)
}

func renderSubtree(section *parser.Section, rootLevel int) string {
	var parts []string

	if body := nodeBody(section); body != "" {
		parts = append(parts, body)
	}

	for _, child := range section.Children {
		level := 1 + (child.HeadingLevel - rootLevel)
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		heading := strings.Repeat("#", level) + " " + CleanHeading(child.Heading)
		childBody := renderSubtree(child, rootLevel)
		if childBody != "" {
			parts = append(parts, heading+"\n\n"+childBody)
		} else {
			parts = append(parts, heading)
		}
	}

	return strings.Join(parts, "\n\n")
}

// nodeBody returns a section's immediate body, un-blockquoted for ability
// sections (whose statblocks are blockquoted in source).
func nodeBody(section *parser.Section) string {
	body := section.BodySource
	if section.Type() == "ability" {
		body = stripBlockquotePrefix(body)
	}
	return body
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `devbox run -- go -C steel-etl test ./internal/content/ -run TestRenderSubtree -v`
Expected: PASS for all three tests.

- [x] **Step 5: Commit**

```bash
git -C steel-etl add internal/content/render_subtree.go internal/content/render_subtree_test.go
git -C steel-etl commit -m "feat: add RenderSubtree book-order serializer"
```

---

## Task 2: Add `PageBody` to `ParsedContent` and populate it in the pipeline

**Files:**
- Modify: `steel-etl/internal/content/parser.go:18`
- Modify: `steel-etl/internal/pipeline/pipeline.go:114`

- [x] **Step 1: Add the `PageBody` field**

In `steel-etl/internal/content/parser.go`, inside the `ParsedContent` struct, add after the `Body` field:

```go
	// PageBody is a full book-order render of this section's subtree (own body +
	// all nested descendants inline), used by reading-format outputs (md-linked).
	// Empty for sections where it is not populated; consumers fall back to Body.
	PageBody string
```

- [x] **Step 2: Populate `PageBody` in the pipeline walk**

In `steel-etl/internal/pipeline/pipeline.go`, immediately after the successful `parsed, err := p.Parse(contextStack, section)` block (after `result.ParsedSections++` at `:120`), add:

```go
			// Full book-order render of this section's subtree for reading pages.
			parsed.PageBody = content.RenderSubtree(section)
```

Confirm `content` is already imported in this file (it is used elsewhere). If not, add `"github.com/SteelCompendium/steel-etl/internal/content"` to the imports.

- [x] **Step 3: Build**

Run: `devbox run -- go -C steel-etl build ./...`
Expected: clean build.

- [x] **Step 4: Commit**

```bash
git -C steel-etl add internal/content/parser.go internal/pipeline/pipeline.go
git -C steel-etl commit -m "feat: populate ParsedContent.PageBody in pipeline walk"
```

---

## Task 3: Emit `PageBody` from the `md-linked` generator

**Files:**
- Modify: `steel-etl/internal/output/linked.go:22-48`
- Test: `steel-etl/internal/output/linked_test.go`

- [x] **Step 1: Write the failing test**

Add to `steel-etl/internal/output/linked_test.go` (match the existing test package/helpers in that file; it already constructs a `LinkedGenerator` with a resolver):

```go
func TestLinkedGenerator_UsesPageBodyWhenPresent(t *testing.T) {
	dir := t.TempDir()
	g := &LinkedGenerator{
		BaseDir:  dir,
		Resolver: scc.NewResolver(scc.NewRegistry(), ".md"),
		LinkMode: scc.LinkAll,
	}
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{"name": "Censor Abilities", "type": "trait"},
		Body:        "structured body only",
		PageBody:    "## Signature Ability\n\n##### Back Blasphemer!\n\nfull page render",
	}
	if err := g.WriteSection("mcdm.heroes.v1/feature.trait.censor.level-1/censor-abilities", parsed); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := os.ReadFile(filepath.Join(dir, "feature/trait/censor/level-1/censor-abilities.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "full page render") {
		t.Error("md-linked should use PageBody when present")
	}
	if strings.Contains(s, "structured body only") {
		t.Error("md-linked should NOT use structured Body when PageBody is present")
	}
}
```

Ensure the test file imports `os`, `path/filepath`, `strings`, `scc`, and `content` (add any missing).

- [x] **Step 2: Run to verify it fails**

Run: `devbox run -- go -C steel-etl test ./internal/output/ -run TestLinkedGenerator_UsesPageBody`
Expected: FAIL (currently writes `Body`).

- [x] **Step 3: Implement — prefer `PageBody`, fall back to `Body`**

In `steel-etl/internal/output/linked.go`, change the body selection in `WriteSection`:

```go
func (g *LinkedGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	// Reading pages use the full book-order subtree render when available,
	// falling back to the structured Body for sections without one.
	bodySource := parsed.PageBody
	if bodySource == "" {
		bodySource = parsed.Body
	}

	// Create a copy with resolved links in the body
	resolved := &content.ParsedContent{
		Frontmatter: parsed.Frontmatter,
		Body:        g.Resolver.ResolveLinks(bodySource, sccCode, g.LinkMode),
		TypePath:    parsed.TypePath,
		ItemID:      parsed.ItemID,
	}

	relPath := SCCToFilePath(sccCode, ".md")
	fullPath := filepath.Join(g.BaseDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	out, err := BuildMarkdownFile(resolved)
	if err != nil {
		return fmt.Errorf("build linked markdown for %s: %w", sccCode, err)
	}

	return os.WriteFile(fullPath, []byte(out), 0644)
}
```

- [x] **Step 4: Run to verify it passes**

Run: `devbox run -- go -C steel-etl test ./internal/output/ -run TestLinkedGenerator -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git -C steel-etl add internal/output/linked.go internal/output/linked_test.go
git -C steel-etl commit -m "feat: md-linked emits PageBody full-subtree render"
```

---

## Task 4: Remove the site composite reassembly

**Files:**
- Modify: `v2/site.yaml`
- Modify: `steel-etl/internal/site/build.go`
- Modify: `steel-etl/internal/site/build_test.go`

- [x] **Step 1: Remove `composites:` from `v2/site.yaml`**

Delete the entire `composites:` block under the `Browse` section (the `base: class` and `base: ancestry` entries, `site.yaml:41-49`). Keep `include:`, `exclude:`, `sort:`, and the `groups:` (kit flatten) blocks. The `exclude: feature/trait/ancestry-traits` line stays (ancestry traits remain non-browsable standalone; they now appear inline on the ancestry page via `PageBody`).

- [x] **Step 2: Verify the config type still parses without composites**

Run: `devbox run -- go -C steel-etl run ./cmd/steel-etl site --config ../v2/site.yaml`
Expected: runs without error (no composites processed). Note: if `CompositeConfig` is a required field this will surface here; it is optional (`len(section.Composites) == 0` is handled at `build.go:500`).

- [x] **Step 3: Remove dead composite code**

In `steel-etl/internal/site/build.go`, remove the now-unused composite machinery: `assembleComposites`, `assembleComposite`, `collectCompositeChildren`, `compositeEntry`, `compositeEmbed`, `levelGroupHeading`, and the call site in `Build` (around `build.go:67-69`). Remove the `Composites` field usage. Keep `rebaseLinks` only if still referenced elsewhere; if it becomes unused, remove it too.

Run: `devbox run -- go -C steel-etl build ./...` after deletion and fix any "declared and not used" / undefined references by removing the corresponding dead code. Do NOT remove section-mapping, `injectH1`, group flattening, search-exclusion, or permalink code.

- [x] **Step 4: Update `build_test.go`**

Remove or rewrite composite-specific tests (any test invoking `assembleComposite*` or asserting "N-Level Features"/"N-Level Abilities" grouping). Keep tests for section mapping, index generation, nav, search exclusion, and permalinks. For each removed test, confirm no remaining test references deleted symbols.

Note: `TestBuild_GeneratesIndexPages` was already failing on `main` before this work (pre-existing, unrelated: "feature index missing ability subdir link"). If it still fails identically after this task, leave it — it is out of scope. If composite removal changes its behavior, adjust only the assertions that referenced composites.

- [x] **Step 5: Run site tests**

Run: `devbox run -- go -C steel-etl test ./internal/site/`
Expected: PASS except the documented pre-existing `TestBuild_GeneratesIndexPages` failure if it persists unchanged.

- [x] **Step 6: Commit**

```bash
git -C steel-etl add internal/site/build.go internal/site/build_test.go
git add v2/site.yaml
git commit -m "refactor: drop composite reassembly; pages come from PageBody section mapping"
```

(Note: `v2/site.yaml` is in the `v2` sub-repo if `v2` is a separate git repo; commit it there. Check `git -C v2 status`. The site builder source is in the `steel-etl` repo. Commit each in its own repo.)

---

## Task 5: End-to-end regeneration and verification

**Files:** none (verification only)

- [x] **Step 1: Run the full pipeline**

Run: `devbox run -- go -C steel-etl run ./cmd/steel-etl gen --config pipeline.yaml`
Expected: completes; reports written files. No new errors vs baseline.

- [x] **Step 2: Build the site**

Run: `devbox run -- go -C steel-etl run ./cmd/steel-etl site --config ../v2/site.yaml`
Expected: completes; reports files copied / index pages.

- [x] **Step 3: Verify the class page has NO duplication and full content**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
grep -c "### Back Blasphemer" v2/docs/Browse/class/censor.md        # expect 1
grep -c "## 1st-Level Abilities" v2/docs/Browse/class/censor.md     # expect 0 (no separate grouped section)
grep -n "^#" v2/docs/Browse/class/censor.md | head -40
```
Expected: each ability appears once, nested under its feature in source order; no "N-Level Abilities" regrouping headings. Verify "Judgment" appears once (was duplicated before).

- [x] **Step 4: Verify the standalone trait page is complete**

Run: `grep -n "^#\|Power Roll" v2/docs/Browse/feature/trait/censor/level-1/censor-abilities.md`
Expected: all signature + heroic abilities present, un-blockquoted, under normalized subheadings.

- [x] **Step 5: Verify a leaf ability page is unchanged**

Run: `cat v2/docs/Browse/feature/ability/censor/level-1/back-blasphemer.md`
Expected: same clean statblock as before this work (frontmatter + un-blockquoted body). Confirms `PageBody == Body` for leaves.

- [x] **Step 6: Verify the chapter page reads like the book**

Run: `grep -n "^#" v2/docs/Read/chapter/classes.md | head -40; wc -l v2/docs/Read/chapter/classes.md`
Expected: the chapter now contains the individual classes (Censor, Fury, …) inline at H2, each with full nested content — substantially larger than the previous 444-line stub.

- [x] **Step 7: Verify ancestry page**

Run: `ls v2/docs/Browse/ancestry/; grep -n "^#" v2/docs/Browse/ancestry/*.md | head`
Expected: ancestry pages include their traits inline; no separate grouped/duplicated trait section; `feature/trait/ancestry-traits` not present as standalone Browse pages.

- [x] **Step 8: Verify flavor blockquotes survive**

Run: `grep -n "We FIGHT" v2/docs/Browse/class/censor.md`
Expected: the "We FIGHT!" quote is still a blockquote (`>` preserved) — confirms only ability sections were un-blockquoted.

- [x] **Step 9: Verify SCC links on the class page resolve**

Run: `grep -n "](\.\./" v2/docs/Browse/class/censor.md | head`
Expected: relative links (e.g. to conditions/skills) resolve relative to the class page location, not broken `scc:` literals.

- [x] **Step 10: Run the full test suite**

Run: `devbox run -- go -C steel-etl test ./...`
Expected: PASS except the documented pre-existing `TestBuild_GeneratesIndexPages` failure if it persists unchanged from `main`.

- [x] **Step 11: Spot-check page count / size sanity**

Run: `find v2/docs/Browse -name '*.md' | wc -l; du -sh v2/docs`
Expected: file count comparable to before; total size larger (pages now contain nested content). No explosion suggesting infinite recursion.

- [x] **Step 12: Commit any regenerated committed artifacts**

If `data/` or `v2/docs/` regenerated files are tracked and intended to be committed, commit them in their respective repos with a message referencing this change. (Generated dirs may be gitignored — check `git status` per repo first; do not force-add ignored output.)

---

## Task 6: Documentation

**Files:**
- Modify: `ARCHITECTURE.md`
- Modify: `steel-etl/CLAUDE.md`
- Modify: `v2/CLAUDE.md`
- Modify: `FOLLOWUPS.md` (if follow-ups remain)

- [x] **Step 1: Update `ARCHITECTURE.md`**

In the `steel-etl site` section, replace the "Composite pages" bullet with a description of the new model: each `md-linked` page is a full book-order render of its source subtree (`PageBody`/`RenderSubtree`); the site builder maps these directly (no composite reassembly). Note that `md`/`json`/`yaml`/`dse` remain per-section structured outputs.

- [x] **Step 2: Update `steel-etl/CLAUDE.md`**

Under "Site builder", remove the "Composite pages" bullet and add a "Book-faithful pages" note: `RenderSubtree` (in `internal/content/render_subtree.go`) produces `ParsedContent.PageBody`, consumed by the `md-linked` generator; ability statblocks are un-blockquoted, headings normalized, document order preserved.

- [x] **Step 3: Update `v2/CLAUDE.md`**

Note that `docs/Browse/*` and `docs/Read/chapter/*` are full-subtree renders generated by `steel-etl`; static overrides still apply last.

- [x] **Step 4: Update `FOLLOWUPS.md`**

Add any deferred refinements surfaced during implementation, e.g.: "Cross-reference links on aggregate pages currently point to standalone pages rather than in-page anchors — consider anchor links as a future enhancement." Remove if not applicable.

- [x] **Step 5: Commit**

```bash
git add ARCHITECTURE.md FOLLOWUPS.md
git -C steel-etl add CLAUDE.md
git -C v2 add CLAUDE.md
git commit -m "docs: document book-faithful page rendering"
git -C steel-etl commit -m "docs: document RenderSubtree/PageBody rendering"
git -C v2 commit -m "docs: note full-subtree generated pages"
```

---

## Self-Review Notes / Open Decisions (defaults chosen)

- **Output format scope:** Only `md-linked` (the site's input) uses `PageBody`. `md`/`json`/`yaml`/`dse`/`stripped`/`aggregate`/`scc-api` keep per-section structured `Body`. Rationale: minimize blast radius to the data repos; the site is the consumer that needs book-faithful pages. If the `data-rules/en/md` repo is later wanted as full-page too, flip the `MarkdownGenerator` the same way as Task 3.
- **Cross-reference links:** Aggregate pages keep linking to standalone pages (existing behavior), not in-page anchors. Recorded as a FOLLOWUP.
- **Standalone per-section pages remain:** Required for SCC permalinks (`/scc/{code}/`), the API, and the `Browse/feature/*` trees. Unchanged.
- **Heading normalization:** page title = H1 (injected by site); a descendant at source depth `d` renders at `H(1+d)`, capped at 6. Verified consistent with prior class-page levels.
- **`RenderSubtree` body fidelity:** the only render-affecting per-parser transform is blockquote-stripping for `ability` sections; `RenderSubtree` replicates it directly rather than invoking parsers. Task 5 Step 5 verifies leaf-page parity; if any other type's standalone rendering diverges from its nested rendering, add a targeted transform in `nodeBody`.
- **Pre-existing failing test:** `TestBuild_GeneratesIndexPages` fails on `main` independently of this work; out of scope.
