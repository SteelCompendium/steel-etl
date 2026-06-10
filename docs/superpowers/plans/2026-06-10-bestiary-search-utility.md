# Bestiary Search & Filter Utility (Part B) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the placeholder Bestiary tab into a client-side faceted **Search & Filter** finder over every statblock / dynamic-terrain / retainer, so Directors can answer queries like "undead minions in the EV 3–6 range."

**Architecture:** Reuse the existing **`SCBrowse` pattern** (the `feature/` landing's `.sc-browse-mount` JSON data island, mounted by `steel-feature-browser.js`). A build-time step (`steel-etl site`) walks the Browse `monster/` `dynamic-terrain/` `retainer/` pages, extracts their frontmatter into a JSON data island, and writes the Bestiary landing (`docs/Bestiary/index.md`). A **sibling JS component** `steel-bestiary-browser.js` (`window.SCBestiary`) mounts that island into a search box + facet chips + numeric range filters (Level, EV) + a **dense sortable table**. No backend; no SCC re-mint; no data-repo change (all from existing frontmatter).

**Tech Stack:** Go (site builder, std `testing`); vanilla JS (mirrors `steel-feature-browser.js`); CSS reusing the `.sc-browse` shell in `steel-indexes.css` + a small new `steel-bestiary.css`; MkDocs Material (`extra_javascript` / `extra_css`).

**Reference spec:** `steel-etl/docs/superpowers/specs/2026-06-10-bestiary-restructure-and-search-design.md` (Part B). **Depends on Part A** (`…/plans/2026-06-10-bestiary-restructure.md`) being merged — the monster/terrain/retainer pages must already live under `docs/Browse/`.

---

## Conventions for every task

- Go runs ONLY via devbox; bare `devbox run -- go` FAILS. Use:
  ```bash
  devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run <TestName> -v'
  ```
- Go tests live in `steel-etl/internal/site/`. Reuse package helpers: `parseFrontmatterField`, `parseFrontmatterList`, `splitFrontmatter`, `stripMD`, `dirURL`, `readFile`, `pathHasSegment`.
- JS lives in `v2/docs/javascripts/`; CSS in `v2/docs/stylesheets/`; both wired in `v2/mkdocs.yml`. There is **no JS unit-test harness** (no `package.json`); JS is verified by `mkdocs build` + manual `mkdocs serve` (Task 6).
- Working off `main` (authorized). Commit after each task with the message shown. No co-author trailers.
- Visual/CSS polish is a later Claude Design pass — this plan delivers a **functional** utility with baseline styling.

---

## File Structure

| File | Responsibility | Change |
|------|----------------|--------|
| `steel-etl/internal/site/bestiary_search.go` | NEW. `bestiaryItem` struct, `collectBestiaryItems(browseDir)` (walks Browse trees → records), `buildBestiarySearchPage(docsDir)` (writes the island landing). | create |
| `steel-etl/internal/site/bestiary_search_test.go` | NEW. Unit tests for collection + page emission. | create |
| `steel-etl/internal/site/build.go` | Call `buildBestiarySearchPage` in `Build()` after `generateIndexPages`. | modify |
| `v2/docs/javascripts/steel-bestiary-browser.js` | NEW. `window.SCBestiary` — mounts `.sc-bestiary-mount`: facets, Level/EV range filters, sortable results table. | create |
| `v2/docs/stylesheets/steel-bestiary.css` | NEW. Results-table + range-input styling (reuses `.sc-browse` shell). | create |
| `v2/mkdocs.yml` | Wire the new JS + CSS. | modify |
| `v2/static_content/docs/Bestiary/index.md` | DELETE the Part-A placeholder (the generated landing supersedes it). | delete |
| `ARCHITECTURE.md`, `steel-etl/CLAUDE.md` | Document the search build step + JS component. | modify |

---

## Task 1: `bestiaryItem` + `collectBestiaryItems`

**Files:**
- Create: `steel-etl/internal/site/bestiary_search.go`
- Test: `steel-etl/internal/site/bestiary_search_test.go`

Walk `docs/Browse/{monster,dynamic-terrain,retainer}` and extract each searchable leaf page's frontmatter. **The `statblock/` folder was hoisted away in a Part A follow-up (`hoistStatblockPath` in `build.go`), so classify by frontmatter `type` + tree, not by a `statblock/` path segment:** `type: statblock` under `monster/` → `statblock`; `type: statblock` under `retainer/` → `retainer`; `type: dynamic-terrain` → `terrain`. Group lore (`type: monster`), Malice/Tactical-Stance featureblocks (`type: featureblock`), and `index.md`/`_Index.md` are excluded. `href` points cross-tab from `Bestiary/` to the Browse page (no `statblock/` segment).

- [ ] **Step 1: Write the failing test** — create `bestiary_search_test.go`:

```go
package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBrowseMD(t *testing.T, path, fm string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\n"+fm+"---\n\n# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCollectBestiaryItems(t *testing.T) {
	browse := filepath.Join(t.TempDir(), "Browse")
	// a monster statblock (hoisted: sits directly under the group, no statblock/)
	writeBrowseMD(t, filepath.Join(browse, "monster", "goblins", "goblin-warrior.md"),
		"ev: \"3\"\nkeywords:\n    - Goblin\n    - Humanoid\nlevel: 1\nname: Goblin Warrior\norganization: Horde\nrole: Harrier\nsize: 1S\ntype: statblock\n")
	// a malice featureblock (must be EXCLUDED — type: featureblock)
	writeBrowseMD(t, filepath.Join(browse, "monster", "goblins", "goblin-malice.md"),
		"name: Goblin Malice\ntype: featureblock\n")
	// a dynamic terrain leaf
	writeBrowseMD(t, filepath.Join(browse, "dynamic-terrain", "mechanisms", "pillar.md"),
		"ev: \"3\"\nlevel: \"2\"\nname: Pillar\nsize: One square\ntype: dynamic-terrain\n")
	// a retainer (also type: statblock, but under retainer/ → classified as retainer)
	writeBrowseMD(t, filepath.Join(browse, "retainer", "angulotl-hopper.md"),
		"ev: '-'\nkeywords:\n    - Angulotl\nlevel: 1\nname: Angulotl Hopper\nrole: Harrier\nsize: 1S\ntype: statblock\n")

	items := collectBestiaryItems(browse)
	if len(items) != 3 {
		t.Fatalf("expected 3 searchable items (malice excluded), got %d: %+v", len(items), items)
	}
	byName := map[string]bestiaryItem{}
	for _, it := range items {
		byName[it.Name] = it
	}
	gw, ok := byName["Goblin Warrior"]
	if !ok {
		t.Fatal("Goblin Warrior missing")
	}
	if gw.Type != "statblock" || gw.Level != 1 || gw.EV != "3" || gw.Role != "Harrier" ||
		gw.Organization != "Horde" || gw.Size != "1S" {
		t.Errorf("Goblin Warrior fields wrong: %+v", gw)
	}
	if len(gw.Keywords) != 2 || gw.Keywords[0] != "Goblin" {
		t.Errorf("Goblin Warrior keywords wrong: %v", gw.Keywords)
	}
	if gw.Href != "../Browse/monster/goblins/goblin-warrior/" {
		t.Errorf("Goblin Warrior href wrong: %q", gw.Href)
	}
	if byName["Pillar"].Type != "terrain" {
		t.Errorf("Pillar type wrong: %q", byName["Pillar"].Type)
	}
	if byName["Angulotl Hopper"].Type != "retainer" || byName["Angulotl Hopper"].EV != "-" {
		t.Errorf("retainer wrong: %+v", byName["Angulotl Hopper"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestCollectBestiaryItems -v'`
Expected: FAIL — `undefined: collectBestiaryItems` / `bestiaryItem`.

- [ ] **Step 3: Implement** — create `bestiary_search.go`:

```go
package site

// Bestiary Search & Filter data island (Plan B). Walks the Browse monster /
// dynamic-terrain / retainer pages and emits one JSON record per searchable
// entity into a .sc-bestiary-mount island on the Bestiary landing, mounted
// client-side by v2/docs/javascripts/steel-bestiary-browser.js (window.SCBestiary).
// SITE-ONLY: all data is read from existing frontmatter — no data-repo change.
// See docs/superpowers/specs/2026-06-10-bestiary-restructure-and-search-design.md.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// bestiaryItem is one searchable record. JSON keys are consumed by
// steel-bestiary-browser.js — keep them in sync with that file.
type bestiaryItem struct {
	Type         string   `json:"type"` // statblock | terrain | retainer
	Name         string   `json:"name"`
	Level        int      `json:"level"`
	EV           string   `json:"ev"` // string: may be "-" (no EV); JS parses for range
	Role         string   `json:"role,omitempty"`
	Organization string   `json:"organization,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
	Size         string   `json:"size,omitempty"`
	Href         string   `json:"href"`
}

// bestiaryItemType classifies a Browse page by its frontmatter `type` + its tree
// (the statblock/ folder was hoisted away, so the path no longer carries a
// statblock segment). Returns "" for non-searchable pages (group lore, Malice
// featureblocks, indexes).
func bestiaryItemType(relSlash, fmType string) string {
	base := relSlash[strings.LastIndexByte(relSlash, '/')+1:]
	if base == "index.md" || base == "_Index.md" {
		return ""
	}
	switch {
	case fmType == "statblock" && strings.HasPrefix(relSlash, "retainer/"):
		return "retainer"
	case fmType == "statblock" && strings.HasPrefix(relSlash, "monster/"):
		return "statblock"
	case fmType == "dynamic-terrain":
		return "terrain"
	default: // featureblock, monster (group lore), anything else
		return ""
	}
}

// collectBestiaryItems walks browseDir (docs/Browse) and returns one record per
// searchable monster-statblock / terrain / retainer leaf, name-sorted by the
// caller's marshal order (stable: file walk is lexical).
func collectBestiaryItems(browseDir string) []bestiaryItem {
	var items []bestiaryItem
	_ = filepath.Walk(browseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(browseDir, path)
		relSlash := filepath.ToSlash(rel)
		fm, _ := splitFrontmatter(readFile(path))
		kind := bestiaryItemType(relSlash, strings.TrimSpace(parseFrontmatterField(fm, "type")))
		if kind == "" {
			return nil
		}
		lvl, _ := strconv.Atoi(unquote(parseFrontmatterField(fm, "level")))
		var kw []string
		for _, k := range parseFrontmatterList(fm, "keywords") {
			kw = append(kw, stripMD(k))
		}
		items = append(items, bestiaryItem{
			Type:         kind,
			Name:         stripMD(parseFrontmatterField(fm, "name")),
			Level:        lvl,
			EV:           unquote(parseFrontmatterField(fm, "ev")),
			Role:         stripMD(parseFrontmatterField(fm, "role")),
			Organization: stripMD(parseFrontmatterField(fm, "organization")),
			Keywords:     kw,
			Size:         stripMD(parseFrontmatterField(fm, "size")),
			Href:         "../Browse/" + dirURL(relSlash),
		})
		return nil
	})
	return items
}

// unquote strips a single layer of surrounding double/single quotes and trims
// (frontmatter scalars like `ev: "3"` or `ev: '-'` keep their quotes through
// parseFrontmatterField).
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// (json import is used by buildBestiarySearchPage in Task 2.)
var _ = json.Marshal
```

> NOTE: the `var _ = json.Marshal` line keeps the `encoding/json` import compiling for Task 1 alone; **delete that line in Task 2** when `buildBestiarySearchPage` (which uses `json.Marshal`) is added. If `unquote` collides with an existing helper, reuse the existing one.

- [ ] **Step 4: Run test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestCollectBestiaryItems -v'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add internal/site/bestiary_search.go internal/site/bestiary_search_test.go
git commit -m "feat(site): collect bestiary search items from Browse frontmatter"
```

---

## Task 2: `buildBestiarySearchPage` + `Build()` hook

**Files:**
- Modify: `steel-etl/internal/site/bestiary_search.go`
- Modify: `steel-etl/internal/site/build.go` (`Build`, after `generateIndexPages` ~line 91)
- Test: `steel-etl/internal/site/bestiary_search_test.go`

Emit `docs/Bestiary/index.md`: search-excluded frontmatter + a short intro + the `.sc-bestiary-mount` island holding the JSON array (inner `<script class="sc-browse-data">`, matching the SCBrowse mount's selector + its `navigation.instant` fallback). Returns `false` (no-op) when there are no items (Monsters book absent).

- [ ] **Step 1: Append the failing test to `bestiary_search_test.go`:**

```go
func TestBuildBestiarySearchPage(t *testing.T) {
	docs := t.TempDir()
	writeBrowseMD(t, filepath.Join(docs, "Browse", "monster", "goblins", "goblin-warrior.md"),
		"ev: \"3\"\nlevel: 1\nname: Goblin Warrior\norganization: Horde\nrole: Harrier\nsize: 1S\ntype: statblock\n")

	ok, err := buildBestiarySearchPage(docs)
	if err != nil || !ok {
		t.Fatalf("expected page written, ok=%v err=%v", ok, err)
	}
	out, err := os.ReadFile(filepath.Join(docs, "Bestiary", "index.md"))
	if err != nil {
		t.Fatalf("Bestiary/index.md not written: %v", err)
	}
	s := string(out)
	for _, want := range []string{
		"search:\n  exclude: true",
		`<div class="sc-bestiary-mount">`,
		`<script type="application/json" class="sc-browse-data">`,
		`"name":"Goblin Warrior"`,
		`"href":"../Browse/monster/goblins/goblin-warrior/"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("search page missing %q in:\n%s", want, s)
		}
	}
}

func TestBuildBestiarySearchPage_NoItems(t *testing.T) {
	docs := t.TempDir()
	if err := os.MkdirAll(filepath.Join(docs, "Browse"), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, err := buildBestiarySearchPage(docs)
	if err != nil || ok {
		t.Errorf("expected no-op (ok=false) with no items, got ok=%v err=%v", ok, err)
	}
	if _, err := os.Stat(filepath.Join(docs, "Bestiary", "index.md")); !os.IsNotExist(err) {
		t.Error("no page should be written when there are no items")
	}
}
```

- [ ] **Step 2: Run to verify it fails** — `undefined: buildBestiarySearchPage`.

- [ ] **Step 3: Implement in `bestiary_search.go`.** Delete the `var _ = json.Marshal` placeholder line from Task 1 and add:

```go
// buildBestiarySearchPage writes docs/Bestiary/index.md: the Search & Filter
// landing carrying a .sc-bestiary-mount JSON data island over every Browse
// statblock / terrain / retainer. Returns false (no write) when there are no
// items, so a build without the Monsters book leaves no empty tab.
func buildBestiarySearchPage(docsDir string) (bool, error) {
	items := collectBestiaryItems(filepath.Join(docsDir, "Browse"))
	if len(items) == 0 {
		return false, nil
	}
	data, err := json.Marshal(items) // default escapes <,>,& → safe inside <script>
	if err != nil {
		return false, err
	}
	var sb strings.Builder
	sb.WriteString("---\nsearch:\n  exclude: true\n---\n\n")
	sb.WriteString("# Bestiary — Search & Filter\n\n")
	sb.WriteString("Find statblocks, dynamic terrain, and retainers across every sourcebook. " +
		"Search by name, filter by type, role, organization, size, or keyword, and narrow by " +
		"**Level** and **EV** range — then jump straight to the page you need.\n\n")
	sb.WriteString("<div class=\"sc-bestiary-mount\">\n")
	sb.WriteString("<script type=\"application/json\" class=\"sc-browse-data\">\n")
	sb.Write(data)
	sb.WriteString("\n</script>\n</div>\n")

	dir := filepath.Join(docsDir, "Bestiary")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(sb.String()), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
```

- [ ] **Step 4: Hook into `Build()` in `build.go`.** After the `generateIndexPages` block (the lines setting `result.IndexPages = indexCount` … `result.Errors = append(result.Errors, indexErrs...)`, ~line 91) and before "Apply search exclusion", insert:

```go
	// Bestiary Search & Filter landing (Plan B): emit the faceted-finder data
	// island over the Browse monster/terrain/retainer pages. No-op when the
	// Monsters book isn't present in this build.
	if ok, err := buildBestiarySearchPage(cfg.DocsDir); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("bestiary search: %v", err))
	} else if ok {
		result.IndexPages++
	}
```

- [ ] **Step 5: Run the new tests + full package:**
```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildBestiarySearchPage -v'
devbox run -- bash -c 'cd steel-etl && go test ./internal/site/'
```
Expected: both new tests PASS; full package PASS.

- [ ] **Step 6: Commit**
```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add internal/site/bestiary_search.go internal/site/bestiary_search_test.go internal/site/build.go
git commit -m "feat(site): emit Bestiary search landing data island in Build()"
```

---

## Task 3: Remove the placeholder; full build emits the search landing

**Files:**
- Delete: `v2/static_content/docs/Bestiary/index.md`

The Part-A placeholder must go, or `copyStaticContent` (which runs after the emitter) would overwrite the generated search landing.

- [ ] **Step 1: Delete the placeholder**
```bash
cd /home/scott/code/steelCompendium/workspace/v2
git rm static_content/docs/Bestiary/index.md
```

- [ ] **Step 2: Rebuild and assert** (gen already ran for Part A; just rebuild the site, or run both to be safe):
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all'
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml'
cd v2/docs
grep -q 'class="sc-bestiary-mount"' Bestiary/index.md && echo "ISLAND OK"
grep -q 'sc-browse-data' Bestiary/index.md && echo "DATA OK"
grep -c '"type":"statblock"' Bestiary/index.md   # should be a few hundred
grep -q '../Browse/monster/' Bestiary/index.md && echo "HREFS OK"
```
Expected: ISLAND OK, DATA OK, a positive statblock count, HREFS OK. (No commit needed for generated `docs/` — it's rebuilt each deploy.)

- [ ] **Step 3: Commit the placeholder removal**
```bash
cd /home/scott/code/steelCompendium/workspace/v2
git commit -m "feat(v2): retire Bestiary placeholder; generated search landing takes over"
```

---

## Task 4: `steel-bestiary-browser.js` (the `SCBestiary` component)

**Files:**
- Create: `v2/docs/javascripts/steel-bestiary-browser.js`
- Modify: `v2/mkdocs.yml` (`extra_javascript`)

Mirror `steel-feature-browser.js`'s `mount()` structure (island lookup with the `navigation.instant` attribute-strip fallback; chip facets; `document$` auto-mount). Differences: **numeric range filters** for Level and EV, and a **sortable results table** instead of preview cards.

- [ ] **Step 1: Create the component** — `v2/docs/javascripts/steel-bestiary-browser.js`:

```javascript
/* ============================================================
   Steel Compendium — steel-bestiary-browser.js
   Client-side SEARCH · FILTER · SORT for the Bestiary tab.
   Sibling of steel-feature-browser.js (SCBrowse): reuses the .sc-browse shell
   (steel-indexes.css) but with statblock facets, Level/EV numeric range filters,
   and a dense sortable results table (steel-bestiary.css).

   Item shape (emitted by steel-etl bestiary_search.go):
     { type:"statblock"|"terrain"|"retainer", name, level, ev,
       role?, organization?, keywords?[], size?, href }
   ============================================================ */
(function () {
  "use strict";

  function esc(s) {
    return String(s == null ? "" : s).replace(/[&<>"]/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c];
    });
  }
  function cap(s) { return String(s).charAt(0).toUpperCase() + String(s).slice(1); }
  function searchSvg() {
    return '<svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/></svg>';
  }
  function evNum(it) { var n = parseFloat(it.ev); return isNaN(n) ? null : n; }

  // Result-table columns. num:true → numeric sort; sortable:false → no header click.
  var COLS = [
    { key: "name", label: "Name", sortable: true },
    { key: "type", label: "Type", sortable: true },
    { key: "level", label: "Level", sortable: true, num: true },
    { key: "ev", label: "EV", sortable: true, num: true },
    { key: "role", label: "Role", sortable: true },
    { key: "organization", label: "Org", sortable: true },
    { key: "size", label: "Size", sortable: true },
    { key: "keywords", label: "Keywords", sortable: false }
  ];

  function uniqueSorted(items, key) {
    var seen = {};
    items.forEach(function (it) {
      var v = it[key];
      if (v == null || v === "") return;
      (Array.isArray(v) ? v : [v]).forEach(function (x) { if (x !== "") seen[x] = true; });
    });
    return Object.keys(seen).sort(function (a, b) { return a.localeCompare(b); });
  }

  function mount(root) {
    // navigation.instant recreates inline <script> elements but strips their
    // attributes (class+type), so after a client nav the island is a bare
    // <script>. Try the precise selector, then fall back to the only <script>.
    var island = root.querySelector("script.sc-browse-data") || root.querySelector("script");
    if (!island) return;
    var items;
    try { items = JSON.parse(island.textContent); } catch (e) { return; }

    var facets = [
      { key: "type", label: "Type", values: uniqueSorted(items, "type"), display: cap },
      { key: "role", label: "Role", values: uniqueSorted(items, "role") },
      { key: "organization", label: "Organization", values: uniqueSorted(items, "organization") },
      { key: "size", label: "Size", values: uniqueSorted(items, "size") },
      { key: "keywords", label: "Keyword", values: uniqueSorted(items, "keywords") }
    ].filter(function (f) { return f.values.length > 1; });

    var state = { q: "", sort: "name", dir: 1, sel: {}, lvlMin: null, lvlMax: null, evMin: null, evMax: null };
    facets.forEach(function (f) { state.sel[f.key] = {}; });

    function rangeInputs(label, key) {
      return '<div class="sc-browse__range"><span class="lbl">' + label + '</span>' +
        '<input type="number" inputmode="numeric" class="sc-range__min" data-range="' + key + '" placeholder="min" aria-label="' + label + ' min">' +
        '<span class="sc-range__dash">–</span>' +
        '<input type="number" inputmode="numeric" class="sc-range__max" data-range="' + key + '" placeholder="max" aria-label="' + label + ' max"></div>';
    }

    root.innerHTML =
      '<div class="sc-browse sc-bestiary-browse">' +
        '<div class="sc-browse__bar">' +
          '<div class="sc-browse__search">' + searchSvg() +
            '<input type="search" placeholder="Search by name, role, or keyword…" aria-label="Search bestiary"></div>' +
        '</div>' +
        '<div class="sc-browse__ranges">' + rangeInputs("Level", "lvl") + rangeInputs("EV", "ev") + '</div>' +
        '<div class="sc-browse__facets">' + facets.map(facetRow).join("") + '</div>' +
        '<div class="sc-browse__head">' +
          '<span class="sc-browse__count"></span>' +
          '<button class="sc-browse__clear" hidden>Clear filters</button>' +
        '</div>' +
        '<div class="sc-browse__results"></div>' +
      '</div>';

    var elSearch = root.querySelector(".sc-browse__search input");
    var elCount = root.querySelector(".sc-browse__count");
    var elClear = root.querySelector(".sc-browse__clear");
    var elResults = root.querySelector(".sc-browse__results");

    elSearch.addEventListener("input", function () { state.q = this.value.trim().toLowerCase(); render(); });

    root.querySelectorAll("input[data-range]").forEach(function (inp) {
      inp.addEventListener("input", function () {
        var v = this.value === "" ? null : parseFloat(this.value);
        var isMin = this.classList.contains("sc-range__min");
        var key = this.dataset.range; // "lvl" | "ev"
        state[key + (isMin ? "Min" : "Max")] = (v == null || isNaN(v)) ? null : v;
        render();
      });
    });

    elClear.addEventListener("click", function () {
      state.q = ""; elSearch.value = "";
      state.lvlMin = state.lvlMax = state.evMin = state.evMax = null;
      root.querySelectorAll("input[data-range]").forEach(function (i) { i.value = ""; });
      facets.forEach(function (f) { state.sel[f.key] = {}; });
      root.querySelectorAll(".sc-chip.is-on").forEach(function (c) {
        c.classList.remove("is-on"); c.setAttribute("aria-pressed", "false");
      });
      render();
    });

    root.querySelectorAll(".sc-chip").forEach(function (chip) {
      chip.addEventListener("click", function () {
        var k = chip.dataset.facet, v = chip.dataset.value;
        if (state.sel[k][v]) { delete state.sel[k][v]; chip.classList.remove("is-on"); chip.setAttribute("aria-pressed", "false"); }
        else { state.sel[k][v] = true; chip.classList.add("is-on"); chip.setAttribute("aria-pressed", "true"); }
        render();
      });
    });

    function matches(it) {
      if (state.q) {
        var hay = (it.name + " " + (it.role || "") + " " + (it.organization || "") + " " + (it.keywords || []).join(" ")).toLowerCase();
        if (hay.indexOf(state.q) === -1) return false;
      }
      if (state.lvlMin != null && it.level < state.lvlMin) return false;
      if (state.lvlMax != null && it.level > state.lvlMax) return false;
      if (state.evMin != null || state.evMax != null) {
        var ev = evNum(it);
        if (ev == null) return false; // EV "-" excluded once an EV bound is set
        if (state.evMin != null && ev < state.evMin) return false;
        if (state.evMax != null && ev > state.evMax) return false;
      }
      for (var k in state.sel) {
        var picks = Object.keys(state.sel[k]);
        if (!picks.length) continue;
        var v = it[k];
        var has = Array.isArray(v) ? v.some(function (x) { return state.sel[k][x]; }) : state.sel[k][String(v)];
        if (!has) return false;
      }
      return true;
    }

    function sortFn(a, b) {
      var k = state.sort, d = state.dir;
      var col = COLS.filter(function (c) { return c.key === k; })[0] || {};
      if (col.num) {
        var av = k === "ev" ? evNum(a) : a[k];
        var bv = k === "ev" ? evNum(b) : b[k];
        if (av == null) av = -Infinity;
        if (bv == null) bv = -Infinity;
        return (av - bv) * d || a.name.localeCompare(b.name);
      }
      var sa = a[k] || "", sb = b[k] || "";
      if (Array.isArray(sa)) sa = sa.join(" ");
      if (Array.isArray(sb)) sb = sb.join(" ");
      return String(sa).localeCompare(String(sb)) * d || a.name.localeCompare(b.name);
    }

    function headHTML() {
      return "<tr>" + COLS.map(function (c) {
        if (!c.sortable) return '<th>' + esc(c.label) + '</th>';
        var arrow = state.sort === c.key ? (state.dir === 1 ? " ▲" : " ▼") : "";
        return '<th class="is-sortable" data-key="' + c.key + '">' + esc(c.label) + arrow + '</th>';
      }).join("") + "</tr>";
    }

    function rowHTML(it) {
      var kw = (it.keywords || []).map(function (k) {
        return '<span class="sc-bestiary__kw">' + esc(k) + '</span>';
      }).join(" ");
      return "<tr>" +
        '<td><a href="' + esc(it.href) + '">' + esc(it.name) + "</a></td>" +
        "<td>" + esc(cap(it.type)) + "</td>" +
        "<td>" + esc(it.level) + "</td>" +
        "<td>" + esc(it.ev) + "</td>" +
        "<td>" + esc(it.role || "—") + "</td>" +
        "<td>" + esc(it.organization || "—") + "</td>" +
        "<td>" + esc(it.size || "—") + "</td>" +
        "<td>" + kw + "</td></tr>";
    }

    function render() {
      var list = items.filter(matches).sort(sortFn);
      var any = state.q || state.lvlMin != null || state.lvlMax != null || state.evMin != null || state.evMax != null ||
        facets.some(function (f) { return Object.keys(state.sel[f.key]).length; });
      elClear.hidden = !any;
      elCount.innerHTML = "<b>" + list.length + "</b> of " + items.length + " entries";
      elResults.innerHTML = list.length
        ? '<table class="sc-bestiary"><thead>' + headHTML() + "</thead><tbody>" + list.map(rowHTML).join("") + "</tbody></table>"
        : '<div class="sc-browse__empty">No creatures match these filters.</div>';
      elResults.querySelectorAll("th.is-sortable").forEach(function (th) {
        th.addEventListener("click", function () {
          var k = this.dataset.key;
          if (state.sort === k) state.dir = -state.dir; else { state.sort = k; state.dir = 1; }
          render();
        });
      });
    }
    render();
  }

  function facetRow(f) {
    var chips = f.values.map(function (v) {
      var label = f.display ? f.display(v) : v;
      return '<button type="button" class="sc-chip" role="button" aria-pressed="false" data-facet="' +
        f.key + '" data-value="' + esc(v) + '">' + esc(label) + "</button>";
    }).join("");
    return '<div class="sc-browse__facet"><span class="lbl">' + esc(f.label) + '</span>' +
      '<div class="sc-browse__chips">' + chips + "</div></div>";
  }

  window.SCBestiary = { mount: mount };

  // ── Advanced-data seam (Plan B §B5, NOT built now) ─────────────────────────
  // To enable "inflicts <condition>"-style facets later, publish a second island
  // (or window.SC_BESTIARY_AUX) keyed by href, and left-join it onto `items`
  // before building `facets` in mount(). The current build ships no aux data, so
  // this hook intentionally does nothing today.

  function init() { document.querySelectorAll(".sc-bestiary-mount").forEach(mount); }
  if (typeof document$ !== "undefined" && document$ && typeof document$.subscribe === "function") {
    document$.subscribe(init);
  } else if (document.readyState !== "loading") {
    init();
  } else {
    document.addEventListener("DOMContentLoaded", init);
  }
})();
```

- [ ] **Step 2: Wire it into `v2/mkdocs.yml`.** In the `extra_javascript:` list, after the `javascripts/steel-feature-browser.js` line, add:
```yaml
  - javascripts/steel-bestiary-browser.js
```

- [ ] **Step 3: Sanity-check the JS parses** (no Node test harness; use `node --check` via devbox):
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'node --check v2/docs/javascripts/steel-bestiary-browser.js && echo "JS SYNTAX OK"'
```
Expected: `JS SYNTAX OK`.

- [ ] **Step 4: Commit**
```bash
cd /home/scott/code/steelCompendium/workspace/v2
git add docs/javascripts/steel-bestiary-browser.js mkdocs.yml
git commit -m "feat(v2): SCBestiary search/filter component for the Bestiary tab"
```

---

## Task 5: `steel-bestiary.css` (results table + range inputs)

**Files:**
- Create: `v2/docs/stylesheets/steel-bestiary.css`
- Modify: `v2/mkdocs.yml` (`extra_css`)

Baseline functional styling only (the high-fantasy-steel polish is a later Claude Design pass). The `.sc-browse` shell (search bar, facet chips, count, clear, empty) is already styled by `steel-indexes.css`; this file adds the range row and the results table.

- [ ] **Step 1: Create `v2/docs/stylesheets/steel-bestiary.css`:**

```css
/* Bestiary Search & Filter — results table + Level/EV range inputs.
   Reuses the .sc-browse shell from steel-indexes.css; this file adds only the
   table + range UI. Baseline functional styling; visual polish is a later pass. */

.sc-browse__ranges {
  display: flex;
  flex-wrap: wrap;
  gap: 1.25rem;
  margin: .25rem 0 .75rem;
}
.sc-browse__range { display: flex; align-items: center; gap: .4rem; }
.sc-browse__range .lbl { font-weight: 600; font-size: .8rem; opacity: .85; }
.sc-browse__range input {
  width: 4.5rem;
  padding: .25rem .4rem;
  border: 1px solid var(--md-default-fg-color--lighter, #ccc);
  border-radius: 4px;
  background: var(--md-default-bg-color, #fff);
  color: inherit;
  font: inherit;
}
.sc-range__dash { opacity: .6; }

.sc-bestiary {
  width: 100%;
  border-collapse: collapse;
  font-size: .82rem;
}
.sc-bestiary thead th {
  position: sticky;
  top: 0;
  background: var(--md-default-bg-color, #fff);
  text-align: left;
  padding: .45rem .6rem;
  border-bottom: 2px solid var(--md-default-fg-color--lighter, #ddd);
  white-space: nowrap;
}
.sc-bestiary th.is-sortable { cursor: pointer; user-select: none; }
.sc-bestiary th.is-sortable:hover { color: var(--md-accent-fg-color, inherit); }
.sc-bestiary tbody td {
  padding: .4rem .6rem;
  border-bottom: 1px solid var(--md-default-fg-color--lightest, #eee);
  vertical-align: top;
}
.sc-bestiary tbody tr:hover { background: var(--md-default-fg-color--lightest, #f3f3f3); }
.sc-bestiary__kw {
  display: inline-block;
  font-size: .72rem;
  padding: .05rem .4rem;
  margin: .05rem .1rem;
  border-radius: 10px;
  background: var(--md-default-fg-color--lightest, #eee);
  white-space: nowrap;
}
```

- [ ] **Step 2: Wire it into `v2/mkdocs.yml`.** In `extra_css:`, after `stylesheets/steel-indexes.css`, add:
```yaml
  - stylesheets/steel-bestiary.css
```

- [ ] **Step 3: Commit**
```bash
cd /home/scott/code/steelCompendium/workspace/v2
git add docs/stylesheets/steel-bestiary.css mkdocs.yml
git commit -m "feat(v2): baseline styling for the Bestiary search table + ranges"
```

---

## Task 6: Full build + manual verification

**Files:** none (verification only).

- [ ] **Step 1: Build the site and confirm the MkDocs build is clean**
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml'
devbox run -- bash -c 'cd v2 && mkdocs build --strict 2>&1 | tail -20'
```
Expected: `site` ok; `mkdocs build` completes (warnings about the pre-existing kit-flatten links are known — see FOLLOWUPS #4 — but the build should not fail on the new files). If `--strict` trips on the pre-existing warnings, drop `--strict` and confirm a plain `mkdocs build` succeeds.

- [ ] **Step 2: Manual review** (Scott's checkpoint — see memory `scott-working-style`)
```bash
cd /home/scott/code/steelCompendium/workspace/v2
devbox run -- bash -c 'cd v2 && mkdocs serve'
```
On `/Bestiary/`, confirm:
1. The search box, Level/EV range inputs, and facet chips (Type / Role / Organization / Size / Keyword) all render.
2. Typing a name filters; the count updates ("N of M entries").
3. The motivating query works: set **Type=Statblock**, **Keyword=Undead**, **EV min=3 max=6** → only undead statblocks in EV 3–6 show.
4. Clicking a column header sorts (▲/▼ toggles); EV and Level sort numerically.
5. A result row's name links to the correct Browse page (`/Browse/monster/...`).
6. "Clear filters" resets everything.
7. Navigate away and back (Material instant nav) — the widget re-mounts (the bare-`<script>` fallback path).

- [ ] **Step 3:** Report results to Scott; do not deploy (deploy is Scott's separate call).

---

## Task 7: Docs sync

**Files:**
- Modify: `ARCHITECTURE.md`, `steel-etl/CLAUDE.md`

- [ ] **Step 1: `ARCHITECTURE.md`** — in the site-builder "Key operations", note the new step: `buildBestiarySearchPage` emits the `Bestiary/index.md` Search & Filter landing (a `.sc-bestiary-mount` JSON island over the Browse statblock/terrain/retainer frontmatter) after index generation; mounted client-side by `steel-bestiary-browser.js`. Update the earlier "Bestiary tab is being repurposed (Plan B)" wording to "Plan B shipped" / present tense.

- [ ] **Step 2: `steel-etl/CLAUDE.md`** — add a key-files row for `internal/site/bestiary_search.go` (collects bestiary frontmatter → the `.sc-bestiary-mount` data island; emitted from `Build()` after `generateIndexPages`; consumed by `SCBestiary`). In the Site-builder feature list, mention the Bestiary search landing as a generated page (not static). Note the advanced-data seam (§B5) is reserved, not built.

- [ ] **Step 3: Mark FOLLOWUPS / spec status** — in `steel-etl/docs/superpowers/specs/2026-06-10-bestiary-restructure-and-search-design.md`, the Part B section can be annotated that the v1 utility shipped (advanced condition-queries still deferred to the §B5 seam pending community data).

- [ ] **Step 4: Commit** (submodule + workspace, as in Part A)
```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add CLAUDE.md docs/superpowers/specs/2026-06-10-bestiary-restructure-and-search-design.md
git commit -m "docs: Bestiary search utility (Plan B) build step + component"
cd /home/scott/code/steelCompendium/workspace
git add ARCHITECTURE.md && git commit -m "docs(bestiary): search utility shipped (Plan B)"
git add steel-etl && git commit -m "chore: bump steel-etl — bestiary search utility (Plan B)"
```

---

## Self-Review (completed during authoring)

- **Spec coverage:** B1 (data island + SCBrowse-sibling architecture) → Tasks 1, 2, 4. B2 (facets: type/role/org/size/keyword + Level/EV ranges + name search) → Task 4. B3 (dense sortable table) → Task 4 (clickable `th` sort, numeric EV/Level). B4 (unified view + type filter; terrain has no role/keywords → those facets simply don't match its rows, and `uniqueSorted` skips empties) → Tasks 1, 4. B5 (advanced-data seam, NOT built) → documented hook in Task 4 + Task 7 note. B6 placeholder→landing transition → Tasks 2–3.
- **Placeholder scan:** none — Go and JS are complete; the `var _ = json.Marshal` line is an explicit, documented Task-1→Task-2 compile bridge that Task 2 Step 3 removes.
- **Type/key consistency:** the Go `bestiaryItem` JSON tags (`type`/`name`/`level`/`ev`/`role`/`organization`/`keywords`/`size`/`href`) exactly match the keys the JS reads (`it.type`, `it.level`, `it.ev`, `it.role`, `it.organization`, `it.keywords`, `it.size`, `it.href`) and the `COLS`/facet `key`s. `collectBestiaryItems`/`buildBestiarySearchPage`/`bestiaryItemType`/`unquote` signatures are consistent across Tasks 1–2. The inner island class `sc-browse-data` matches the JS mount's selector + instant-nav fallback; the outer mount class `sc-bestiary-mount` matches the JS auto-mount query.
- **Dependency:** requires Part A merged (the Browse monster/terrain/retainer pages must exist for `collectBestiaryItems` to walk).
- **Known soft spots flagged in-plan:** JS has no unit-test harness (verified via `node --check` + manual browser review); `mkdocs build --strict` may trip on pre-existing unrelated warnings (fallback to plain build).
