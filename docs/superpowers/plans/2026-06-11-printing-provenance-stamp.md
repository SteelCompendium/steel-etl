# Printing Provenance Stamp Implementation Plan

## Status

**Executed 2026-06-11** (subagent-driven, all 7 tasks). Workspace doc references in
Task 6 use the pre-restructure numbering (ROADMAP #8 is now #6; dated SCC history
lives in workspace `docs/scc-log.md`) — kept as written per the archives convention.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Flow the heroes source's `printing: "1.01b"` frontmatter through the pipeline as a non-identity build stamp — registry → SCC API JSON → rendered v2 pages — so any SCC code/page answers "which source printing generated this data?"

**Architecture:** The pipeline reads `printing` from each book's document frontmatter and records it in the SCC registry (`classification.json`) as a `books` map keyed by book id. The SCC API generator surfaces it in `index.json`/`scc.json` (a `books` map) and on every per-entry `resolve/*.json` (a `printing` field). The site builder gains a final stamping pass that injects `printing`/`printing_book` frontmatter into every built page whose `scc` book prefix has a recorded printing; a v2 theme partial renders it as a muted "Source: Heroes · printing 1.01b" line. **Identity never changes** — no SCC re-mint, no data-repo (JSON/YAML/md) output changes, no schema changes. Spec: `docs/superpowers/specs/2026-06-11-printing-provenance-and-code-lifecycle-design.md`.

**Tech Stack:** Go (steel-etl), MkDocs Material Jinja overrides (v2 repo).

**Environment gotcha:** Go is NOT on the system PATH. Run every Go command from the workspace root as:
`devbox run -- bash -c 'cd steel-etl && <go command>'`

**Commit conventions:** No AI/co-author attribution trailers in any commit message. steel-etl and v2 are separate git repos — commit each in its own repo.

---

## File structure

| File | Change |
|---|---|
| `internal/scc/registry.go` | `BookMeta`/`books` map + `SetBookPrinting`/`BookPrintings`, JSON round-trip |
| `internal/scc/registry_test.go` | round-trip + omitempty tests |
| `internal/pipeline/pipeline.go` | read `printing` fm; copy/record into registry (RunWithConfig + RunSharedOutputs); thread `Printings` into `SCCAPIGenerator` in `buildGenerators` |
| `internal/pipeline/pipeline_test.go` | pipeline-records-printing test |
| `internal/output/scc_api.go` | `Printings` field; `books` map in index/scc.json; `printing` on entries |
| `internal/output/scc_api_test.go` | API stamp test |
| `internal/site/config.go` | `Registry` config field |
| `internal/site/build.go` | `applyPrintingStamps` pass + `BuildResult.PrintingStamps` |
| `internal/site/build_test.go` | site stamp test |
| `internal/cli/site.go` | print stamp count |
| `v2/site.yaml` (v2 repo) | `registry: ../steel-etl/classification.json` |
| `v2/overrides/partials/content.html` (v2 repo) | provenance line |
| `v2/docs/stylesheets/extra.css` (v2 repo) | `.sc-provenance` styling |
| Docs | spec status table, workspace `ROADMAP.md` #8, both CLAUDE.md files, workspace CLAUDE.md SCC paragraph |

---

### Task 1: Registry book-printings map

**Files:**
- Modify: `internal/scc/registry.go`
- Test: `internal/scc/registry_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/scc/registry_test.go` (check the file's existing imports; these tests need `os`, `path/filepath`, `strings`, `testing`):

```go
func TestRegistryBookPrintingsRoundTrip(t *testing.T) {
	r := NewRegistry()
	r.Add("mcdm.heroes.v1/class/fury")
	r.SetBookPrinting("mcdm.heroes.v1", "1.01b")
	r.SetBookPrinting("", "9.99")      // ignored: empty book
	r.SetBookPrinting("mcdm.x.v1", "") // ignored: empty printing

	path := filepath.Join(t.TempDir(), "classification.json")
	if err := r.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got := loaded.BookPrintings()
	if len(got) != 1 || got["mcdm.heroes.v1"] != "1.01b" {
		t.Errorf("BookPrintings = %v, want map[mcdm.heroes.v1:1.01b]", got)
	}
}

func TestRegistryBookPrintingsAbsent(t *testing.T) {
	// A registry without printings must round-trip cleanly and omit the books key.
	r := NewRegistry()
	r.Add("mcdm.heroes.v1/class/fury")
	path := filepath.Join(t.TempDir(), "classification.json")
	if err := r.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := loaded.BookPrintings(); len(got) != 0 {
		t.Errorf("BookPrintings = %v, want empty", got)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), `"books"`) {
		t.Error("books key written for registry with no printings")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/scc/ -run TestRegistryBookPrintings -v'`
Expected: FAIL to compile — `r.SetBookPrinting undefined`

- [ ] **Step 3: Implement**

In `internal/scc/registry.go`:

Add to the `Registry` struct (after `schemeVersion int`):

```go
	books         map[string]BookMeta
```

Add the type after the `Registry` struct:

```go
// BookMeta holds non-identity provenance metadata for a book source.
// The printing is the source-document errata printing (e.g. "1.01b") the
// book's content was generated from — it is NOT part of any SCC identity.
// See docs/superpowers/specs/2026-06-11-printing-provenance-and-code-lifecycle-design.md.
type BookMeta struct {
	Printing string `json:"printing,omitempty"`
}
```

Add to `registryJSON` (after `Aliases`):

```go
	Books         map[string]BookMeta `json:"books,omitempty"`
```

In `NewRegistry()`, add to the struct literal:

```go
		books:         make(map[string]BookMeta),
```

Add methods (near `AddAlias`/`Aliases`):

```go
// SetBookPrinting records the source-document printing (e.g. "1.01b") for a
// book. Empty book or printing is a no-op.
func (r *Registry) SetBookPrinting(book, printing string) {
	if book == "" || printing == "" {
		return
	}
	meta := r.books[book]
	meta.Printing = printing
	r.books[book] = meta
}

// BookPrintings returns a copy of book → printing for books that declared one.
func (r *Registry) BookPrintings() map[string]string {
	out := make(map[string]string, len(r.books))
	for b, m := range r.books {
		if m.Printing != "" {
			out[b] = m.Printing
		}
	}
	return out
}
```

In `Save()`, after the aliases block:

```go
	if len(r.books) > 0 {
		data.Books = r.books
	}
```

In `LoadRegistry()`, after the aliases loop:

```go
	for book, meta := range raw.Books {
		r.books[book] = meta
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/scc/ -v'`
Expected: all PASS (including the pre-existing registry tests — `LoadRegistry` must still default `books` via `NewRegistry`)

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add internal/scc/registry.go internal/scc/registry_test.go
git commit -m "feat(scc): record per-book printing provenance in the registry"
```

---

### Task 2: Pipeline captures `printing` frontmatter into the registry

**Files:**
- Modify: `internal/pipeline/pipeline.go` (RunWithConfig ~line 65-95; RunSharedOutputs ~line 255-268)
- Test: `internal/pipeline/pipeline_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/pipeline/pipeline_test.go`. Add `"github.com/SteelCompendium/steel-etl/internal/scc"` to the file's imports.

```go
func TestRunPipelineRecordsPrinting(t *testing.T) {
	input := `---
book: mcdm.test.v1
printing: "1.01b"
---

<!-- @type: chapter | @id: intro -->
# Intro

Some text.
`
	inputPath := filepath.Join(t.TempDir(), "book.md")
	if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	registryPath := filepath.Join(t.TempDir(), "classification.json")

	if _, err := Run(inputPath, outputDir, registryPath); err != nil {
		t.Fatalf("Run: %v", err)
	}

	loaded, err := scc.LoadRegistry(registryPath)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if got := loaded.BookPrintings()["mcdm.test.v1"]; got != "1.01b" {
		t.Errorf("printing = %q, want \"1.01b\"", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/pipeline/ -run TestRunPipelineRecordsPrinting -v'`
Expected: FAIL — `printing = "", want "1.01b"`

- [ ] **Step 3: Implement**

In `internal/pipeline/pipeline.go`, `RunWithConfig`:

(a) After the `bookSource` extraction block (~line 72), add:

```go
	bookPrinting := ""
	if p, ok := doc.Frontmatter["printing"]; ok {
		if s, ok := p.(string); ok {
			bookPrinting = s
		}
	}
```

(b) Inside the existing-registry load block, after the aliases copy loop (`for alias, canonical := range existing.Aliases() { ... }`, ~line 88), add — this preserves *other* books' printings on a single-book run, mirroring how codes/aliases merge:

```go
			for book, printing := range existing.BookPrintings() {
				sccRegistry.SetBookPrinting(book, printing)
			}
```

(c) Immediately after the whole registry-load block (before `contextStack := ...`), add — this must run **before** `buildGenerators` so the SCC API generator (Task 3) sees the current book's printing:

```go
	// Non-identity provenance: which source-document printing this book's
	// content was generated from. Never part of any SCC code. See
	// docs/superpowers/specs/2026-06-11-printing-provenance-and-code-lifecycle-design.md.
	sccRegistry.SetBookPrinting(bookSource, bookPrinting)
```

(d) In `RunSharedOutputs` (~line 263), after its aliases copy loop, add the same merge so the shared-output API generator sees every book's printing:

```go
			for book, printing := range existing.BookPrintings() {
				sccRegistry.SetBookPrinting(book, printing)
			}
```

The registry `Save` at the end of `RunWithConfig` (~line 228) already persists the map — no change needed there.

- [ ] **Step 4: Run tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/pipeline/ -v'`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add internal/pipeline/pipeline.go internal/pipeline/pipeline_test.go
git commit -m "feat(pipeline): capture printing frontmatter into the SCC registry"
```

---

### Task 3: SCC API surfaces printings

**Files:**
- Modify: `internal/output/scc_api.go`
- Modify: `internal/pipeline/pipeline.go:386-394` (`buildGenerators`, SCC API construction)
- Test: `internal/output/scc_api_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/output/scc_api_test.go`. First look at the existing test at the top of that file (line ~22) and reuse its `content.ParsedContent` construction helper/style if one exists; otherwise this standalone form works:

```go
func TestSCCAPIPrintingStamp(t *testing.T) {
	dir := t.TempDir()
	gen := &SCCAPIGenerator{
		OutputDir: dir,
		BaseURL:   "https://example.com",
		Printings: map[string]string{"mcdm.heroes.v1": "1.01b"},
	}
	parsed := &content.ParsedContent{Frontmatter: map[string]any{
		"name": "Fury",
		"type": "class",
	}}
	if err := gen.WriteSection("mcdm.heroes.v1/class/fury", parsed); err != nil {
		t.Fatal(err)
	}
	if err := gen.Finalize(); err != nil {
		t.Fatal(err)
	}

	idx, err := os.ReadFile(filepath.Join(dir, "v1", "index.json"))
	if err != nil {
		t.Fatalf("read index.json: %v", err)
	}
	if !strings.Contains(string(idx), `"printing": "1.01b"`) {
		t.Errorf("index.json missing printing stamp:\n%s", idx)
	}

	res, err := os.ReadFile(filepath.Join(dir, "v1", "resolve", "mcdm.heroes.v1", "class", "fury.json"))
	if err != nil {
		t.Fatalf("read resolve entry: %v", err)
	}
	if !strings.Contains(string(res), `"printing": "1.01b"`) {
		t.Errorf("resolve entry missing printing:\n%s", res)
	}
}

func TestSCCAPINoPrintings(t *testing.T) {
	// Without printings, output must not contain a books key or printing field.
	dir := t.TempDir()
	gen := &SCCAPIGenerator{OutputDir: dir, BaseURL: "https://example.com"}
	parsed := &content.ParsedContent{Frontmatter: map[string]any{"name": "Fury", "type": "class"}}
	if err := gen.WriteSection("mcdm.heroes.v1/class/fury", parsed); err != nil {
		t.Fatal(err)
	}
	if err := gen.Finalize(); err != nil {
		t.Fatal(err)
	}
	idx, _ := os.ReadFile(filepath.Join(dir, "v1", "index.json"))
	if strings.Contains(string(idx), `"books"`) {
		t.Errorf("index.json has books key without printings:\n%s", idx)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/output/ -run TestSCCAPI -v'`
Expected: FAIL to compile — `unknown field Printings`

- [ ] **Step 3: Implement**

In `internal/output/scc_api.go`:

(a) Add to the `SCCAPIGenerator` struct (after `Aliases`):

```go
	Printings     map[string]string    // book source → printing (non-identity provenance)
```

(b) Add to `apiEntry` (after `Source`):

```go
	Printing string `json:"printing,omitempty"`
```

(c) Add a book-metadata type next to the other api types:

```go
// apiBook is per-book non-identity provenance metadata surfaced by the API.
type apiBook struct {
	Printing string `json:"printing,omitempty"`
}
```

(d) Add `Books map[string]apiBook \`json:"books,omitempty"\`` to **both** `apiIndex` (after `BaseURL`) and `apiRegistry` (after `BaseURL`).

(e) In `WriteSection`, after `source := extractSource(sccCode)`, set the field; the entry literal gains `Printing`:

```go
	g.entries[sccCode] = apiEntry{
		SCC:      sccCode,
		URL:      url,
		Name:     name,
		Type:     typeName,
		Source:   source,
		Printing: g.Printings[source],
	}
```

(f) In `Finalize`, before the `index.json` write, build the books map once:

```go
	var books map[string]apiBook
	if len(g.Printings) > 0 {
		books = make(map[string]apiBook, len(g.Printings))
		for b, p := range g.Printings {
			books[b] = apiBook{Printing: p}
		}
	}
```

and add `Books: books,` to both the `apiIndex{...}` and `apiRegistry{...}` literals. (`types.json` and alias resolve files need no change — alias entries copy the canonical `apiEntry`, which now carries `Printing`.)

(g) In `internal/pipeline/pipeline.go` `buildGenerators` (~line 386), add to the `SCCAPIGenerator` literal:

```go
			Printings:     sccRegistry.BookPrintings(),
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/output/ ./internal/pipeline/ -v'`
Expected: all PASS (pre-existing scc_api tests stay green — every new JSON field is omitempty)

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add internal/output/scc_api.go internal/output/scc_api_test.go internal/pipeline/pipeline.go
git commit -m "feat(scc-api): surface per-book printing provenance in API JSON"
```

---

### Task 4: Site builder printing-stamp pass

**Files:**
- Modify: `internal/site/config.go` (Config struct, ~line 14-39)
- Modify: `internal/site/build.go` (BuildResult ~line 25; `Build()` after the static-content copy; new func near `applySearchExclusion` ~line 699)
- Modify: `internal/cli/site.go` (~line 43-48, result printing)
- Test: `internal/site/build_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/site/build_test.go`. Add `"github.com/SteelCompendium/steel-etl/internal/scc"` to the file's imports.

```go
func TestBuild_PrintingStamps(t *testing.T) {
	srcDir := t.TempDir()
	classDir := filepath.Join(srcDir, "class")
	os.MkdirAll(classDir, 0755)
	page := "---\nname: Fury\nscc: mcdm.heroes.v1/class/fury\ntype: class\n---\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(classDir, "fury.md"), []byte(page), 0644); err != nil {
		t.Fatal(err)
	}

	regPath := filepath.Join(t.TempDir(), "classification.json")
	reg := scc.NewRegistry()
	reg.Add("mcdm.heroes.v1/class/fury")
	reg.SetBookPrinting("mcdm.heroes.v1", "1.01b")
	if err := reg.Save(regPath); err != nil {
		t.Fatal(err)
	}

	docsDir := filepath.Join(t.TempDir(), "docs")
	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Registry:  regPath,
		Books:     []BookConfig{{Key: "mcdm.heroes.v1", Label: "Heroes"}},
		Sections: []SectionConfig{
			{Name: "Browse", Include: []string{"class/"}},
		},
	}
	cfg.normalizeSources()

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if result.PrintingStamps == 0 {
		t.Fatal("expected at least one printing stamp")
	}

	data, err := os.ReadFile(filepath.Join(docsDir, "Browse", "class", "fury.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "printing: \"1.01b\"") {
		t.Errorf("missing printing frontmatter:\n%s", got)
	}
	if !strings.Contains(got, "printing_book: \"Heroes\"") {
		t.Errorf("missing printing_book frontmatter:\n%s", got)
	}
}
```

Note: if other tests in this file don't call `cfg.normalizeSources()`, drop that line here too (Build handles the legacy singular `SourceDir` via `SourceDirList`). Match the file's existing pattern.

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuild_PrintingStamps -v'`
Expected: FAIL to compile — `unknown field Registry`

- [ ] **Step 3: Implement**

(a) `internal/site/config.go` — add to `Config` (after `Books`):

```go
	// Registry is the path to the pipeline's SCC registry (classification.json),
	// used to read per-book printing provenance for page stamps. Optional —
	// empty disables stamping. Resolved relative to ConfigDir.
	Registry string `yaml:"registry,omitempty"`
```

(b) `internal/site/build.go` — add `PrintingStamps int` to `BuildResult` (after `SCCStubs`). Add `"github.com/SteelCompendium/steel-etl/internal/scc"` to imports.

(c) In `Build()`, after the static-content override copy block (stamping runs last so relocated, group-landing, and static-override pages are all covered):

```go
	// Printing provenance stamps: inject non-identity printing/printing_book
	// frontmatter from the classification registry (rendered by the v2 theme's
	// content partial). Runs after static overrides so every page is covered.
	stampCount, stampErrs := applyPrintingStamps(cfg)
	result.PrintingStamps = stampCount
	result.Errors = append(result.Errors, stampErrs...)
```

(d) Add near `applySearchExclusion`:

```go
// sccFrontmatterRe extracts the scc code from a page's frontmatter block.
var sccFrontmatterRe = regexp.MustCompile(`(?m)^scc: (\S+)$`)

// applyPrintingStamps injects printing provenance frontmatter (printing,
// printing_book) into every built page whose scc book prefix has a recorded
// source-document printing in the classification registry. Non-identity:
// purely presentational metadata, no SCC/URL impact. No-op when the config
// has no registry path or the registry records no printings.
func applyPrintingStamps(cfg *Config) (int, []string) {
	if cfg.Registry == "" {
		return 0, nil
	}
	reg, err := scc.LoadRegistry(cfg.ResolvePath(cfg.Registry))
	if err != nil {
		return 0, []string{fmt.Sprintf("printing stamps: load registry: %v", err)}
	}
	printings := reg.BookPrintings()
	if len(printings) == 0 {
		return 0, nil
	}

	labels := make(map[string]string, len(cfg.Books))
	for _, b := range cfg.Books {
		labels[b.Key] = b.Label
	}

	count := 0
	var errs []string
	filepath.Walk(cfg.DocsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("printing stamp read %s: %v", path, err))
			return nil
		}
		body := string(data)
		if !strings.HasPrefix(body, "---\n") {
			return nil
		}
		end := strings.Index(body[4:], "\n---")
		if end < 0 {
			return nil
		}
		m := sccFrontmatterRe.FindStringSubmatch(body[4 : 4+end])
		if m == nil {
			return nil
		}
		book, _, found := strings.Cut(m[1], "/")
		if !found {
			return nil
		}
		printing, ok := printings[book]
		if !ok {
			return nil
		}
		inject := fmt.Sprintf("printing: %q\n", printing)
		if label := labels[book]; label != "" {
			inject += fmt.Sprintf("printing_book: %q\n", label)
		}
		if err := os.WriteFile(path, []byte("---\n"+inject+body[4:]), 0644); err != nil {
			errs = append(errs, fmt.Sprintf("printing stamp write %s: %v", path, err))
			return nil
		}
		count++
		return nil
	})
	return count, errs
}
```

(e) `internal/cli/site.go` — after the `SCC stubs:` printf, add:

```go
	fmt.Printf("Printing stamps: %d files\n", result.PrintingStamps)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -v'`
Expected: all PASS (existing Build tests unaffected — they have no `Registry`, so the pass is a no-op)

- [ ] **Step 5: Run the full test suite**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./...'`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
cd steel-etl
git add internal/site/config.go internal/site/build.go internal/site/build_test.go internal/cli/site.go
git commit -m "feat(site): stamp printing provenance frontmatter onto built pages"
```

---

### Task 5: v2 repo — site.yaml registry path, theme partial, CSS

**Files (all in the `v2/` repo — separate git repo):**
- Modify: `v2/site.yaml`
- Modify: `v2/overrides/partials/content.html`
- Modify: `v2/docs/stylesheets/extra.css`

- [ ] **Step 1: Point site.yaml at the registry**

In `v2/site.yaml`, add next to `source_dirs:` (paths resolve relative to the config file, which lives in `v2/`):

```yaml
# SCC registry (for per-book printing provenance stamps)
registry: ../steel-etl/classification.json
```

- [ ] **Step 2: Render the stamp in the theme**

In `v2/overrides/partials/content.html`, after the `{{ page.content }}` line and before `{% include "partials/source-file.html" %}`:

```jinja
{% if page.meta.printing %}
<div class="sc-provenance">
  <small>
    Source:
    {%- if page.meta.printing_book %} {{ page.meta.printing_book }} &middot;{% endif %}
    printing {{ page.meta.printing }}
  </small>
</div>
{% endif %}
```

Also update the partial's header comment (`This file was automatically generated - do not edit`) — it is already hand-edited (the h1 block); replace that line with:

```jinja
{#-
Hand-edited Material content partial: drops the auto-injected h1 and renders
the per-page printing-provenance stamp (page.meta.printing / printing_book,
injected by `steel-etl site`). Re-sync against upstream on Material upgrades.
-#}
```

- [ ] **Step 3: Style it**

Append to `v2/docs/stylesheets/extra.css`:

```css
/* Per-page source-printing provenance stamp (printing/printing_book
   frontmatter injected by steel-etl site; rendered in overrides/partials/content.html) */
.sc-provenance {
  margin-top: 2.4rem;
  text-align: right;
  opacity: 0.55;
  font-size: 0.64rem;
}
```

- [ ] **Step 4: Verify rendering locally**

```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all'
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml'
grep -A1 '^printing:' "v2/docs/Browse/class/fury.md" | head -4
```

Expected: the site command prints a non-zero `Printing stamps:` count; `v2/docs/Browse/class/fury.md` frontmatter contains `printing: "1.01b"` and `printing_book: "Heroes"`. (Pages from books without a `printing:` field — beastheart/monsters/summoner — get no stamp; that's correct.)

Optionally serve and eyeball: `devbox run -- bash -c 'cd v2 && mkdocs serve'` → open a class page, confirm the muted "Source: Heroes · printing 1.01b" line above the footer.

- [ ] **Step 5: Commit (v2 repo)**

```bash
cd v2
git add site.yaml overrides/partials/content.html docs/stylesheets/extra.css
git commit -m "feat: render per-page source-printing provenance stamp"
```

Do **not** commit the regenerated `v2/docs/` content or the dirty `data/` repos — that's deploy output, deployed separately via `just deploy*`.

---

### Task 6: Printing tag + docs sync

**Files:**
- Modify: `steel-etl/docs/superpowers/specs/2026-06-11-printing-provenance-and-code-lifecycle-design.md` (Decision status table)
- Modify: `steel-etl/CLAUDE.md`
- Modify: workspace `ROADMAP.md` (#8), workspace `CLAUDE.md` (SCC paragraph)

- [ ] **Step 1: Tag the current printing**

The heroes source content in steel-etl HEAD is the 1.01b printing:

```bash
cd steel-etl
git tag heroes-printing-1.01b
git tag -l 'heroes-printing-*'
```

Expected: `heroes-printing-1.01b` listed. (Push the tag with the next push: `git push origin heroes-printing-1.01b` — pushing is the user's call.)

- [ ] **Step 2: Update the spec's Decision status table**

In the spec's final table, change the row
`| Printing stamp wired through registry → API → rendered pages; per-printing git tags | **Deferred** — see workspace ROADMAP.md |`
to
`| Printing stamp wired through registry → API → rendered pages; per-printing git tags | **Done** (2026-06-XX) — plan: docs/superpowers/plans/2026-06-11-printing-provenance-stamp.md |`
(use the actual completion date).

- [ ] **Step 3: Document the ingest convention in steel-etl/CLAUDE.md**

In the "SCC classification" section, extend the existing ⚠️ printing paragraph: replace the sentence `the printing is recorded in the separate, currently-inert printing: frontmatter field.` with:

```markdown
the printing is recorded in the separate `printing:` frontmatter field, which flows as a
non-identity build stamp: registry `books` map → SCC API (`index.json`/`scc.json` `books`,
per-entry `printing`) → site page frontmatter (`printing`/`printing_book`, injected by
`applyPrintingStamps` when `site.yaml` sets `registry:`; rendered by the v2 content partial).
**When ingesting a new errata printing:** update `printing:` in the book's frontmatter,
apply the content edits, then tag the commit `<book>-printing-<version>`
(e.g. `heroes-printing-1.01b`) so the exact source is recoverable via `git show`.
```

- [ ] **Step 4: Update workspace ROADMAP.md #8 and CLAUDE.md**

In `ROADMAP.md` #8: change `**Status:** direction settled 2026-06-11; implementation deferred.` to `**Status:** (a) printing stamp **done** <date> (plan: steel-etl/docs/superpowers/plans/2026-06-11-printing-provenance-stamp.md); (b) tombstone lifecycle still blocked on MCDM decision triggers.` and adjust the `- **Decision triggers:**` bullet to drop "implement (a) whenever convenient".

In workspace `CLAUDE.md`, in the SCC paragraph's printing sentence, replace `The printing lives in non-identity `printing:` frontmatter (inert for now); the build-stamp wiring and the removal/tombstone lifecycle model are deferred` with `The printing lives in non-identity `printing:` frontmatter and flows as a build stamp (registry → SCC API → page frontmatter/footer line); the removal/tombstone lifecycle model remains deferred`.

- [ ] **Step 5: Commit (both repos)**

```bash
cd steel-etl
git add CLAUDE.md docs/superpowers/specs/2026-06-11-printing-provenance-and-code-lifecycle-design.md
git commit -m "docs: printing provenance stamp shipped; record printing-ingest convention"
cd ..
git add ROADMAP.md CLAUDE.md steel-etl
git commit -m "docs: printing stamp (ROADMAP #8a) done + bump steel-etl"
```

---

### Task 7: End-to-end verification

- [ ] **Step 1: Full pipeline + checks**

```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go test ./... && go vet ./...'
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all'
```

Expected: tests/vet clean; gen reports the usual ~2,956 codes across 4 books, no warnings.

- [ ] **Step 2: Verify each surface**

```bash
# Registry
jq '.books' steel-etl/classification.json
# → {"mcdm.heroes.v1": {"printing": "1.01b"}}

# API index + a resolve entry (path depends on pipeline.yaml scc_api output_dir)
jq '.books' steelCompendium.github.io/docs/api/v1/index.json
jq '.printing' steelCompendium.github.io/docs/api/v1/resolve/mcdm.heroes.v1/class/fury.json
# → "1.01b"

# Site page frontmatter (after the Task 5 site run)
sed -n '1,8p' v2/docs/Browse/class/fury.md
# → frontmatter contains printing: "1.01b" / printing_book: "Heroes"

# A non-heroes page must NOT be stamped (monsters book has no printing: field)
grep -rl '^printing:' v2/docs/Browse/monster/ | wc -l
# → 0
```

- [ ] **Step 3: SCC stability check**

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate --config pipeline.yaml --scc-stable'
```

Expected: no code changes — the stamp is non-identity by construction.

- [ ] **Step 4: Report**

Deployment (`just deploy`) is the user's decision — report verification results and stop. Note for the deploy: the dirty `data/`, `steelCompendium.github.io`, and `v2/docs` trees from verification runs are normal regenerated output.

---

## Self-review notes

- **Spec coverage:** registry (Task 1-2), API (Task 3), rendered pages (Task 4-5), git tags + ingest convention (Task 6) — all "Done/implement" rows of the spec's status table are covered; the tombstone lifecycle is explicitly out of scope (still deferred).
- **No data-format changes:** JSON/YAML/md data-repo outputs, schemas (both copies), and SDK transforms are untouched — the stamp rides the registry, the API JSON, and site-build-time frontmatter only. SCC codes unchanged (`--scc-stable` verifies).
- **Books without printings** (beastheart/monsters/summoner) degrade gracefully everywhere: omitted from `books` maps, no `printing` field, no page stamp. Add `printing:` to their frontmatter when known.
