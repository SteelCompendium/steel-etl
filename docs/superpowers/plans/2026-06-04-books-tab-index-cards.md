# Books-Tab Index Cards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the flat link-list index pages under the v2 site's "Books" tab (the `Read` section) with rich `.sc-card` cards, matching the visual style of the "Browse" tab's type-index pages.

**Architecture:** The Books tab is the `Read` section (`title: Books`) in `v2/site.yaml`, rendered with `GroupByBook: true`. Its two index types — the section landing (`Read/index.md`, listing books) and each per-book index (`Read/<book>/index.md`, listing chapters) — are emitted by `writeBookNavAndIndexes` in `steel-etl/internal/site/build.go` as `<div class="browse-index">` link lists. We swap those for `<div class="sc-cards">` card grids built with the existing `card()` helper from `cards.go`. Book cards get a per-book icon + hand-authored description from new `site.yaml` fields; chapter cards get a shared icon + a blurb auto-extracted from the chapter's first prose paragraph. All required CSS (`.sc-cards`, `.sc-card`, `.sc-card__flavor`, `.sc-crest`, …) already exists in `v2/docs/stylesheets/steel-redesign.css` and is reused unchanged.

**Tech Stack:** Go (steel-etl site builder), `gopkg.in/yaml.v3`, MkDocs Material, inline-SVG Material Design Icon glyphs.

---

## Background facts (read once before starting)

- **Build the project:** Go is not on PATH; use devbox. From the workspace root (`/home/vexa/code/steel_compendium/workspace`):
  ```bash
  devbox run -- bash -c 'cd steel-etl && go test ./internal/site/...'
  ```
- **The card helper already exists.** `card(file, icon, typeLabel, name, inner string) string` in `steel-etl/internal/site/cards.go:340` emits a `<div class="sc-card sc-fil">` with a stretched-link `<a href="dirURL(file)">`, a `.sc-crest` icon (`crestSVG(icon)` → looks up `iconPaths[icon]`, falling back to the `scroll` glyph), a type label, the name, and the `inner` HTML. We reuse it verbatim.
- **`crestSVG(icon)`** (`cards.go:536`) returns inline `<svg>` for `iconPaths[icon]`; unknown keys fall back to the `scroll` glyph.
- **Blurb extraction already exists.** `bodyBlurb(body, max)` (`cards.go:803`) returns the first prose paragraph (markdown stripped, links flattened to their text), truncated to `max` runes. `flavorDiv(text, max)` (`cards.go:514`) wraps prose text in `<div class="sc-card__flavor">` (escaping; `max<=0` = no truncation). `blurbBlock(text)` wraps in `<div class="sc-card__blurb">`.
- **`dirURL(href)`** (`cards.go:422`) maps `"ancestries.md"`→`"ancestries/"` and `"heroes/index.md"`→`"heroes/"`, matching MkDocs `use_directory_urls`. So `card("heroes/index.md", …)` links to the book folder, and `card("ancestries.md", …)` links to the sibling chapter — both correct relative to their index page.
- **The function we modify:** `writeBookNavAndIndexes(cfg *Config, section SectionConfig)` in `build.go:236`. It already collects `chapterRef{file, name, order}` per book and writes per-book + section `index.md`/`.nav.yml`. We change only the two `index.md` body builders (per-book at `build.go:296-309`, section landing at `build.go:332-340`), plus extend `chapterRef` to carry a blurb.
- **`chapterRef`** is defined at `build.go:227`.
- **Existing tests to keep green:** `TestBuildBookNavAndIndexes` (`build_test.go:940`) asserts chapter names appear in source order in the per-book index; `TestBuildBookPlaceholderForEmptyBook` (`build_test.go:1055`) asserts the empty-book placeholder text + label survive. Card markup must preserve those substrings (the card name *is* the chapter/book label).
- **Books configured today** (`v2/site.yaml`): `heroes` (Heroes), `beastheart` (Beastheart), `bestiary` (Bestiary).

---

## File Structure

- **Modify** `steel-etl/internal/site/config.go` — add `Description` + `Icon` fields to `BookConfig`.
- **Modify** `steel-etl/internal/site/config_test.go` — assert the new fields parse.
- **Modify** `steel-etl/internal/site/cards.go` — add `book`, `chapter`, `sword-cross`, `paw`, `dragon` glyphs to `iconPaths`.
- **Create** `steel-etl/internal/site/cards_book.go` — `bookCard` + `chapterCard` builders (keeps `cards.go` from growing further; it is already 870 lines).
- **Create** `steel-etl/internal/site/cards_book_test.go` — unit tests for the two builders.
- **Modify** `steel-etl/internal/site/build.go` — extend `chapterRef` with `blurb`; emit `.sc-cards` grids in `writeBookNavAndIndexes`.
- **Modify** `steel-etl/internal/site/build_test.go` — assert card markup in the generated indexes.
- **Modify** `v2/site.yaml` — add `description` + `icon` per book.

---

## Task 1: Add `Description` and `Icon` fields to `BookConfig`

**Files:**
- Modify: `steel-etl/internal/site/config.go:110-115`
- Test: `steel-etl/internal/site/config_test.go`

- [ ] **Step 1: Write the failing test**

Add this test to `steel-etl/internal/site/config_test.go` (after `TestLoadSiteConfig`, around line 70):

```go
func TestLoadSiteConfig_BookDescriptionAndIcon(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "site.yaml")
	content := `
source_dir: ../output/en/md-linked
docs_dir: ./docs
books:
  - key: mcdm.heroes.v1
    folder: heroes
    label: Heroes
    order: 1
    icon: sword-cross
    description: The core rulebook for building and playing heroes.
sections:
  - name: Read
    title: Books
    include:
      - chapter/
    group_by_book: true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadSiteConfig(path)
	if err != nil {
		t.Fatalf("LoadSiteConfig failed: %v", err)
	}
	if len(cfg.Books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(cfg.Books))
	}
	if cfg.Books[0].Icon != "sword-cross" {
		t.Errorf("Icon = %q, want %q", cfg.Books[0].Icon, "sword-cross")
	}
	want := "The core rulebook for building and playing heroes."
	if cfg.Books[0].Description != want {
		t.Errorf("Description = %q, want %q", cfg.Books[0].Description, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestLoadSiteConfig_BookDescriptionAndIcon -v'`
Expected: FAIL — the test references `cfg.Books[0].Icon` / `.Description`, which do not exist yet, so the package fails to compile (`unknown field`).

- [ ] **Step 3: Add the fields**

In `steel-etl/internal/site/config.go`, replace the `BookConfig` struct (lines 110-115):

```go
type BookConfig struct {
	Key    string `yaml:"key"`
	Folder string `yaml:"folder"`
	Label  string `yaml:"label"`
	Order  int    `yaml:"order"`
	// Description is a hand-authored blurb shown on the book's card in the
	// Read-section landing index. Optional.
	Description string `yaml:"description,omitempty"`
	// Icon is the iconPaths key for the book card's crest (e.g. "sword-cross").
	// Empty falls back to the generic "book" glyph. See iconPaths in cards.go.
	Icon string `yaml:"icon,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestLoadSiteConfig_BookDescriptionAndIcon -v'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git -C steel-etl add internal/site/config.go internal/site/config_test.go
git -C steel-etl commit -m "feat(site): add description + icon fields to BookConfig"
```

---

## Task 2: Add book/chapter icon glyphs to `iconPaths`

The book cards use per-book icons (`sword-cross`, `paw`, `dragon`); chapter cards use a shared `chapter` glyph; the book fallback is `book`. None of these keys exist in `iconPaths` yet. We fetch the exact Material Design Icon path data from the canonical MDI SVG repo (the same source the existing glyphs were copied from) rather than hand-typing path strings.

**Files:**
- Modify: `steel-etl/internal/site/cards.go:855-870` (the `iconPaths` map literal)
- Test: `steel-etl/internal/site/cards_book_test.go` (created in Task 3; the glyph-presence assertion is added in Task 3 Step 1)

- [ ] **Step 1: Fetch the five glyph path strings**

Run this from the workspace root. It prints five ready-to-paste Go map lines (key → `<path d="…"/>`):

```bash
for spec in book-open-variant:book book-open-page-variant:chapter sword-cross:sword-cross paw:paw dragon:dragon; do
  name="${spec%%:*}"; key="${spec##*:}"
  d=$(curl -fsSL "https://raw.githubusercontent.com/Templarian/MaterialDesign-SVG/master/svg/${name}.svg" \
       | grep -o 'd="[^"]*"' | head -1 | sed 's/^d="//; s/"$//')
  if [ -z "$d" ]; then echo "!! FAILED to fetch $name"; else
    printf '\t"%s": `<path d="%s"/>`, // %s\n' "$key" "$d" "$name"
  fi
done
```

Expected: five lines, e.g. `"book": `<path d="M5,3H7V5H5V10A2,2..."/>`, // book-open-variant` (exact `d` values come from the fetch). If any line prints `!! FAILED`, fetch that icon's `d` manually from https://pictogrammers.com/library/mdi/icon/<name>/ ("Copy Path") instead.

- [ ] **Step 2: Paste the five lines into `iconPaths`**

In `steel-etl/internal/site/cards.go`, inside the `iconPaths` map literal, add the five fetched lines immediately before the existing `"scroll":` entry (around line 869). The result looks like (with the real `d` values substituted from Step 1):

```go
	"negotiation":  `<path d="M17,12V3A1,1 ...Z"/>`, // forum  (existing — do not change)
	"book":         `<path d="…book-open-variant path…"/>`,      // book-open-variant (book card default)
	"chapter":      `<path d="…book-open-page-variant path…"/>`, // book-open-page-variant (chapter cards)
	"sword-cross":  `<path d="…sword-cross path…"/>`,            // sword-cross (heroes book)
	"paw":          `<path d="…paw path…"/>`,                    // paw (beastheart book)
	"dragon":       `<path d="…dragon path…"/>`,                 // dragon (bestiary book)
	"scroll":       `<path d="M17.8,20C17.4,21.2 ...Z"/>`, // script-text (existing — do not change)
```

- [ ] **Step 3: Verify the package compiles**

Run: `devbox run -- bash -c 'cd steel-etl && go build ./internal/site/'`
Expected: no output (success). A stray backtick or unescaped char in a pasted path is caught here.

- [ ] **Step 4: Commit**

```bash
git -C steel-etl add internal/site/cards.go
git -C steel-etl commit -m "feat(site): add book/chapter/per-book MDI glyphs to iconPaths"
```

---

## Task 3: Add `bookCard` and `chapterCard` builders

**Files:**
- Create: `steel-etl/internal/site/cards_book.go`
- Test: `steel-etl/internal/site/cards_book_test.go`

- [ ] **Step 1: Write the failing tests**

Create `steel-etl/internal/site/cards_book_test.go`:

```go
package site

import "strings"

import "testing"

func TestBookCard_IconLabelDescriptionLink(t *testing.T) {
	b := BookConfig{
		Folder:      "heroes",
		Label:       "Heroes",
		Order:       1,
		Icon:        "sword-cross",
		Description: "The core rulebook for building heroes.",
	}
	got := bookCard(b)
	for _, want := range []string{
		`class="sc-card`,                  // it is a stat-card
		`href="heroes/"`,                  // stretched link to the book folder
		`>Book<`,                          // type label
		`>Heroes<`,                        // name
		`class="sc-card__flavor"`,         // description rendered as flavor
		"The core rulebook for building heroes.",
		iconPaths["sword-cross"],          // the per-book crest glyph
	} {
		if !strings.Contains(got, want) {
			t.Errorf("bookCard missing %q in:\n%s", want, got)
		}
	}
}

func TestBookCard_DefaultsToBookGlyphWhenNoIcon(t *testing.T) {
	got := bookCard(BookConfig{Folder: "bestiary", Label: "Bestiary"})
	if !strings.Contains(got, iconPaths["book"]) {
		t.Errorf("bookCard without Icon should use the 'book' glyph:\n%s", got)
	}
	if strings.Contains(got, `class="sc-card__flavor"`) {
		t.Errorf("bookCard without Description should emit no flavor div:\n%s", got)
	}
}

func TestChapterCard_NameBlurbAndLink(t *testing.T) {
	body := "# Ancestries\n\n---\n\nFantastic peoples inhabit the worlds of [Draw Steel](x.md).\n"
	got := chapterCard("ancestries.md", "Ancestries", body)
	for _, want := range []string{
		`href="ancestries/"`,             // link to the sibling chapter
		`>Chapter<`,                       // type label
		`>Ancestries<`,                    // name
		`class="sc-card__flavor"`,         // blurb
		"Fantastic peoples inhabit the worlds of Draw Steel.", // links flattened to text
		iconPaths["chapter"],              // shared chapter glyph
	} {
		if !strings.Contains(got, want) {
			t.Errorf("chapterCard missing %q in:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run "TestBookCard|TestChapterCard" -v'`
Expected: FAIL to compile — `bookCard` / `chapterCard` undefined.

- [ ] **Step 3: Write the builders**

Create `steel-etl/internal/site/cards_book.go`:

```go
package site

// Book/chapter index cards for the "Books" tab (the GroupByBook "Read" section).
//
// The Read-section landing lists books as bookCard()s; each per-book index lists
// its chapters as chapterCard()s. Both reuse card() (cards.go) and the shared
// .sc-card CSS, matching the Browse tab's type-index cards. Emitted by
// writeBookNavAndIndexes() in build.go.

// bookCard renders one book's card for the Read-section landing index: a per-book
// crest (BookConfig.Icon, falling back to the generic "book" glyph), the type
// label "Book", the book label, and the hand-authored description (site.yaml) as
// flavor prose. The stretched link points at the book folder.
func bookCard(b BookConfig) string {
	icon := b.Icon
	if icon == "" {
		icon = "book"
	}
	inner := flavorDiv(b.Description, 0) // "" when no description → empty string
	return card(b.Folder+"/index.md", icon, "Book", b.Label, inner)
}

// chapterCard renders one chapter's card for a per-book index: the shared
// "chapter" crest, the type label "Chapter", the chapter name, and a blurb taken
// from the chapter body's first prose paragraph (truncated). file is the chapter
// basename (e.g. "ancestries.md"); the stretched link resolves to its directory
// URL (e.g. "ancestries/").
func chapterCard(file, name, body string) string {
	inner := flavorDiv(bodyBlurb(body, 200), 0)
	return card(file, "chapter", "Chapter", name, inner)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run "TestBookCard|TestChapterCard" -v'`
Expected: PASS (all three)

- [ ] **Step 5: Commit**

```bash
git -C steel-etl add internal/site/cards_book.go internal/site/cards_book_test.go
git -C steel-etl commit -m "feat(site): add bookCard and chapterCard builders"
```

---

## Task 4: Emit card grids from `writeBookNavAndIndexes`

Replace the two `<div class="browse-index">` link lists with `<div class="sc-cards">` card grids, and carry each chapter's blurb on `chapterRef`.

**Files:**
- Modify: `steel-etl/internal/site/build.go:227-231` (`chapterRef` struct), `build.go:262-272` (chapter collection), `build.go:296-309` (per-book index body), `build.go:332-340` (section landing body)
- Test: `steel-etl/internal/site/build_test.go`

- [ ] **Step 1: Write the failing test**

Add this test to `steel-etl/internal/site/build_test.go` (after `TestBuildBookNavAndIndexes`, around line 997):

```go
func TestBuildBookIndexesUseCards(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	docs := filepath.Join(dir, "docs")
	writeFile(t, filepath.Join(src, "chapter", "ancestries.md"),
		"---\nname: Ancestries\nscc: mcdm.heroes.v1/chapter/ancestries\ntype: chapter\norder: 3\n---\n\nFantastic peoples inhabit the worlds of Draw Steel.\n")
	cfg := &Config{
		SourceDirs: []string{src},
		DocsDir:    docs,
		Books: []BookConfig{
			{Key: "mcdm.heroes.v1", Folder: "heroes", Label: "Heroes", Order: 1,
				Icon: "sword-cross", Description: "The core rulebook."},
		},
		Sections: []SectionConfig{{Name: "Read", Title: "Books", Include: []string{"chapter/"}, GroupByBook: true}},
	}
	if _, err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}

	// Section landing: a book card grid with the book's description + icon.
	landing, _ := os.ReadFile(filepath.Join(docs, "Read", "index.md"))
	for _, want := range []string{`<div class="sc-cards">`, `class="sc-card`, ">Heroes<", "The core rulebook.", iconPaths["sword-cross"]} {
		if !strings.Contains(string(landing), want) {
			t.Errorf("Read landing missing %q:\n%s", want, landing)
		}
	}
	if strings.Contains(string(landing), "browse-index") {
		t.Errorf("Read landing should no longer use browse-index:\n%s", landing)
	}

	// Per-book index: a chapter card grid with the auto-extracted blurb.
	idx, _ := os.ReadFile(filepath.Join(docs, "Read", "heroes", "index.md"))
	for _, want := range []string{`<div class="sc-cards">`, ">Ancestries<", "Fantastic peoples inhabit the worlds of Draw Steel.", iconPaths["chapter"]} {
		if !strings.Contains(string(idx), want) {
			t.Errorf("heroes index missing %q:\n%s", want, idx)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildBookIndexesUseCards -v'`
Expected: FAIL — the generated indexes still contain `browse-index` and lack `sc-cards`.

- [ ] **Step 3: Extend `chapterRef` with a blurb**

In `steel-etl/internal/site/build.go`, replace the `chapterRef` struct (lines 227-231):

```go
type chapterRef struct {
	file  string // basename, e.g. "rewards.md"
	name  string // frontmatter name, e.g. "Rewards"
	order int
	blurb string // first prose paragraph of the chapter body (for the card)
}
```

- [ ] **Step 4: Capture the blurb during chapter collection**

In `writeBookNavAndIndexes`, replace the chapter-collection block (`build.go:262-272`) — currently:

```go
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
```

with (read the body too, and compute the blurb):

```go
			fm, body := splitFrontmatter(readFile(filepath.Join(bookDir, e.Name())))
			name := parseFrontmatterField(fm, "name")
			if name == "" {
				name = fileToTitle(e.Name())
			}
			chapters = append(chapters, chapterRef{
				file:  e.Name(),
				name:  name,
				order: parseFrontmatterInt(fm, "order", 1<<30),
				blurb: bodyBlurb(body, 200),
			})
```

- [ ] **Step 5: Emit a chapter card grid in the per-book index**

In `writeBookNavAndIndexes`, replace the per-book `index.md` body builder (`build.go:296-309`) — currently:

```go
		// Per-book index.md: ordered chapter list, or a placeholder when the
		// book has no chapters yet.
		var ib strings.Builder
		ib.WriteString("# " + b.Label + "\n\n---\n\n")
		if len(chapters) == 0 {
			ib.WriteString("*Chapters for this book haven't been added to the compendium yet.*\n")
		} else {
			ib.WriteString("<div class=\"browse-index\" markdown>\n\n")
			for _, c := range chapters {
				ib.WriteString("- [" + c.name + "](" + c.file + ")\n")
			}
			ib.WriteString("\n</div>\n")
		}
```

with:

```go
		// Per-book index.md: ordered chapter cards, or a placeholder when the
		// book has no chapters yet.
		var ib strings.Builder
		ib.WriteString("# " + b.Label + "\n\n---\n\n")
		if len(chapters) == 0 {
			ib.WriteString("*Chapters for this book haven't been added to the compendium yet.*\n")
		} else {
			ib.WriteString("<div class=\"sc-cards\">\n")
			for _, c := range chapters {
				ib.WriteString(chapterCard(c.file, c.name, c.blurb))
			}
			ib.WriteString("</div>\n")
		}
```

Note: `chapterCard`'s third arg is the body, but here we pass the already-extracted `c.blurb`. `bodyBlurb` on an already-stripped single paragraph is idempotent (it re-runs `firstProse`+`truncate`, which leave clean prose unchanged), so passing the blurb is safe and avoids re-reading the file.

- [ ] **Step 6: Emit a book card grid in the section landing**

In `writeBookNavAndIndexes`, replace the section-landing `index.md` body builder (`build.go:332-340`) — currently:

```go
	// Section landing index.md: lists the books. (Search exclusion frontmatter
	// is injected later by applySearchExclusion for search-excluded sections.)
	var lb strings.Builder
	lb.WriteString("# " + title + "\n\n---\n\n<div class=\"browse-index\" markdown>\n\n")
	for _, b := range present {
		lb.WriteString("- [" + b.Label + "](" + b.Folder + "/)\n")
	}
	lb.WriteString("\n</div>\n")
```

with:

```go
	// Section landing index.md: a card per book. (Search exclusion frontmatter
	// is injected later by applySearchExclusion for search-excluded sections.)
	var lb strings.Builder
	lb.WriteString("# " + title + "\n\n---\n\n<div class=\"sc-cards\">\n")
	for _, b := range present {
		lb.WriteString(bookCard(b))
	}
	lb.WriteString("</div>\n")
```

- [ ] **Step 7: Run the new test + the two existing book tests**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run "TestBuildBookIndexesUseCards|TestBuildBookNavAndIndexes|TestBuildBookPlaceholderForEmptyBook" -v'`
Expected: PASS (all three). The existing tests still pass because the chapter/book labels remain present as card names and the empty-book placeholder text is unchanged.

- [ ] **Step 8: Run the full site package test suite with race detector**

Run: `devbox run -- bash -c 'cd steel-etl && go test -race ./internal/site/...'`
Expected: `ok  ...steel-etl/internal/site`

- [ ] **Step 9: Commit**

```bash
git -C steel-etl add internal/site/build.go internal/site/build_test.go
git -C steel-etl commit -m "feat(site): render Books-tab indexes as sc-card grids"
```

---

## Task 5: Populate `site.yaml` with per-book descriptions and icons

**Files:**
- Modify: `v2/site.yaml` (the `books:` list, lines ~17-30)

- [ ] **Step 1: Add `description` and `icon` to each book**

In `v2/site.yaml`, replace the `books:` list:

```yaml
books:
  - key: mcdm.heroes.v1
    folder: heroes
    label: Heroes
    order: 1
  - key: mcdm.beastheart.v1
    folder: beastheart
    label: Beastheart
    order: 2
  - key: mcdm.monsters.v1
    folder: bestiary
    label: Bestiary
    order: 3
```

with:

```yaml
books:
  - key: mcdm.heroes.v1
    folder: heroes
    label: Heroes
    order: 1
    icon: sword-cross
    description: The core rulebook — everything you need to build and play a hero: ancestries, classes, kits, careers, and the rules of play.
  - key: mcdm.beastheart.v1
    folder: beastheart
    label: Beastheart
    order: 2
    icon: paw
    description: The beastheart class and its companion — bond with a monstrous ally and fight as one.
  - key: mcdm.monsters.v1
    folder: bestiary
    label: Bestiary
    order: 3
    icon: dragon
    description: Monsters and adversaries to challenge your heroes, with statblocks and encounter-building guidance.
```

- [ ] **Step 2: Confirm the config parses**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml --help >/dev/null 2>&1 || true'`
Then validate the YAML loads cleanly with a one-off:

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestLoadSiteConfig -v'`
Expected: PASS (the existing loader tests still pass; the new fields are optional and don't affect them).

- [ ] **Step 3: Commit**

```bash
git -C v2 add site.yaml
git -C v2 commit -m "feat: add book descriptions + icons for Books-tab cards"
```

---

## Task 6: Regenerate the site and verify visually

**Files:** none (build + visual verification only)

- [ ] **Step 1: Build the v2 site**

Run from the workspace root: `just deploy-v2`
Expected: completes without error. (This runs `steel-etl gen --all` then `steel-etl site --config v2/site.yaml`, regenerating `v2/docs/Read/`.)

- [ ] **Step 2: Confirm generated markup**

Run: `grep -l 'class="sc-cards"' v2/docs/Read/index.md v2/docs/Read/heroes/index.md`
Expected: both files listed.

Run: `grep -c 'class="sc-card ' v2/docs/Read/heroes/index.md`
Expected: a count equal to the number of Heroes chapters (> 10).

Run: `grep -q 'browse-index' v2/docs/Read/index.md && echo "STILL HAS browse-index (BAD)" || echo "no browse-index (good)"`
Expected: `no browse-index (good)`

- [ ] **Step 3: Serve and screenshot the Books landing + a book page**

Start the site:

Run: `devbox run -- bash -c 'cd v2 && mkdocs serve -a 127.0.0.1:8765' &` then wait ~4s for "Serving on".

Use Playwright to screenshot and visually confirm the cards render with crests, titles, and descriptions:
- Navigate to `http://127.0.0.1:8765/v2/Read/` (the Books landing) → confirm three book cards, each with a distinct crest icon (sword-cross / paw / dragon) and a description line.
- Navigate to `http://127.0.0.1:8765/v2/Read/heroes/` → confirm a grid of chapter cards, each with the chapter glyph, name, and a blurb.

Verify the per-book icons are the intended shapes (not the fallback `scroll` glyph, which would mean the icon key was wrong). Stop the server when done (`kill %1`).

Expected: cards visually match the Browse-tab type-index pages; icons are correct; descriptions/blurbs present; no broken layout.

- [ ] **Step 4: Commit the regenerated site (if `v2/docs/Read/` is tracked)**

Check first: `git -C v2 status --short docs/Read/`
- If files show as modified/untracked, commit them:
  ```bash
  git -C v2 add docs/Read
  git -C v2 commit -m "chore: regenerate Read tab with index cards"
  ```
- If `docs/Read/` is gitignored (generated output), skip — nothing to commit.

---

## Self-Review

**Spec coverage:**
- "index pages under the Books tab… visually interesting like Browse" → Tasks 3-4 swap both Books index types to `.sc-cards` reusing the Browse card helper + CSS. ✓
- "items are just books or chapters, cards relatively simple" → `bookCard`/`chapterCard` are minimal: crest + type label + name + one prose line, no stat grids. ✓
- "small description of each book card in the main index listing" → `BookConfig.Description` (Task 1) + `site.yaml` values (Task 5), rendered by `bookCard` as flavor (Task 3). ✓
- Per-book icons in `site.yaml`, generic book fallback → Task 1 (`Icon` field), Task 2 (glyphs), Task 3 (`bookCard` fallback to `"book"`). ✓
- Chapter cards = title + blurb, shared chapter glyph → Task 3 `chapterCard` + Task 4 blurb capture. ✓

**Placeholder scan:** The only "…" appears in Task 2 Step 2, where the real `d` values are produced by the Step 1 fetch command — concrete, not a placeholder. No "TBD"/"add error handling"/"write tests for the above" remain.

**Type consistency:** `card(file, icon, typeLabel, name, inner)`, `flavorDiv(text, max)`, `bodyBlurb(body, max)`, `crestSVG`/`iconPaths` are used with signatures matching their definitions in `cards.go`. `bookCard(BookConfig) string` and `chapterCard(file, name, body string) string` are defined in Task 3 and called with matching arguments in Task 4. `chapterRef.blurb` added in Task 4 Step 3 and consumed in Step 5. Icon keys (`book`, `chapter`, `sword-cross`, `paw`, `dragon`) added in Task 2 match the `site.yaml` values in Task 5 and the test references in Tasks 3-4. Consistent. ✓
