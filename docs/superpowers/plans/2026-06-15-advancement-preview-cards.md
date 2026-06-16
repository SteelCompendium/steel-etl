# Advancement-Features Preview Cards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The `advancement-features` card on companion / summoner-fixture group-landing index pages should list the features gained and the level each is gained at (e.g. Panther → `L3 Cat and Mouse · L6 Single Bound · L10 Panther Spirit`), instead of a bare "Advancement Features" card.

**Architecture:** Site-only change. The advancement-features leaf is `type: featureblock` with a `features: [{name, level, body}, …]` frontmatter list (already non-lossy). `buildAdvancementPairContent` (`advancement_pairs.go`) renders the adv slot as `card(p.adv, icon, "Advancement Features", name, "")` with an empty inner. Build that inner from the adv leaf's frontmatter `features[]` — a compact one-row-per-feature list (level badge + name), reusing the parse path of `featureblock_page.go` (`fbDoc`/`fbFeature`) and the link-stripping (`linkText`) + escaping conventions of `statblock_preview.go`. Add minimal CSS to `v2/docs/stylesheets/steel-redesign.css`.

**Tech Stack:** Go (`internal/site`), `gopkg.in/yaml.v3`, CSS.

---

### Task 1: Render the advancement feature list inner HTML

**Files:**
- Modify: `steel-etl/internal/site/advancement_pairs.go`
- Test: `steel-etl/internal/site/advancement_pairs_test.go`

The new helper reads the adv leaf's frontmatter, unmarshals the `features:` list via the existing `fbDoc` struct (`featureblock_page.go`), and emits a `<ul class="sc-card__advlist">` with one `<li>` per feature: a `L<level>` badge + the feature name. Names are run through `linkText` (strip any `scc:` anchors) then HTML-escaped. Returns `""` when the file has no features (so the card falls back to its current bare form). The pair builder then passes this inner into the existing `card(...)` call.

- [ ] **Step 1: Write the failing test**

Add to `advancement_pairs_test.go`:

```go
func TestAdvancementCardInner(t *testing.T) {
	dir := t.TempDir()
	fm := "name: Panther Advancement Features\ntype: featureblock\n" +
		"features:\n" +
		"    - name: Cat and Mouse\n      level: 3\n      body: x\n" +
		"    - name: Single Bound\n      level: 6\n      body: y\n" +
		"    - name: Panther Spirit\n      level: 10\n      body: z\n"
	if err := os.WriteFile(filepath.Join(dir, "panther-advancement-features.md"),
		[]byte("---\n"+fm+"\n---\n\nbody"), 0644); err != nil {
		t.Fatal(err)
	}
	inner := advancementCardInner(dir, "panther-advancement-features.md")
	for _, want := range []string{
		`class="sc-card__advlist"`,
		`>L3<`, `>Cat and Mouse<`,
		`>L6<`, `>Single Bound<`,
		`>L10<`, `>Panther Spirit<`,
	} {
		if !strings.Contains(inner, want) {
			t.Errorf("inner missing %q:\n%s", want, inner)
		}
	}
	// Order: L3 before L6 before L10 (document order preserved).
	if strings.Index(inner, ">L3<") > strings.Index(inner, ">L6<") ||
		strings.Index(inner, ">L6<") > strings.Index(inner, ">L10<") {
		t.Errorf("levels out of document order:\n%s", inner)
	}
}

func TestAdvancementCardInner_NoFeatures(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x-advancement-features.md"),
		[]byte("---\nname: X\ntype: featureblock\n---\n\nbody"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := advancementCardInner(dir, "x-advancement-features.md"); got != "" {
		t.Errorf("expected empty inner for a featureless adv leaf, got: %q", got)
	}
}

func TestAdvancementCardInner_StripsLinks(t *testing.T) {
	dir := t.TempDir()
	fm := "name: X\ntype: featureblock\nfeatures:\n" +
		"    - name: \"[Cat and Mouse](../../foo.md)\"\n      level: 3\n      body: x\n"
	if err := os.WriteFile(filepath.Join(dir, "x-advancement-features.md"),
		[]byte("---\n"+fm+"\n---\n\nbody"), 0644); err != nil {
		t.Fatal(err)
	}
	inner := advancementCardInner(dir, "x-advancement-features.md")
	if !strings.Contains(inner, ">Cat and Mouse<") || strings.Contains(inner, "foo.md") {
		t.Errorf("expected link stripped to plain text, got:\n%s", inner)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestAdvancementCardInner -v'`
Expected: FAIL — `undefined: advancementCardInner`.

- [ ] **Step 3: Implement `advancementCardInner`**

Add to `advancement_pairs.go` (it already imports `path/filepath` and `strings`; add `fmt` and `html`):

```go
// advancementCardInner builds the index-card inner HTML for an advancement-features
// leaf: a compact one-row-per-feature list of the level each feature is gained at
// plus its name (e.g. "L3 Cat and Mouse"). Data comes from the leaf's frontmatter
// features[] (the same fbDoc shape featureblock_page.go renders on the full page),
// which survives the leaf's HTML transform — so no cache is needed. Names are
// link-stripped (linkText) then escaped. Returns "" when the leaf has no features,
// so the caller falls back to the bare "Advancement Features" card.
func advancementCardInner(dir, advFile string) string {
	fm, _ := splitFrontmatter(readFile(filepath.Join(dir, advFile)))
	var doc fbDoc
	if err := yaml.Unmarshal([]byte(fm), &doc); err != nil || len(doc.Features) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<ul class="sc-card__advlist">`)
	for _, f := range doc.Features {
		b.WriteString(`<li class="sc-card__advfeat">`)
		if f.Level > 0 {
			fmt.Fprintf(&b, `<span class="sc-card__advlvl">L%d</span>`, f.Level)
		}
		b.WriteString(`<span class="sc-card__advname">` +
			html.EscapeString(linkText(f.Name)) + `</span></li>`)
	}
	b.WriteString("</ul>\n")
	return b.String()
}
```

Add `"fmt"` and `"html"` to the import block:

```go
import (
	"fmt"
	"html"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestAdvancementCardInner -v'`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git -C steel-etl add internal/site/advancement_pairs.go internal/site/advancement_pairs_test.go
git -C steel-etl commit -m "feat(site): build advancement-features card inner from features[]"
```

---

### Task 2: Wire the inner into the pair builder

**Files:**
- Modify: `steel-etl/internal/site/advancement_pairs.go:131-139`
- Test: `steel-etl/internal/site/advancement_pairs_test.go`

- [ ] **Step 1: Extend the existing pair-content test to assert the feature list**

Append to `TestBuildAdvancementPairContent` (after the existing eyebrow assertion, before its closing brace), and give the wolf adv leaf real features. Replace the `write("wolf-advancement-features.md", …)` line with:

```go
	write("wolf-advancement-features.md",
		"name: Wolf Advancement Features\ntype: featureblock\n"+
			"features:\n    - name: Pack Tactics\n      level: 3\n      body: x\n")
```

Then add these assertions at the end of the test:

```go
	// The advancement card now lists its gained features with level badges.
	if !strings.Contains(out, `class="sc-card__advlist"`) ||
		!strings.Contains(out, ">L3<") || !strings.Contains(out, ">Pack Tactics<") {
		t.Errorf("expected advancement feature list on the card:\n%s", out)
	}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildAdvancementPairContent$ -v'`
Expected: FAIL — the `sc-card__advlist` / `Pack Tactics` assertions miss (inner still empty).

- [ ] **Step 3: Pass the inner into the card call**

In `buildAdvancementPairContent`, change the adv-card line (currently `advancement_pairs.go:138`):

```go
			sb.WriteString(card(p.adv, icon, "Advancement Features", name, ""))
```

to:

```go
			sb.WriteString(card(p.adv, icon, "Advancement Features", name, advancementCardInner(dir, p.adv)))
```

- [ ] **Step 4: Run the full site test suite**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ && go vet ./internal/site/'`
Expected: PASS, no vet output. (The companion-preview and fixture tests use adv leaves with no `features:` → empty inner → unchanged, still green.)

- [ ] **Step 5: Commit**

```bash
git -C steel-etl add internal/site/advancement_pairs.go internal/site/advancement_pairs_test.go
git -C steel-etl commit -m "feat(site): list gained features + levels on advancement cards"
```

---

### Task 3: Style the advancement feature list

**Files:**
- Modify: `v2/docs/stylesheets/steel-redesign.css` (after the `.sc-card__sig-name` rule, ~line 294)

CSS-only, no test. Mirror the compact eyebrow/metal tone of `.sc-card__sig*` and the level/usage type sizing from `.sb-prev__feat-*`.

- [ ] **Step 1: Add the styles**

Insert after the `.sc-card__sig-name { … }` line:

```css
/* Advancement-features index card: gained features + the level each is gained at */
.sc-card__advlist { list-style: none; margin: .8rem 0 0; padding: .7rem 0 0;
  border-top: 1px solid var(--fx-metal-faint); display: flex; flex-direction: column; gap: .35rem; }
.sc-card__advfeat { display: flex; align-items: baseline; gap: .5rem; position: relative; z-index: 1; }
.sc-card__advlvl { flex: 0 0 auto; min-width: 2.1rem; font-family: var(--md-small-header-font);
  font-variant: small-caps; letter-spacing: .04em; font-size: .72rem; color: var(--fx-metal); }
.sc-card__advname { font-size: .9rem; color: var(--md-default-fg-color); }
```

- [ ] **Step 2: Verify the CSS is well-formed (build the v2 site)**

Run: `cd /home/vexa/code/steel_compendium/workspace && devbox run -- mkdocs build -f v2/mkdocs.yml 2>&1 | tail -5`
Expected: build completes without CSS-related errors (warnings about nav are pre-existing/unrelated).

- [ ] **Step 3: Commit**

```bash
git -C v2 add docs/stylesheets/steel-redesign.css
git -C v2 commit -m "feat(v2): style advancement-features index card list"
```

---

### Task 4: Regenerate, visually verify, deploy

**Files:** none (generated output + submodule bump)

- [ ] **Step 1: Regenerate and screenshot the Panther/companion landing**

From workspace root:

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all && go run ./cmd/steel-etl site --config ../v2/site.yaml'
devbox run -- mkdocs build -f v2/mkdocs.yml
/opt/brave.com/brave/brave --headless --no-sandbox --screenshot=/tmp/adv-cards.png \
  "file://$(pwd)/v2/site/Browse/monster/companion/beastheart/index.html"
```

Read `/tmp/adv-cards.png` and confirm the advancement cards now show `L3 …`, `L6 …`, `L10 …` rows.

- [ ] **Step 2: Bump the steel-etl submodule pointer in the workspace**

Only after `steel-etl` `main` carries Tasks 1–2 (fetch/rebase `origin/main` first if it advanced):

```bash
cd /home/vexa/code/steel_compendium/workspace
git fetch origin && git -C steel-etl log --oneline -1
git add steel-etl && git commit -m "chore: bump steel-etl to <sha> (advancement card features)"
```

- [ ] **Step 3: Deploy v2 (regenerates content + pushes)**

```bash
just deploy-v2
```

Then push `steel-etl` and `workspace` `main` (deploy-v2 does not push them):

```bash
git -C steel-etl push origin main
git push origin main
```

- [ ] **Step 4: Update docs**

- Update `project_statblock_preview_cards` memory: advancement cards now list gained features + levels (ROADMAP "advancement-features preview cards" task done).
- Refresh `docs/handoffs/HANDOFF.md` "Next up" to the next ROADMAP item.
- If this was a tracked ROADMAP item, move it to the archive with its `(was #N)` handle.

---

## Self-Review

- **Spec coverage:** the handoff's "Next up" asks for feature names + levels on the advancement card, shared by companions AND summoner fixtures, site-only, no SCC/schema change. Task 1 builds the list, Task 2 wires it in via the shared `buildAdvancementPairContent` (covers both kinds), Task 3 styles it, Task 4 ships. ✓
- **Placeholder scan:** `<sha>` in Task 4 is a runtime value (the actual commit hash), not a code placeholder. No TBD/TODO steps. ✓
- **Type consistency:** `advancementCardInner(dir, advFile string) string` is defined in Task 1 and called identically in Task 2. Reuses existing `fbDoc`/`fbFeature` (`.Features`, `.Name`, `.Level`), `splitFrontmatter`, `readFile`, `linkText` — all confirmed to exist. CSS classes `sc-card__advlist/advfeat/advlvl/advname` are introduced in Task 1's HTML and styled in Task 3. ✓
