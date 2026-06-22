# Subclass eyebrow on nested sub-trait cards

**Date:** 2026-06-22
**Status:** Design approved
**Scope:** `steel-etl` only (`internal/content/render_subtree.go`, `internal/site/trait_cards.go`)

## Problem

A feature's `subclass` shows in the card eyebrow (`<Class> Feature · <Subclass>`) on
**standalone leaf pages** and on **preview/index cards**, but is **missing on class
aggregate pages** (`Browse/class/censor.md`, `conduit.md`, `elementalist.md`,
`shadow.md`, `tactician.md`, `troubadour.md`) and on Read chapter pages.

Measured gap (post-deploy generated output):

| Surface | subclass eyebrow |
|---|---|
| leaf page (`feature/censor/level-4/oracular-warning.md`) | ✓ "Censor Feature · Fate" |
| preview/index cards | ✓ |
| class aggregate page (`class/censor.md`) | ✗ (no eyebrow at all) |

`class/censor.md` showed 0/38 feature-card eyebrows with a subclass; `conduit` 0/41,
`elementalist` 0/40, etc.

## Root cause

Subclass features are grouped under a generic **container card** — "4th-Level Domain
Feature" (and the analogous order / college / doctrine / tradition / aspect / class-act
containers). The container's children render via **`renderTraitNode`**
(`internal/site/trait_cards.go`), which intentionally emits **no eyebrow** (nested
children are assumed to inherit the parent's context line). The per-child `subclass`
lives only in each child's **leaf frontmatter**, which never reaches the nested render:
the nested tree is parsed from the container body markdown (`parseTraitTree`), where the
only per-child datum carried is the `{data-scc}` heading marker (→ name + level), not the
subclass.

`renderTraitNode` is the single renderer behind the container **leaf page**, the **class
aggregate page** (which transcludes the container leaf card), and **Read chapters** — so
one fix corrects all three surfaces at once.

## Decision

Nested children that carry a subclass get the **full eyebrow, identical to the standalone
leaf** (`<Class> Feature · <Subclass>`, e.g. "Censor Feature · Fate"). Children **without**
a subclass keep their current eyebrow-less rendering — no blanket eyebrow added to every
nested sub-trait.

The class-context prefix ("Censor Feature") is **inherited from the container card's
frontmatter** (the container and its grouped options share a class); each child appends
its own `· <Subclass>`. This is exact for the grouping containers in scope (domain / order
/ college / doctrine / tradition / aspect / class-act — container and children share the
class and feature noun).

## Design

### 1. Stamp `data-subclass` on child headings — `internal/content/render_subtree.go`

In `renderSubtree`, alongside the existing `data-scc` / `data-cost` heading stamps, add a
`data-subclass="<slug>"` attr when the child section has a subclass annotation:

```go
if sub := strings.TrimSpace(child.Annotation["subclass"]); sub != "" {
    attrs = append(attrs, `data-subclass="`+sub+`"`)
}
```

`child.Annotation["subclass"]` is the authored slug (`fate`) — the same value the leaf
frontmatter carries (verified: `feature.go` reads `section.Annotation["subclass"]` into
`fm["subclass"]`). attr_list turns it into a `data-subclass` attribute on the rendered
heading; it is inert on every existing surface until `trait_cards.go` reads it.

### 2. Carry + render the subclass — `internal/site/trait_cards.go`

- Add `subclass string` to `traitNode`.
- In `parseTraitTree`, capture `data-subclass="…"` from each heading (mirroring the
  existing `traitSCCRe` capture of `data-scc`).
- `renderTraitCard` computes the inheritable eyebrow **prefix** once from the container
  frontmatter — the class + feature-noun portion of `traitEyebrow` **without** the
  subclass suffix (factor the prefix out of `traitEyebrow` so both call sites share it) —
  and threads it through `renderTraitBody` into `renderTraitNode`.
- `renderTraitNode`, when `n.subclass != ""`, emits the eyebrow
  `prefix + " · " + titleCase(strings.ReplaceAll(n.subclass, "-", " "))` via the same
  `sc-trait__eyebrow` markup `wrapTraitSection` already uses (pass the eyebrow string
  where it currently passes `""`). When `n.subclass == ""`, pass `""` (unchanged).

No CSS change: `.sc-trait__eyebrow` already styles the leaf eyebrow and applies to nested
`.sc-trait` sections identically.

## Testing

- `internal/site/trait_cards_test.go`:
  - nested node whose heading carries `{data-subclass="fate"}` under a "Censor Feature"
    container → rendered nested card contains
    `<div class="sc-trait__eyebrow">…Censor Feature · Fate</div>`.
  - nested node with no `data-subclass` → no `sc-trait__eyebrow` in that child (current
    behavior preserved).
- `internal/content/render_subtree_test.go` (or the existing subtree test): a child
  section with a `subclass` annotation → its stamped heading carries
  `{data-scc="…" data-subclass="fate"}`; a child without → no `data-subclass`.
- Regenerate and confirm `Browse/class/censor.md` shows "Censor Feature · Fate" (and the
  other domains) on the nested options, matching the leaf.

## Out of scope

- No change to standalone leaf or preview-card rendering (already correct).
- No new CSS / no chip (the user chose the eyebrow, kept as the single display vehicle).
- No change to non-subclass nested traits.
- Abilities nested under containers (`.sc-ability`, `feature.ability.*`) are a separate
  renderer and unaffected.

## Risk

Low. `data-subclass` is additive and inert except where `trait_cards.go` reads it; the
prefix-inheritance assumption holds for the in-scope grouping containers (container and
children share class + feature noun). If a future container groups children of a different
class, the prefix would be the container's — acceptable and correctable by stamping a
per-child prefix later.
