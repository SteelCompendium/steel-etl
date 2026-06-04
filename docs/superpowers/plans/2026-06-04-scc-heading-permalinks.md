# SCC Heading Permalinks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the big page-title "Copy permalink" button with the native mkdocs heading-anchor (¶) icon, and make each heading's ¶ copy a stable `/scc/<code>/` permalink when that heading maps to an SCC code, or a plain friendly `#anchor` link when it does not.

**Architecture:** The pipeline already classifies every annotated section to an SCC code in the same walk that builds each page's `PageBody` (`internal/pipeline/pipeline.go`). We thread a `map[*parser.Section]string` (section → final SCC code) into `RenderSubtree`, which emits a `{data-scc="<code>"}` attr_list marker on every descendant heading that has a code. Client JS (`scc-headerlinks.js`) reads `data-scc` (and the existing `<meta name="scc-permalink">` for the page H1) to decide, per heading, whether the ¶ copies the stable SCC URL or a default friendly `#anchor` URL. SCC-backed ¶ icons get a distinct accent style. Clicking copies to the clipboard **and** keeps Material's native jump-to-anchor behavior.

**Tech Stack:** Go 1.26 (steel-etl pipeline + tests, run under devbox), MkDocs Material (`attr_list`, `toc: permalink: true`), vanilla JS (`document$` instant-nav hook), CSS.

**Key design decisions (resolved with the user):**
- ¶ click = **copy + jump** (do not `preventDefault`; copy to clipboard, let the native hash navigation proceed).
- SCC-backed permalinks get a **distinct accent style** vs. the muted default ¶.
- Headings with **no** SCC code copy the **friendly page URL + `#anchor`** (useful in-page deep link, explicitly not restructure-stable).
- Roadmap item 3 (in-page anchors) is **not** a dependency: every SCC code already has a page-level redirect stub pointing at that item's own canonical standalone page, so an SCC link copied from an aggregate heading correctly resolves to that item's page.

**Repo layout note:** Go work is in `steel-etl/` (run `go`/`just` via `devbox run --` from the workspace root — Go is not on PATH). Site assets are in `v2/`. Paths below are workspace-relative unless a command `cd`s.

---

### Task 1: Emit `data-scc` markers on subheadings in `RenderSubtree`

Change `RenderSubtree` to accept a `section → SCC code` map and append an `attr_list` `{data-scc="<code>"}` marker to each descendant heading that has a code. A `nil` map preserves current behavior (no markers).

**Files:**
- Modify: `steel-etl/internal/content/render_subtree.go`
- Test: `steel-etl/internal/content/render_subtree_test.go`

- [ ] **Step 1: Write the failing test**

Add this test to `steel-etl/internal/content/render_subtree_test.go` (append at end of file):

```go
func TestRenderSubtree_EmitsDataSCCOnCodedHeadings(t *testing.T) {
	ability := &parser.Section{Heading: "Gouge", HeadingLevel: 3, Annotation: map[string]string{"type": "ability"}, BodySource: "Stab them."}
	structural := &parser.Section{Heading: "Heroic Resource", HeadingLevel: 3, BodySource: "You have Ferocity."}
	class := &parser.Section{
		Heading:      "Fury",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "fury"},
		BodySource:   "You rage.",
		Children:     []*parser.Section{ability, structural},
	}

	scc := map[*parser.Section]string{
		ability: "mcdm.heroes.v1/feature.ability.fury.level-1/gouge",
		// structural intentionally absent: no SCC code
	}

	got := RenderSubtree(class, scc)

	if !strings.Contains(got, `## Gouge {data-scc="mcdm.heroes.v1/feature.ability.fury.level-1/gouge"}`) {
		t.Errorf("coded heading missing data-scc marker:\n%s", got)
	}
	if strings.Contains(got, "Heroic Resource {data-scc") {
		t.Error("structural heading (no code) must not get a data-scc marker")
	}
	if !strings.Contains(got, "## Heroic Resource\n") {
		t.Errorf("structural heading should render as a plain heading:\n%s", got)
	}
}

func TestRenderSubtree_NilMapEmitsNoMarkers(t *testing.T) {
	class := &parser.Section{
		Heading:      "Fury",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "fury"},
		BodySource:   "You rage.",
		Children:     []*parser.Section{{Heading: "Gouge", HeadingLevel: 3, Annotation: map[string]string{"type": "ability"}, BodySource: "Stab."}},
	}
	got := RenderSubtree(class, nil)
	if strings.Contains(got, "data-scc") {
		t.Errorf("nil map must emit no data-scc markers:\n%s", got)
	}
	if !strings.Contains(got, "## Gouge") {
		t.Error("heading should still render with a nil map")
	}
}
```

You must also update the **existing** tests in this file to pass the new second argument (they currently call `RenderSubtree(x)`). Change every existing `RenderSubtree(...)` call to pass `nil`:
- `RenderSubtree(class)` → `RenderSubtree(class, nil)` (in `TestRenderSubtree_NormalizesHeadingsAndOrder`)
- `RenderSubtree(ability)` → `RenderSubtree(ability, nil)` (in `TestRenderSubtree_LeafEqualsOwnBody`)
- `RenderSubtree(root)` → `RenderSubtree(root, nil)` (in `TestRenderSubtree_ClampsShallowChildToH1` and `TestRenderSubtree_CapsDeepNestingAtH6`)
- `RenderSubtree(chapter)` → `RenderSubtree(chapter, nil)` (in `TestRenderSubtree_ChapterPreservesSourceLevels`)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/content/ -run RenderSubtree"`
Expected: FAIL — compile error `not enough arguments in call to RenderSubtree` (signature still takes one arg).

- [ ] **Step 3: Implement the new signature and marker emission**

Replace the body of `steel-etl/internal/content/render_subtree.go` (keep the package + imports) so the two functions read:

```go
// RenderSubtree serializes a section's entire subtree as book-order markdown:
// the section's own immediate body, followed by every descendant (annotated or
// not) inline in document order. Heading levels are normalized so the section
// itself occupies the page's H1 (added separately by the site builder via H1
// injection) and descendants nest by their source depth. Ability statblocks
// (sections with @type: ability), which are blockquoted in source, are
// un-blockquoted to match how standalone ability pages render; genuine flavor
// blockquotes (which are not ability sections) are preserved.
//
// sccBySection maps a descendant section to its final (post-override) SCC code.
// Each descendant heading that has a code gets an attr_list `{data-scc="<code>"}`
// marker so the v2 client can offer a stable /scc/<code>/ permalink on that
// heading's anchor icon. A nil map emits no markers. Headings without a code
// (structural sections) are left plain. attr_list (enabled in v2/mkdocs.yml)
// turns the marker into a data-scc attribute on the rendered <hN> without
// affecting the toc-generated heading id.
//
// scc: links in bodies are left in their raw form; the md-linked generator
// resolves them relative to the page's own SCC code.
func RenderSubtree(section *parser.Section, sccBySection map[*parser.Section]string) string {
	return renderSubtree(section, section.HeadingLevel, sccBySection)
}

func renderSubtree(section *parser.Section, rootLevel int, sccBySection map[*parser.Section]string) string {
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
		if code := sccBySection[child]; code != "" {
			heading += ` {data-scc="` + code + `"}`
		}
		childBody := renderSubtree(child, rootLevel, sccBySection)
		if childBody != "" {
			parts = append(parts, heading+"\n\n"+childBody)
		} else {
			parts = append(parts, heading)
		}
	}

	return strings.Join(parts, "\n\n")
}
```

(`nodeBody` and `stripBlockquotePrefix` are unchanged — leave them as-is below.)

- [ ] **Step 4: Run the tests to verify they pass**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/content/ -run RenderSubtree -v"`
Expected: PASS — all six `RenderSubtree` tests pass, including the two new ones.

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add internal/content/render_subtree.go internal/content/render_subtree_test.go
git commit -m "feat(render): emit data-scc markers on coded subheadings in RenderSubtree"
```

---

### Task 2: Build the section→SCC map and pass it to `RenderSubtree`

The pipeline walk classifies each section as it descends, but renders a parent's `PageBody` (line ~141) **before** its children are classified — so the map isn't complete at render time. Fix by collecting classified sections during the walk (populating the map with final, post-override codes) and deferring `PageBody` rendering + generator writes to a single pass **after** the walk completes, when the map is fully populated.

**Files:**
- Modify: `steel-etl/internal/pipeline/pipeline.go` (the `walk` closure ~lines 104-208)

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/pipeline/pagebody_scc_test.go`:

```go
package pipeline

import (
	"strings"
	"testing"
)

// A coded descendant heading on an aggregate page must carry its own SCC code
// as a data-scc attr_list marker in the rendered PageBody, so the v2 client can
// offer a stable /scc/<code>/ permalink on that heading. Assertions are
// structural (presence + suffix), NOT exact codes — the classifier's TypePath/
// ItemID for a given type are an implementation detail this test must not pin.
func TestPageBody_SubheadingsCarryDataSCC(t *testing.T) {
	out := buildPageBodies(t)

	// Find the page that contains the Gouge ability (keyed by its SCC code,
	// which we deliberately do not hardcode).
	var furyBody string
	for code, body := range out {
		if strings.Contains(code, "fury") && strings.Contains(body, "Gouge") {
			furyBody = body
			break
		}
	}
	if furyBody == "" {
		t.Fatalf("no page containing the Gouge ability found; keys: %v", keysOf(out))
	}
	if !strings.Contains(furyBody, `Gouge {data-scc="`) {
		t.Errorf("Gouge heading missing data-scc marker:\n%s", furyBody)
	}
	if !strings.Contains(furyBody, `/gouge"}`) {
		t.Errorf("Gouge data-scc value should end in /gouge:\n%s", furyBody)
	}
	if strings.Contains(furyBody, `Heroic Resource {data-scc`) {
		t.Error("structural 'Heroic Resource' heading must not carry a data-scc marker")
	}
}
```

Add this **test helper** to the same file. It runs the pipeline over a minimal in-memory document and returns each classified section's resolved `PageBody`, keyed by SCC code. It uses the real `RunPipeline` entry point with an in-memory doc:

```go
// buildPageBodies runs the pipeline over a small synthetic Draw Steel document
// and returns a map of SCC code -> resolved PageBody. It writes to a temp dir.
func buildPageBodies(t *testing.T) map[string]string {
	t.Helper()

	src := strings.Join([]string{
		"---",
		"title: Test",
		"---",
		"",
		"# Classes <!-- @type: chapter @id: classes -->",
		"",
		"How classes work.",
		"",
		"## Fury <!-- @type: class @id: fury -->",
		"",
		"You rage.",
		"",
		"### Heroic Resource",
		"",
		"You have Ferocity.",
		"",
		"### Gouge <!-- @type: ability -->",
		"",
		"Stab them.",
		"",
	}, "\n")

	got, err := runPipelineForTest(t, src)
	if err != nil {
		t.Fatalf("pipeline run: %v", err)
	}
	return got
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
```

> **Note on `runPipelineForTest`:** The pipeline's public entry point and its config/result types are defined in this package (`RunPipeline`, `Config`, `Result`, `ClassifiedItem`). Before writing `runPipelineForTest`, read `steel-etl/internal/pipeline/pipeline.go` (top of file) and any existing `*_test.go` in this package to see how tests already construct a `Config` and invoke the pipeline. Implement `runPipelineForTest(t, src)` to: write `src` to a temp `.md`, build a minimal `Config` pointing input at it with the `md-linked` format enabled and `source: mcdm.heroes.v1`, run the pipeline, then read the generated `md-linked` files and return `map[sccCode]PageBody` by parsing each file's frontmatter `scc` + body. If an existing test in the package already has such a harness, reuse it instead of writing a new one.
>
> **If a pipeline test harness is impractical to build cheaply:** this integration test is best-effort, not a hard gate. The `data-scc` emission logic is already deterministically covered by Task 1's unit test (hand-built map), and the real end-to-end pipeline path is verified by the generated-markdown grep in Task 7 Step 2. In that case, skip this test (delete `pagebody_scc_test.go`) and rely on Task 1 + Task 7 — but **still** do the deferred-render refactor in Step 3 and verify it via the full suite in Step 5.

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/pipeline/ -run TestPageBody_SubheadingsCarryDataSCC -v"`
Expected: FAIL — `data-scc` absent from the fury PageBody (the pipeline does not yet build or pass the map), and/or a compile error until the deferred-render refactor in Step 3 lands.

- [ ] **Step 3: Build the map during the walk; defer render + writes**

In `steel-etl/internal/pipeline/pipeline.go`, locate the section just before the `walk` closure (after `chapterOrder := 0`, ~line 106) and add the map plus a pending-items slice:

```go
	seenSCC := make(map[string]string)
	chapterOrder := 0

	// sccBySection maps each classified section to its final (post-override) SCC
	// code so RenderSubtree can mark coded descendant headings. PageBody render +
	// generator writes are deferred until after the walk so the map is complete
	// (a parent is visited before its children, so its descendants' codes are not
	// yet known at parent-render time).
	sccBySection := make(map[*parser.Section]string)
	type pendingWrite struct {
		section *parser.Section
		parsed  *content.ParsedContent
		sccCode string
	}
	var pending []pendingWrite
```

Then, in the `walk` closure, replace the block that currently runs from `parsed.PageBody = content.RenderSubtree(section)` through the generator-write loop and `result.Classified` append (current lines ~140-177). Specifically:

Delete this line (it must NOT render here anymore):
```go
		// Full book-order render of this section's subtree for reading pages.
		parsed.PageBody = content.RenderSubtree(section)
```

And inside the `if parsed.TypePath != nil && parsed.ItemID != "" {` block, after the override handling sets the final `sccCode`/`parsed.Frontmatter["scc"]` and after the duplicate-detection (`seenSCC[sccCode] = section.Heading`), **replace** the generator-write loop and the `result.Classified` append with map population + a pending append:

Remove this (current ~lines 166-176):
```go
				// Write to all generators
				for _, gen := range generators {
					if err := gen.WriteSection(sccCode, parsed); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("write %s [%s]: %v", sccCode, gen.Format(), err))
					} else {
						result.WrittenFiles++
					}
				}

				// Record for cross-book shared-output generation.
				result.Classified = append(result.Classified, ClassifiedItem{SCCCode: sccCode, Parsed: parsed})
```

Replace with:
```go
				// Record the final code so coded descendant headings can be marked
				// in PageBody, and defer the render + writes until the walk is done.
				sccBySection[section] = sccCode
				pending = append(pending, pendingWrite{section: section, parsed: parsed, sccCode: sccCode})
```

Then, immediately after `walk(doc.Sections)` (current line ~182) and **before** the "Finalize generators" block, add the deferred render + write pass:

```go
	walk(doc.Sections)

	// Now that every section's SCC code is known, render each page's book-order
	// PageBody (marking coded descendant headings with data-scc) and write to all
	// generators. Deferred from the walk so the sccBySection map is complete.
	for _, pw := range pending {
		pw.parsed.PageBody = content.RenderSubtree(pw.section, sccBySection)
		for _, gen := range generators {
			if err := gen.WriteSection(pw.sccCode, pw.parsed); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("write %s [%s]: %v", pw.sccCode, gen.Format(), err))
			} else {
				result.WrittenFiles++
			}
		}
		result.Classified = append(result.Classified, ClassifiedItem{SCCCode: pw.sccCode, Parsed: pw.parsed})
	}
```

> The "Finalize generators that implement BulkGenerator" loop must remain **after** this new pass (bulk generators aggregate per-section writes, so all `WriteSection` calls must complete first). Leave the finalize / frozen-registry-validate / registry-save blocks in their existing order after the new loop.

- [ ] **Step 4: Run the pipeline test to verify it passes**

Run: `devbox run -- bash -c "cd steel-etl && go test ./internal/pipeline/ -run TestPageBody_SubheadingsCarryDataSCC -v"`
Expected: PASS — the Gouge heading carries a `data-scc="…/gouge"` marker and the structural "Heroic Resource" heading has no marker.

- [ ] **Step 5: Run the full Go test suite + vet to confirm no regressions**

Run: `devbox run -- bash -c "cd steel-etl && go build ./... && go test ./... && go vet ./..."`
Expected: PASS — build succeeds, all tests pass (the deferred-write refactor preserves `result.WrittenFiles`, `result.Classified` order, dup detection, and chapter ordering), vet clean.

- [ ] **Step 6: Commit**

```bash
cd steel-etl
git add internal/pipeline/pipeline.go internal/pipeline/pagebody_scc_test.go
git commit -m "feat(pipeline): defer PageBody render and pass section->SCC map for heading markers"
```

---

### Task 3: Expose the SCC base URL to the client via a meta tag

The client needs `config.site_url` to compose `<base>scc/<code>/` for subheadings. The page-title `<meta name="scc-permalink">` (full URL, page-level code) already exists and stays — the new JS uses it for the H1. Add an unconditional `<meta name="scc-base">` for subheading composition.

**Files:**
- Modify: `v2/overrides/main.html` (the `extrahead` block, ~lines 51-60)

- [ ] **Step 1: Add the scc-base meta tag**

In `v2/overrides/main.html`, inside `{% block extrahead %}` (after `{{ super() }}`), add the `scc-base` meta **before** the existing `{% if page and page.meta and page.meta.scc %}` block:

```html
{% block extrahead %}
{{ super() }}
{#-
  SCC base URL for heading-level permalinks. scc-headerlinks.js composes a
  stable /scc/<code>/ link as `${scc-base}${code}/` for any heading that carries
  a data-scc attribute. Emitted on every page (subheadings can be SCC-coded even
  when the page itself has no scc frontmatter field).
-#}
<meta name="scc-base" content="{{ config.site_url }}scc/">
{% if page and page.meta and page.meta.scc %}
```

Leave the existing `scc-permalink` meta block and the font-prefs `<script>` exactly as they are.

- [ ] **Step 2: Verify the meta renders after a build**

This is verified end-to-end in Task 7 (grep built HTML for `name="scc-base"`). No standalone command here — proceed to the commit.

- [ ] **Step 3: Commit**

```bash
cd v2
git add overrides/main.html
git commit -m "feat(v2): expose scc-base meta for heading-level permalinks"
```

---

### Task 4: New client script — wire ¶ heading anchors to copy SCC / default links

Add `scc-headerlinks.js`: on every `document$` render, for each content heading's `.headerlink` (¶) anchor, decide the link to copy (H1 → page `scc-permalink` meta; heading with `data-scc` → `scc-base + code + "/"`; otherwise friendly URL + `#id`), mark SCC-backed anchors with a class, and copy on click without suppressing the native jump.

**Files:**
- Create: `v2/docs/javascripts/scc-headerlinks.js`

- [ ] **Step 1: Create the script**

Create `v2/docs/javascripts/scc-headerlinks.js` with this exact content:

```javascript
/**
 * SCC-aware heading permalinks.
 *
 * Reuses mkdocs-material's native heading-anchor (¶) icon (rendered by
 * `toc: permalink: true`, class `.headerlink`) as the permalink-copy affordance,
 * replacing the old standalone page-title "Copy permalink" button.
 *
 * For each content heading:
 *   - The page H1 copies the page-level SCC permalink from <meta name="scc-permalink">.
 *   - A heading carrying data-scc (emitted by steel-etl RenderSubtree) copies the
 *     stable `${scc-base}${code}/` URL — this resolves via the /scc/<code>/ redirect
 *     stub to that item's canonical page, surviving site restructures.
 *   - Any other (structural) heading copies the friendly page URL + #anchor: a useful
 *     in-page deep link, but NOT restructure-stable.
 *
 * Click copies to the clipboard AND lets Material's native jump-to-anchor proceed
 * (no preventDefault). SCC-backed anchors get `.headerlink--scc` for distinct styling.
 * Uses document$ so it re-runs under instant navigation. Scoped to .md-content and
 * iterates only headings (tens per page), so it adds no per-link DOM-walk cost.
 */
(function () {
  "use strict";

  function metaContent(name) {
    var m = document.querySelector('meta[name="' + name + '"]');
    return m ? m.getAttribute("content") : null;
  }

  function copyText(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    // Fallback for non-secure contexts (e.g. http:// during local preview).
    return new Promise(function (resolve, reject) {
      try {
        var ta = document.createElement("textarea");
        ta.value = text;
        ta.setAttribute("readonly", "");
        ta.style.position = "absolute";
        ta.style.left = "-9999px";
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        document.body.removeChild(ta);
        resolve();
      } catch (e) {
        reject(e);
      }
    });
  }

  function flash(anchor) {
    anchor.classList.add("headerlink--copied");
    setTimeout(function () {
      anchor.classList.remove("headerlink--copied");
    }, 1200);
  }

  // Returns { url, scc } for a heading.
  function linkFor(heading, sccBase) {
    if (heading.matches(".md-content h1")) {
      var pageLink = metaContent("scc-permalink");
      if (pageLink) return { url: pageLink, scc: true };
    }
    var code = heading.getAttribute("data-scc");
    if (code && sccBase) return { url: sccBase + code + "/", scc: true };
    var base = location.origin + location.pathname;
    return { url: heading.id ? base + "#" + heading.id : base, scc: false };
  }

  function wire(heading, sccBase) {
    var anchor = heading.querySelector("a.headerlink");
    if (!anchor || anchor.dataset.sccWired) return;
    anchor.dataset.sccWired = "1";

    var info = linkFor(heading, sccBase);
    if (info.scc) anchor.classList.add("headerlink--scc");
    anchor.setAttribute(
      "title",
      info.scc
        ? "Copy stable permalink (" + info.url + ")"
        : "Copy link to this section"
    );
    anchor.setAttribute("aria-label", anchor.getAttribute("title"));

    anchor.addEventListener("click", function () {
      // Do NOT preventDefault: native jump (hash update + scroll) still happens.
      copyText(info.url).then(
        function () { flash(anchor); },
        function () { /* clipboard blocked; native jump still works */ }
      );
    });
  }

  function render() {
    var sccBase = metaContent("scc-base");
    var headings = document.querySelectorAll(
      ".md-content h1, .md-content h2, .md-content h3, .md-content h4, .md-content h5, .md-content h6"
    );
    headings.forEach(function (h) { wire(h, sccBase); });
  }

  if (typeof document$ !== "undefined") {
    document$.subscribe(render);
  } else {
    document.addEventListener("DOMContentLoaded", render);
  }
})();
```

- [ ] **Step 2: Commit**

```bash
cd v2
git add docs/javascripts/scc-headerlinks.js
git commit -m "feat(v2): SCC-aware heading permalink copy via native anchor icon"
```

---

### Task 5: Style SCC-backed anchors + copied flash; remove old button CSS

**Files:**
- Modify: `v2/docs/stylesheets/extra.css` (remove `.scc-permalink-copy*` rules ~lines 181-220; add `.headerlink--scc` + `.headerlink--copied`)

- [ ] **Step 1: Remove the old button CSS**

In `v2/docs/stylesheets/extra.css`, delete the entire block from the comment `/* SCC permalink copy button (next to the page title) */` through the closing brace of `.scc-permalink-copy__icon { ... }` (the contiguous run of `.scc-permalink-copy`, `.scc-permalink-copy:hover`, `.scc-permalink-copy:focus-visible`, `.scc-permalink-copy--flash`, and `.scc-permalink-copy__icon` rules — current lines ~181-220).

- [ ] **Step 2: Add the new heading-anchor styling**

In its place, add:

```css
/* SCC-backed heading permalinks: distinct accent vs. the muted default ¶.
   Material renders the ¶ as a.headerlink (hover-revealed). scc-headerlinks.js
   adds .headerlink--scc to anchors that copy a stable /scc/<code>/ link, and a
   transient .headerlink--copied on a successful copy. */
.md-typeset .headerlink--scc {
    color: var(--md-accent-fg-color);
}

.md-typeset .headerlink--scc:hover {
    color: var(--md-accent-fg-color);
    opacity: 1;
}

/* Brief "Copied!" confirmation rendered next to the icon. */
.md-typeset .headerlink.headerlink--copied::after {
    content: " Copied!";
    font-size: 0.7em;
    font-weight: 600;
    color: var(--md-accent-fg-color);
    vertical-align: middle;
}
```

- [ ] **Step 3: Confirm no other references to the removed classes remain**

Run: `cd v2 && grep -rn "scc-permalink-copy" docs/ overrides/ mkdocs.yml`
Expected: no matches (only `scc-permalink` the meta name may remain elsewhere — that is a different string; this grep is for the hyphenated `scc-permalink-copy` button class/file).

- [ ] **Step 4: Commit**

```bash
cd v2
git add docs/stylesheets/extra.css
git commit -m "feat(v2): style SCC heading anchors; remove old permalink button CSS"
```

---

### Task 6: Remove the old button script and re-wire mkdocs.yml

**Files:**
- Delete: `v2/docs/javascripts/scc-permalink-copy.js`
- Modify: `v2/mkdocs.yml` (`extra_javascript` list, ~line 108)

- [ ] **Step 1: Delete the old script**

```bash
cd v2 && git rm docs/javascripts/scc-permalink-copy.js
```

- [ ] **Step 2: Swap the script reference in mkdocs.yml**

In `v2/mkdocs.yml`, in the `extra_javascript:` list, replace the line:

```yaml
  - javascripts/scc-permalink-copy.js
```

with:

```yaml
  - javascripts/scc-headerlinks.js
```

(Leave the other `extra_javascript` entries unchanged.)

- [ ] **Step 3: Commit**

```bash
cd v2
git add mkdocs.yml
git commit -m "chore(v2): replace scc-permalink-copy.js with scc-headerlinks.js"
```

---

### Task 7: End-to-end build verification

Regenerate the pipeline output and build the site, then confirm `data-scc` markers reach the generated markdown and the built HTML, the `scc-base` meta renders, and the old button is gone.

**Files:** none (verification only)

- [ ] **Step 1: Regenerate the md-linked output for all books**

Run: `devbox run -- bash -c "cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all"`
Expected: completes with no errors (the `--all` flag regenerates every book; a bare `gen` would only do the primary book — see `steel-etl/CLAUDE.md`).

- [ ] **Step 2: Confirm data-scc markers landed in generated markdown**

Run: `cd /home/vexa/code/steel_compendium/workspace && grep -rl 'data-scc=' data/*/en/md-linked/ | head`
Expected: at least one match (e.g. a class or chapter page). Then spot-check one:
Run: `grep -m3 'data-scc=' "$(grep -rl 'data-scc=' data/*/en/md-linked/ | head -1)"`
Expected: heading lines like `## Gouge {data-scc="mcdm.heroes.v1/feature.ability.fury.level-1/gouge"}` — coded abilities/features marked, structural headings unmarked.

> If `data/*/en/md-linked/` does not match, find the real md-linked output dir: `grep -rn "md-linked" steel-etl/pipeline.yaml` shows the configured `BaseDir`; grep there instead.

- [ ] **Step 3: Build the v2 site**

Run: `devbox run -- bash -c "cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml"`
Then: `devbox run -- bash -c "cd v2 && mkdocs build"`
Expected: both complete without errors. (`mkdocs` is pip-installed and run via `devbox run --`; it is not on the shellenv PATH.)

- [ ] **Step 4: Confirm the attribute survives into built HTML and the meta renders**

Run: `cd /home/vexa/code/steel_compendium/workspace/v2 && grep -rl 'data-scc=' site/ | head`
Expected: matches under `site/` — attr_list converted the marker into a `data-scc` attribute on `<hN>` tags.

Run: `cd /home/vexa/code/steel_compendium/workspace/v2 && grep -rl 'name="scc-base"' site/ | head`
Expected: matches (meta present on built pages).

Run: `cd /home/vexa/code/steel_compendium/workspace/v2 && test -f site/javascripts/scc-headerlinks.js && echo PRESENT; grep -rl 'scc-permalink-copy' site/ || echo "old button gone"`
Expected: `PRESENT` for the new script and `old button gone` (no references to the removed button).

- [ ] **Step 5: Manual smoke test (browser)**

Serve the built site and check a class/chapter page (`Read/chapter/classes/` or a `Browse/class/*` page):
- Hover a coded subheading (e.g. an ability): the ¶ icon shows in the accent color (`.headerlink--scc`), tooltip reads "Copy stable permalink (…/scc/…/)".
- Click it: a "Copied!" flash appears, the page also jumps to the heading, and the clipboard holds the `/scc/<code>/` URL.
- Hover a structural heading (e.g. "Heroic Resource"): default muted ¶, tooltip "Copy link to this section"; click copies the friendly URL + `#anchor`.
- The page H1's ¶ copies the page-level SCC permalink; the old big "Copy permalink" button is gone.

> Note: headless Chromium is unreliable in this environment (see `docs/handoffs/HANDOFF.md` gotchas). Prefer a real browser, or verify via the grep checks in Step 4 plus reading the served HTML. `devbox run -- bash -c "cd v2 && mkdocs serve"` serves locally.

- [ ] **Step 6: Final regression pass**

Run: `devbox run -- bash -c "cd steel-etl && go test ./... && go vet ./..."`
Expected: PASS.

> **Note:** `data/` is gitignored build output (per `reference_gen_all_flag` memory) — do not commit it. The committed deliverables are the steel-etl Go changes (Tasks 1-2) and the v2 asset changes (Tasks 3-6). The actual prod deploy is a separate `just deploy-v2` run, done by the user when ready.

---

## Post-implementation: docs to update

Per the workspace "keeping docs in sync" rule, before considering this done:
- **`v2/.repo-docs/decisions/`** — add a short ADR (e.g. `2026-06-04-scc-heading-permalinks.md`) recording: the ¶ icon now doubles as the permalink-copy affordance; SCC-backed headings copy `/scc/<code>/`, others copy friendly `#anchor`; the page-title button was removed; `RenderSubtree` emits `data-scc`; the `scc-base` meta was added. Cross-link the two existing permalink ADRs.
- **`ROADMAP.md` item 5** — mark `**Status:** done` (do not delete; periodic cleanup prunes). Note that item 5's "gated on roadmap item 3" caveat proved unnecessary (page-level stubs already resolve aggregate-heading SCC links correctly).
- **`steel-etl/CLAUDE.md` / `ARCHITECTURE.md`** — if either documents the `RenderSubtree`/`PageBody` flow, note that headings now carry `data-scc` markers and that `PageBody` render is deferred to a post-walk pass so the section→SCC map is complete.
```
