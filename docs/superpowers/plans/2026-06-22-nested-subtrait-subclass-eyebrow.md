# Subclass eyebrow on nested sub-trait cards — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show each subclass feature's full eyebrow (`<Class> Feature · <Subclass>`) on nested option cards (class aggregate pages, container leaves, Read chapters), matching the standalone leaf.

**Architecture:** `render_subtree.go` stamps a `{data-subclass="<slug>"}` attr on child headings that carry a subclass annotation. `trait_cards.go` captures it into `traitNode.subclass` and renders the eyebrow on nested children, inheriting the class-context prefix from the container card.

**Tech Stack:** Go (devbox), steel-etl `internal/content` + `internal/site`.

## Global Constraints

- All Go commands run under devbox from the workspace root: `devbox run -- go -C steel-etl <args>` (Go is not on PATH; devbox runs from the workspace dir, so target the module with `-C steel-etl`).
- Eyebrow text on nested children = full leaf form `<prefix> · <Subclass>` (prefix inherited from container; subclass title-cased from its slug). Children without a subclass stay eyebrow-less.
- `data-subclass` is additive and inert until `trait_cards.go` reads it; no CSS change.
- Branch: `feat/nested-subclass-eyebrow` (already created in `steel-etl`).

---

### Task 1: Stamp `data-subclass` on child headings — `internal/content/render_subtree.go`

**Files:**
- Modify: `steel-etl/internal/content/render_subtree.go` (the attrs block, ~lines 54-69)
- Test: `steel-etl/internal/content/render_subtree_test.go`

**Interfaces:**
- Produces: descendant headings rendered by `RenderSubtree` carry `data-subclass="<slug>"` (from `section.Annotation["subclass"]`) in their attr_list, alongside the existing `data-scc` / `data-cost`.

- [ ] **Step 1: Write the failing test**

Add to `steel-etl/internal/content/render_subtree_test.go`:

```go
func TestRenderSubtree_StampsSubclassOnChildHeadings(t *testing.T) {
	container := &parser.Section{
		Heading:      "4th-Level Domain Feature",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "Choose one.",
		Children: []*parser.Section{
			{Heading: "Oracular Warning", HeadingLevel: 5,
				Annotation: map[string]string{"type": "feature", "subclass": "fate"},
				BodySource: "Premonitions help."},
			{Heading: "Plain Child", HeadingLevel: 5,
				Annotation: map[string]string{"type": "feature"},
				BodySource: "No subclass here."},
		},
	}
	codes := map[*parser.Section]string{
		container.Children[0]: "mcdm.heroes.v1/feature.censor.level-4/oracular-warning",
		container.Children[1]: "mcdm.heroes.v1/feature.censor.level-4/plain-child",
	}
	got := RenderSubtree(container, codes)

	if !strings.Contains(got, `data-subclass="fate"`) {
		t.Errorf("child with subclass annotation should stamp data-subclass:\n%s", got)
	}
	// the plain child's heading line must carry data-scc but NOT data-subclass
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "Plain Child") && strings.Contains(line, "data-subclass") {
			t.Errorf("plain child must not get data-subclass:\n%s", line)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- go -C steel-etl test ./internal/content/ -run TestRenderSubtree_StampsSubclassOnChildHeadings -v`
Expected: FAIL (`data-subclass="fate"` not found).

- [ ] **Step 3: Implement the stamp**

In `steel-etl/internal/content/render_subtree.go`, inside the child loop, after the `data-cost` block (around line 69, before `if len(attrs) > 0`), add:

```go
		if sub := strings.TrimSpace(child.Annotation["subclass"]); sub != "" {
			attrs = append(attrs, `data-subclass="`+sub+`"`)
		}
```

(`child.Annotation` is `map[string]string` and is nil-safe for index reads in Go, but the surrounding code already dereferences `child.Type()`/`child.NoClassify()`, so the map access is consistent with existing usage.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- go -C steel-etl test ./internal/content/ -run TestRenderSubtree_StampsSubclassOnChildHeadings -v`
Expected: PASS.

- [ ] **Step 5: Run the package tests (no regressions)**

Run: `devbox run -- go -C steel-etl test ./internal/content/`
Expected: ok.

- [ ] **Step 6: Commit**

```bash
cd steel-etl && git add internal/content/render_subtree.go internal/content/render_subtree_test.go && git commit -m "feat: stamp data-subclass on child headings in RenderSubtree"
```

---

### Task 2: Render the nested eyebrow — `internal/site/trait_cards.go`

**Files:**
- Modify: `steel-etl/internal/site/trait_cards.go`
- Test: `steel-etl/internal/site/trait_cards_test.go`

**Interfaces:**
- Consumes: `data-subclass="<slug>"` attrs from Task 1.
- Produces: `renderTraitNode` emits `<div class="sc-trait__eyebrow">…<prefix> · <Subclass></div>` for nested children that carry a subclass.

- [ ] **Step 1: Write the failing test**

Add to `steel-etl/internal/site/trait_cards_test.go`:

```go
func TestRenderTraitCard_NestedChildShowsSubclassEyebrow(t *testing.T) {
	fm := "name: 4th-Level Domain Feature\ntype: feature\nclass: censor\nscc: mcdm.heroes.v1/feature.censor.level-4/4th-level-domain-feature"
	body := "\nChoose one of the following.\n\n" +
		"### Oracular Warning {data-scc=\"mcdm.heroes.v1/feature.censor.level-4/oracular-warning\" data-subclass=\"fate\"}\n\n" +
		"Premonitions help you stay alive.\n\n" +
		"### Plain Child {data-scc=\"mcdm.heroes.v1/feature.censor.level-4/plain-child\"}\n\n" +
		"No subclass here.\n"
	got := renderTraitCard(fm, body)

	if !strings.Contains(got, `<div class="sc-trait__eyebrow"><span class="sc-trait__dia"></span>Censor Feature · Fate</div>`) {
		t.Errorf("nested subclass child should carry the full eyebrow:\n%s", got)
	}
	// the plain child must remain eyebrow-less: only the container eyebrow ("Censor Feature")
	// plus the Fate child's eyebrow exist — so exactly 2 eyebrows total.
	if n := strings.Count(got, "sc-trait__eyebrow"); n != 2 {
		t.Errorf("expected exactly 2 eyebrows (container + Fate child), got %d:\n%s", n, got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- go -C steel-etl test ./internal/site/ -run TestRenderTraitCard_NestedChildShowsSubclassEyebrow -v`
Expected: FAIL (no nested eyebrow; count is 1).

- [ ] **Step 3: Add the `subclass` field + parse capture**

In `steel-etl/internal/site/trait_cards.go`:

(a) Add the regex next to `traitSCCRe` (after line 31 `traitCostRe`):

```go
	traitSubclassRe = regexp.MustCompile(`data-subclass="([^"]+)"`)
```

(b) Add the field to `traitNode` (after `scc string`, line 61):

```go
	subclass  string
```

(c) In `parseTraitTree`, inside the `if loc[6] >= 0 {` attrs block (after the `traitCostRe` capture, ~line 513), add:

```go
			if m := traitSubclassRe.FindStringSubmatch(attrs); m != nil {
				sub = m[1]
			}
```

and declare `sub` next to `scc := ""` (around line 503):

```go
		scc := ""
		sub := ""
```

and set it on the node literal (in the `flat = append(...&traitNode{...})`, after `scc: scc,`):

```go
			subclass:  sub,
```

- [ ] **Step 4: Factor the eyebrow prefix out of `traitEyebrow`**

Replace `traitEyebrow` (lines 429-448) with a prefix helper + thin wrapper:

```go
// traitEyebrowPrefix is the class/source + feature-noun portion of the eyebrow,
// without the subclass suffix: "<Class> Feature" / "<Ancestry> Trait". Nested
// children inherit this from their container and append their own subclass.
func traitEyebrowPrefix(fm string) string {
	source := ""
	for _, key := range []string{"class", "ancestry", "kit"} {
		if v := strings.TrimSpace(parseFrontmatterField(fm, key)); v != "" {
			source = titleCase(strings.ReplaceAll(v, "-", " "))
			break
		}
	}
	if fs := strings.TrimSpace(parseFrontmatterField(fm, "feature_source")); fs != "" && fs != "summoner" && source != "" {
		source = strings.TrimSpace(source + " " + titleCase(strings.ReplaceAll(fs, "-", " ")))
	}
	return strings.TrimSpace(source + " " + featureNoun(parseFrontmatterField(fm, "type")))
}

// traitEyebrow is the full source context line, with the subclass appended when present.
func traitEyebrow(fm string) string {
	label := traitEyebrowPrefix(fm)
	if sub := strings.TrimSpace(parseFrontmatterField(fm, "subclass")); sub != "" {
		label += " · " + titleCase(strings.ReplaceAll(sub, "-", " "))
	}
	return label
}
```

- [ ] **Step 5: Thread the prefix through the render chain**

In `renderTraitCard` (line 90), compute the prefix and pass it:

```go
	prefix := traitEyebrowPrefix(fm)
	bodyHTML, leadProse := renderTraitBody(intro, children, prefix)
```

Change `renderTraitBody`'s signature (line 170) and its recursive child call (line 213):

```go
func renderTraitBody(intro string, children []*traitNode, prefix string) (body string, leadProse bool) {
```
```go
			} else {
				b.WriteString(renderTraitNode(c, prefix))
			}
```

Replace `renderTraitNode` (lines 105-110) with the eyebrow-aware version:

```go
// renderTraitNode renders a nested sub-trait (no crest, no drop cap). It carries
// the full eyebrow (inherited prefix + its own subclass) when it has a subclass;
// otherwise it stays eyebrow-less, as before. Its level pill is derived from the scc.
func renderTraitNode(n *traitNode, prefix string) string {
	intro, _ := parseTraitTree(n.content) // sub-headings already split into n.children
	bodyHTML, _ := renderTraitBody(intro, n.children, prefix)
	tag := traitTag(n.cost, "", n.scc)
	eyebrow := ""
	if n.subclass != "" {
		sub := titleCase(strings.ReplaceAll(n.subclass, "-", " "))
		if prefix != "" {
			eyebrow = prefix + " · " + sub
		} else {
			eyebrow = sub
		}
	}
	return wrapTraitSection("sc-trait", "", "", eyebrow, strings.TrimSpace(n.name), tag, bodyHTML)
}
```

- [ ] **Step 6: Run the new test to verify it passes**

Run: `devbox run -- go -C steel-etl test ./internal/site/ -run TestRenderTraitCard_NestedChildShowsSubclassEyebrow -v`
Expected: PASS.

- [ ] **Step 7: Run the site package tests (no regressions)**

Run: `devbox run -- go -C steel-etl test ./internal/site/`
Expected: ok. (If a pre-existing test asserted a child had no eyebrow when it now carries a subclass, update it; none is expected since prior fixtures use no `data-subclass`.)

- [ ] **Step 8: Commit**

```bash
cd steel-etl && git add internal/site/trait_cards.go internal/site/trait_cards_test.go && git commit -m "feat: render subclass eyebrow on nested sub-trait cards"
```

---

### Task 3: Integration verify — regenerate and confirm class pages

**Files:** none (verification only)

**Interfaces:**
- Consumes: Tasks 1-2.
- Produces: confidence that class aggregate pages now show the subclass eyebrow, with no regression.

- [ ] **Step 1: Full build + test suite**

Run: `devbox run -- go -C steel-etl build ./... && devbox run -- go -C steel-etl test ./...`
Expected: build ok; all packages PASS.

- [ ] **Step 2: Regenerate heroes output**

Run: `devbox run -- go -C steel-etl run ./cmd/steel-etl gen --config pipeline.yaml`
Expected: completes (`Classified: …, Written: …`).

- [ ] **Step 3: Confirm class aggregate pages now carry subclass eyebrows**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace
for f in censor conduit elementalist; do
  n=$(grep -oE 'sc-trait__eyebrow"><span[^>]*></span>[^<]*·[^<]*</div>' "v2/docs/Browse/class/$f.md" | wc -l)
  echo "$f.md: $n feature-card eyebrows with subclass"
done
grep -oE 'sc-trait__eyebrow"><span[^>]*></span>[^<]*·[^<]*</div>' v2/docs/Browse/class/censor.md | sed 's/<[^>]*>//g' | grep -i fate | head
```
Expected: each class page now reports a non-zero count (was 0); `censor.md` shows "Censor Feature · Fate" on Oracular Warning. (Regenerated `v2/docs/` is build output — do not hand-commit it; the deploy recipe owns it.)

- [ ] **Step 4: Restore generated output (leave deploy to its recipe)**

Run: `cd /home/vexa/code/steel_compendium/workspace && git -C v2 checkout -- docs/ 2>/dev/null; git -C steel-etl status --short`
Expected: only the source changes from Tasks 1-2 are committed; no stray generated files staged. (Deploy regenerates `v2/docs/` fresh.)

---

## Self-Review notes

- **Spec coverage:** stamp `data-subclass` (Task 1); `traitNode.subclass` + parse capture + prefix factoring + threading + nested eyebrow render (Task 2); regenerate + confirm class pages (Task 3) — all spec sections covered.
- **Type consistency:** `traitSubclassRe`, `traitNode.subclass`, `traitEyebrowPrefix(fm) string`, `renderTraitBody(intro string, children []*traitNode, prefix string)`, `renderTraitNode(n *traitNode, prefix string)` are used identically across steps.
- **Prefix fallback:** `renderTraitNode` falls back to subclass-only when `prefix == ""`, so a container lacking class context never yields a dangling "· Fate".
- **No CSS change:** `.sc-trait__eyebrow` already styles both leaf and nested `.sc-trait` sections.
