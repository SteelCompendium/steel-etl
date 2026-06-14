# Advancement-features nav flatten + paired index cards

**Date:** 2026-06-14
**Scope:** v2 Browse navigation only (steel-etl site builder + v2 CSS). SCC codes, the
`/scc/` permalink stubs, and the `data/data-*` repos are **untouched**.

## Problem

Beastheart companions and summoner fixtures each split a base entity from its
"advancement features" across two SCC paths:

- base: `monster.companion.beastheart.statblock/<species>` /
  `monster.fixture.<element>.featureblock/<id>`
- advancement: `monster.companion.beastheart.advancement-features/<species>` /
  `monster.fixture.<element>.advancement-features/<id>`

In the v2 Browse tree the base page is **hoisted** to sit directly under its group
(`hoistStatblockPath` drops the `statblock/`/`featureblock/` segment), but the
advancement page keeps its `advancement-features/` segment, so it lands one level
deeper in a sub-folder. Result: in the left sidebar the advancement pages hide inside
an `Advancement Features` sub-folder, visually divorced from the companion they belong
to, and the index page lists them in a separate collapsible block.

Goal: every base + advancement page sits at the **same level** under its group folder,
and the group index pairs each base with its advancement features on one row.

## Non-goals

- **No SCC code changes.** The `advancement-features` segment stays in the SCC code and
  in the `/scc/.../advancement-features/<id>/` permalink. This is nav-only, consistent
  with the existing deliberate code≠path divergence (`hoistStatblockPath`).
- **No data-repo changes.** `data/data-*` output keeps the full SCC hierarchy verbatim
  (it never passes through the site builder).
- **No page-content changes.** The advancement page body, frontmatter, and embedded
  Forged-Band card are unchanged. Base and advancement remain **separate pages** (this
  is not the pending entity-embedding/compositing effort).

## Design

### Part 1 — Path flatten (`internal/site/build.go`)

Add `flattenAdvancementFeaturesPath(relPath string) string`, modeled on the adjacent
`hoistStatblockPath`:

- Scoped to the bestiary tree (`bestiaryGroupParents[parts[0]]`), same guard as the
  hoist.
- When a **non-leaf** path segment is exactly `advancement-features`, drop that segment
  and rename the leaf `<id>.md` → `<id>-advancement-features.md`.
- Examples:
  - `monster/companion/beastheart/advancement-features/wolf.md`
    → `monster/companion/beastheart/wolf-advancement-features.md`
  - `monster/fixture/demon/advancement-features/the-boil.md`
    → `monster/fixture/demon/the-boil-advancement-features.md`
- Non-matching paths returned unchanged.

The slug deliberately echoes the SCC `advancement-features` segment so the URL keeps a
textual breadcrumb back to the code.

Wire it in **two** places, exactly mirroring how `hoistStatblockPath` is applied (so
inbound cross-section links to relocated pages don't 404):

1. Dest-path computation in `buildSection` — immediately after the
   `destRel = hoistStatblockPath(destRel)` call (~line 221).
2. The inbound-link rewrite mirror in `rewriteSectionLinks` — immediately after the
   `relTarget = hoistStatblockPath(relTarget)` call (~line 966).

Order matters only in that flatten runs on a path that has already been hoisted; the two
transforms operate on disjoint segments (`statblock`/`featureblock` vs
`advancement-features`) so they compose cleanly in either order, but we apply hoist then
flatten for consistency.

The now-empty `advancement-features/` sub-directories (and their generated `index.md` /
`.nav.yml`) simply stop being produced; the site builder's clean step removes the stale
ones on the next rebuild.

### Part 2 — Paired index cards

By the time `generateIndexesRecursive` runs, the files are already flat siblings, so the
group index directory (`monster/companion/beastheart/`, each `monster/fixture/<element>/`)
contains `<id>.md` + `<id>-advancement-features.md` pairs.

A new/extended index builder emits a `.sc-cards` grid of **separate, single-link**
preview cards (the existing whole-card overlay `.sc-card` — no new card variant):

- For each base `<id>.md`, emit its card, immediately followed by its
  `<id>-advancement-features.md` card (matched by stripping the `-advancement-features`
  suffix), so the grid order is `base, advancement, base, advancement, …`.
- Card label/name comes from each page's frontmatter `name` (already "Wolf" /
  "Wolf Advancement Features"), same as today's card builders.
- A base with no advancement sibling (none today, but handle defensively) emits a normal
  single card; an orphan advancement page with no base emits its own card.
- The old `Advancement Features` sub-folder card (`.sc-folders` entry) disappears with
  the flattened folder.

To guarantee a base+advancement pair actually **shares a row** (the default `.sc-cards`
grid is `repeat(auto-fill, minmax(20rem, 1fr))` and can show 3+ columns), the grid for
these paired indexes uses a 2-column modifier — a single grid-level CSS rule in
`v2/docs/stylesheets/` (e.g. `.sc-cards--pairs { grid-template-columns: repeat(2, 1fr) }`,
collapsing to 1 column at the existing mobile breakpoint). This is a one-rule grid
modifier, **not** the heavyweight two-link card-overlay variant that was considered and
rejected.

### Left-nav ordering (minor)

After flattening, the group folder lists base + advancement as adjacent siblings (the
desired "same level" outcome). Filename sort places `<id>-advancement-features.md` just
before `<id>.md` (`-` < `.`); awesome-nav title sort places "Wolf" before "Wolf
Advancement Features". Either adjacency is acceptable; no special ordering work is
planned unless review asks for it.

## Affected directories

- `monster/companion/beastheart/` — 14 companions × (base + advancement) = 28 pages.
- `monster/fixture/{demon,elemental,fey,undead}/` — 1 base + 1 advancement each.

## Testing

- Table-driven unit tests for `flattenAdvancementFeaturesPath` (companion path, fixture
  path, non-bestiary path unchanged, leaf-named edge cases), following existing
  `build_test.go` / `permalinks_test.go` patterns.
- Index-builder test asserting paired ordering and separate single-link cards for a
  fixture directory containing a base + advancement pair.
- `go test -race ./...` green; a full `steel-etl site` build spot-checked against the v2
  Browse tree (advancement pages flattened, pairs adjacent on the index, `/scc/` stubs
  still resolve).

## Risks

- **Cross-section links to advancement pages** must follow the relocation — covered by
  wiring the transform into `rewriteSectionLinks` (Part 1, point 2). Verify no dangling
  relative `.md` links after build.
- **Permalink stubs** must still target the unchanged SCC code — they're built from
  frontmatter, not the Browse path, so they're unaffected; verify a stub resolves
  post-build.
