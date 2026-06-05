# steel-etl

Go CLI tool that processes annotated Draw Steel TTRPG markdown into structured, multi-format output.

## Dev environment

This project uses **devbox** for toolchain management. Go is NOT on the system PATH -- you must activate devbox first.

```bash
# From the workspace root (parent of steel-etl/):
devbox run --

# Then Go commands work:
cd steel-etl
go build ./...
go test ./...
go run ./cmd/steel-etl gen --config pipeline.yaml
```

For example, `devbox run -- go build ./...`

The `justfile` has convenience recipes that assume Go is on PATH (they work inside `devbox shell`):

```bash
just build    # Build binary
just test     # Run tests with -race
just cover    # Coverage report
just vet      # go vet
just fmt      # gofmt + goimports
just run gen --config pipeline.yaml  # Run with args
```

## Key files

| File | Purpose |
|------|---------|
| `pipeline.yaml` | Pipeline config: input, output formats, classification settings |
| `classification.json` | SCC registry (generated, gitignored) |
| `input/heroes/Draw Steel Heroes.md` | Annotated source — hand-maintained, canonical (the former `annotate_heroes.py` generator was retired; the `.md` holds ~4,055 cross-reference links and annotations that live only here) |
| `input/monsters/Draw Steel Monsters.md` | Annotated Monsters book source. Hand-maintained going forward; the initial annotation pass was bootstrapped by `scripts/annotate_monsters.pl` (by-heading-level: H2→monster group, H7→statblock, H9→featureblock/terrain). |
| `internal/cli/*.go` | CLI commands: gen, validate, classify, strip, site |
| `internal/content/registry.go` | Content parser registry (24 parsers) |
| `internal/pipeline/pipeline.go` | Main pipeline: parse -> classify -> generate |
| `internal/scc/registry.go` | SCC registry with freeze enforcement |
| `internal/site/build.go` | Site builder: maps ETL output to MkDocs structure |
| `internal/site/config.go` | Site builder config types (sections, groups, books) |
| `internal/site/cards.go` | Rich `.sc-card` index cards for Browse-tab type indexes (kit, class, ancestry, …) + shared `card()`/`crestSVG`/`iconPaths` |
| `internal/site/cards_book.go` | `.sc-card` index cards for the Books tab (`bookCard`, `chapterCard`) |
| `internal/site/permalinks.go` | SCC permalink redirect-stub generator |

## CLI commands

- `gen` -- Run full pipeline (parse + classify + output). **Processes only the primary book unless you pass `--all` (every book) or `--book <id>`.** `pipeline.yaml`'s `books:` list (beastheart, monsters) is skipped by a bare `gen`, so their `data/data-*` output goes stale; the `just deploy*` recipes pass `--all`. See `selectBookConfigs` in `internal/cli/gen.go`.
- `validate` -- Check annotation coverage, unknown types, SCC stability
- `classify` -- Display/export SCC codes, diff against registry
- `strip` -- Remove annotations from markdown (also `--for-translation`)
- `site` -- Build MkDocs site structure from ETL output (see below)

## Site builder (`steel-etl site`)

The `site` command replaces the old bash-based justfile pipeline for building the v2 MkDocs site. Config lives in `v2/site.yaml`.

```bash
steel-etl site --config v2/site.yaml
```

Features:
- **Section mapping**: copies ETL md-linked output into MkDocs tab directories (Browse, Read)
- **Book-faithful pages**: each `md-linked` page is a full book-order render of its source subtree via `RenderSubtree` (`internal/content/render_subtree.go`) → `ParsedContent.PageBody`. The `md-linked` generator emits `PageBody`; the site builder maps pages directly (no composite reassembly). Ability statblocks are un-blockquoted, headings normalized, document order preserved. `RenderSubtree` also stamps `{data-scc="<code>"}` (attr_list) onto descendant headings that have an SCC code (for the v2 per-heading permalink icons); the section→SCC map is built during the pipeline walk and `PageBody` render + writes are **deferred to a post-walk pass** so the map is complete (a parent renders before its children are classified). See `internal/pipeline/pipeline.go`.
- **Group remapping**: nests kit abilities under a "Kits" subdirectory by cross-referencing the `kit/` source directory
- **Per-book Read grouping**: when a section sets `group_by_book: true`, pages are placed under `Read/<book-folder>/` (folder/label/order from the `books:` list in `v2/site.yaml`, keyed by SCC prefix — the substring before the first `/`). Each book gets a source-ordered `.nav.yml` + `index.md`, and the section gets a landing `index.md`. Both index types are rendered as `.sc-card` grids (`cards_book.go`): the landing shows one `bookCard` per book (per-book `icon` + `description` from `site.yaml`, falling back to the generic `book` glyph), and each book index shows one `chapterCard` per chapter (shared `chapter` glyph + a blurb auto-extracted from the chapter's first prose paragraph). Chapter order comes from the `order:` frontmatter field the pipeline assigns in document order. Intra-book links are rewritten to the per-book folder.
- **Natural sort**: numeric-aware ordering in generated index pages (Level 1, 2, ... 10)
- **H1 injection**: adds `# Name` headers from frontmatter when the body lacks one
- **SCC permalink stubs**: generates `scc/{code}/index.html` redirect stubs for every page with an `scc` frontmatter field. The SCC URL is a stable, shareable redirect entry point; the friendly Browse page is the canonical, indexable location. (The client-side address-bar rewrite and its `scc-manifest.js` map were retired 2026-05-31 — see `v2/.repo-docs/decisions/2026-05-23-scc-permalink-system.md`.)
- **Search exclusion**: injects `search: exclude: true` frontmatter into Read section pages
- **Static content overrides**: copies hand-authored pages last, overriding generated content

## SCC classification

The SCC registry (`classification.json`) is generated by the pipeline and gitignored. The `classification.freeze` setting in `pipeline.yaml` controls whether existing codes can change between runs.

Use `steel-etl validate --scc-stable` to verify no codes have changed.
Use `steel-etl classify --diff` to see what would change.

## Content embedding patterns

Kits and traits can embed child abilities as structured nested objects in JSON/YAML output:

- **Kits**: `signature_ability` field — KitParser finds child `@type: ability | @subtype: signature` sections via `findSignatureAbilityChild()`
- **Traits**: `ability` field — FeatureParser finds child `@type: ability` sections via `findAbilityChild()`

Both patterns: the child ability is parsed by `AbilityParser`, stored in `ParsedContent.Children`, and embedded by the SDK transformer. The child ability also gets its own standalone output file when the pipeline walks the section tree.

Blockquote headings (`> ######`) get context-aware tree levels (previous regular heading + 1, capped at 6) so they nest as proper children of their parent sections.

## Monsters book (statblocks)

The Monsters book uses **H7 for statblocks and H9 for malice/terrain blocks** — heading levels goldmark doesn't parse (CommonMark caps at H6). `collectDeepHeadings` (`internal/parser/document.go`) captures these at level 6; **H8 is intentionally not collected** so retainer advancement sub-blocks fold into their parent statblock's body. Parsers: `monster` (group lore page + `category` context), `statblock`, `featureblock` (malice), `dynamic-terrain`, and the non-code `monster-group` container (`internal/content/monster.go`).

SCC hierarchy (nested like treasure): a group is `monster.<category>/<category>` (lore page, `monster/<category>/<category>.md`); statblocks are `monster.<category>[.<subcategory>].statblock/<id>`; malice featureblocks are `monster.<category>[.<subcategory>]/<id>` (sibling of the `statblock/` folder). `<subcategory>` is an echelon (`1st-echelon`…) for Rivals/Demons/Undead/War Dogs whose statblock names repeat per echelon. Retainers are `retainer.statblock/<id>`; terrain is `dynamic-terrain.<category>/<id>`.

Monster **group** pages are lore-only in Browse (the pipeline skips `RenderSubtree` for `@type: monster`, so reading-format generators fall back to the lore `Body`); the book-faithful everything-inline view lives on the Read tab's `chapter/monsters` page. `StatblockParser` parses the stat grid + embedded ability/trait blockquotes; `ParseStatblockFeatures` + `transformStatblock` build the SDK `statblock.schema.json` JSON with a `features[]` array.

## Architecture

See the workspace-level [ARCHITECTURE.md](../ARCHITECTURE.md) for the full pipeline data flow.

Key references:
- `ANNOTATION-GUIDE.md` -- Annotation format and conventions
- `pipeline.yaml` -- Pipeline configuration
- `../v2/site.yaml` -- Site builder configuration
- `docs/linking-guide.md` -- Rules for adding SCC cross-reference links (conditions, skills, movement, negotiation, culture disambiguation)
- `docs/linking-reference.md` -- All 441 linkable terms with display names, variants, and SCC codes
