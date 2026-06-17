# Callout annotation & owner-based suppression

**Date:** 2026-06-17
**Status:** Designed
**Scope:** `steel-etl` only (`internal/content` render + `internal/cli` validate). No
site-builder, schema, or SCC-registry changes in Phase 1.

## Problem

Some book sources contain callouts (blockquotes) that the publisher placed wherever
whitespace happened to fall on a printed page, not because they belong to the section
they sit under. Today every blockquote in a section's body renders verbatim on every
page that section appears on, because `RenderSubtree` (`internal/content/render_subtree.go`)
emits `BodySource` as-is (it only un-blockquotes `ability` statblocks and demotes 7+-hash
overflow headings).

Concrete case: the **"Minions and Treasures"** callout in the Summoner book sits in the
body of `#### Leader Formation`
(`input/summoner/Draw Steel Summoner.md`). It is genuine rules content, but it is *not*
about Leader Formation — MCDM just had room for it there. It currently renders on:

- `feature/summoner/level-1/leader-formation.md` (Browse leaf — **unwanted**: the page is
  *about* Leader Formation, and the callout isn't),
- `class/summoner.md` (Browse class page — **wanted**: a long, book-faithful page),
- `chapter/the-summoner-class.md` (Read chapter — **wanted**: book-faithful).

So the decision is **not** "Read vs Browse." Some Browse pages (class/chapter-scale) should
show callouts exactly as the book does; only the narrow entity page that a callout
incidentally landed on should hide it.

A second class of callout *does* belong to its immediate header — e.g. a callout that
states a rule and then offers an alternative rule to override the default. Those must show
everywhere their header shows, including the narrow leaf page.

In the future we will likely want dedicated pages + SCC codes for callouts (and tables).
The design must not paint us into a corner there.

## Core principle: truth in source, policy in the renderer

The source annotation records **what the callout semantically belongs to** — never a
per-page or per-tab render decision. Views derive visibility from that truth. This keeps
the source stable as Browse behaviors, dedicated callout/table pages, and SCC codes are
added later: each becomes a new *policy* reading the same annotation, not a re-annotation
pass.

## Annotation grammar

A callout is marked with an HTML-comment annotation immediately preceding the blockquote,
matching the existing single-line annotation form
(`internal/parser/annotations.go`, `singleLineRe`):

```
<!-- @type: callout | @owner: self -->
> **Some Callout Title**
> ...

<!-- @type: callout | @owner: loose -->
> **Minions and Treasures**
> ...
```

- `@type: callout` — marks the following blockquote as a callout. Unlike every other
  `@type` value, `callout` is **not** a section-parser selector: a callout is a blockquote
  *inside a section body*, not a heading-delimited section, so it never mints an entity or
  SCC code in Phase 1. It is a **body-level render directive**.
- `@owner` — the callout's semantic owner. Binary in Phase 1:
  - `self` — belongs to the immediate enclosing header. **Always rendered**, including on
    that header's own narrow entity page.
  - `loose` — incidental; does not belong to the immediate header. **Suppressed only on the
    page that is rooted at the section the callout sits in**; rendered on every broader page.

`@owner` is **required** whenever `@type: callout` is present (see Validation). This fits
the codebase's explicit-annotation culture and makes every callout's fate self-documenting
in the source — no silent default to flip later.

### Grow-later path (NOT built in Phase 1, reserved by design)

`@owner` is an open value space, not a boolean, so it can later take:

- a coarser scope token (e.g. `@owner: chapter`) → "suppress unless the page root is the
  chapter," and/or
- an explicit SCC code / id reference once callouts become first-class entities with their
  own pages and codes.

Both are the *same seam* (compare the render root against the owner) at a different
threshold. Nothing in Phase 1 grammar or behavior needs to change to add them.

## The render rule

Every site page — Browse leaf, Browse class page, Read chapter — is a `RenderSubtree`
render **rooted at one section**. A callout living in section *C*'s body therefore renders
as part of:

- the **root body** when the page is rooted at *C* (the entity page that is *about* C), or
- a **descendant body** when the page is rooted at an ancestor of *C* (C appears deep inside
  a larger, book-faithful page).

The rule:

> A `@owner: loose` callout is stripped **only from the root body** of a subtree render.
> In any descendant body it renders normally. A `@owner: self` callout is never stripped.

This produces exactly the desired matrix for "Minions and Treasures" (`@owner: loose`):

| Page | Render root | Callout is in… | Result |
|---|---|---|---|
| `feature/summoner/level-1/leader-formation.md` | Leader Formation | root body | **stripped** |
| `feature/summoner/level-1/formation.md` (feature-group) | Formation group | descendant body | shown |
| `class/summoner.md` | class | descendant body | shown |
| `chapter/the-summoner-class.md` | chapter | descendant body | shown |

**Accepted Phase-1 coarseness:** a `loose` callout shows on *every* ancestor page, including
a mid-level feature-group page (`formation.md`), not only at class/chapter scale. This is the
explicit trade of the binary model; the grow-later targeted `@owner` refines it. The two
cases the design must get right today — narrow leaf (hide) and class/chapter (show) — are
both correct.

`@owner: self` callouts render on all of the above, leaf included.

**Untagged blockquotes are never touched** — flavor and ordinary rules blockquotes render
exactly as today. The behavior is fully opt-in.

## Implementation

### Render seam (`internal/content/render_subtree.go`)

`renderSubtree` already calls `nodeBody` for the root section and, recursively, for every
descendant. The root is identifiable because the top-level call passes
`rootLevel = section.HeadingLevel`, so `section.HeadingLevel == rootLevel` is true only for
the page root.

1. Thread an `isRoot bool` into `nodeBody`:

   ```go
   func nodeBody(section *parser.Section, isRoot bool) string {
       body := section.BodySource
       if section.Type() == "ability" {
           body = stripBlockquotePrefix(body)
       }
       if isRoot {
           body = stripLooseCallouts(body)
       }
       return demoteOverflowHeadings(body)
   }
   ```

   Call sites: the root body render (`renderSubtree` line that does `nodeBody(section)`) passes
   `isRoot = section.HeadingLevel == rootLevel`; descendant renders happen through the recursive
   `renderSubtree(child, rootLevel, …)` call, so they evaluate the same expression and naturally
   get `false`.

2. `stripLooseCallouts(body string) string` removes each `<!-- @type: callout | @owner: loose -->`
   comment **and the contiguous blockquote run that immediately follows it** (the run of lines
   beginning with `>`, allowing blank `>`-prefixed separator lines within the quote). It must:
   - tolerate annotation-key order and surrounding whitespace, mirroring `singleLineRe`
     (the live source even has a trailing space: `<!-- @type: callout --> `),
   - leave `@owner: self` callouts (and their comments) untouched,
   - strip the comment line for the matched `loose` callout along with the blockquote (no
     dangling empty comment), and collapse the blank line(s) it leaves behind so paragraph
     spacing stays clean.

   A dedicated regex/parse helper alongside `overflowHeadingRe` is appropriate; it operates
   line-wise because a blockquote is a multi-line run.

**On pages where a callout is kept**, the `<!-- @type: callout | @owner: … -->` comment is an
HTML comment and is already invisible in rendered HTML. Phase 1 leaves kept comments in place
(no behavior change, lowest risk). Stripping kept comments for tidiness is a possible later
cleanup, explicitly out of scope here.

### Validation (`internal/cli` validate)

`validate` gains a callout check that scans body sources (or the parsed comment stream) for
`@type: callout` annotations and reports, as warnings:

- a callout missing `@owner`,
- a callout whose `@owner` value is outside the known set (Phase 1: `self`, `loose`).

This is a warning, not a hard failure — consistent with `validate`'s existing
annotation-coverage reporting — so an unrecognized future value (added ahead of renderer
support) surfaces loudly without breaking the build.

## Testing

Table-driven tests in `internal/content/render_subtree_test.go`:

- `@owner: loose` callout in the **root** body → stripped; body around it intact.
- `@owner: loose` callout in a **descendant** body (render rooted at an ancestor) → kept.
- `@owner: self` callout in the root body → kept.
- Untagged blockquote in the root body → kept (regression guard).
- Comment with reordered keys and a trailing space → still matched.
- Multiple blockquotes where only the callout-tagged one is stripped, adjacent flavor
  blockquote preserved.

A focused `stripLooseCallouts` unit test covers the blockquote-run boundary (callout
followed by a blank line then a normal paragraph; callout immediately followed by another
heading; callout with internal blank `>` lines).

Validation: a small test that a `@type: callout` without `@owner` and one with a bogus
`@owner` each produce a warning.

## Non-goals (Phase 1)

- `@owner: aside`-style visual treatment / admonition boxes. The earlier per-fate model
  (`drop`/`inline`/`aside`/`page`) is **superseded** by the owner model and is not built.
- Dedicated callout/table pages or SCC codes (the grow-later identity path).
- Stripping kept callout comments from rendered output.
- Any change to the site builder, schemas, or SCC registry.
