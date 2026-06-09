# Skill Groups: Nesting Design

**Date:** 2026-06-08
**Status:** Approved (brainstorm) → ready for implementation plan

## Problem

The 57 hero skills are authored under five H5 group headings in
`input/heroes/Draw Steel Heroes.md` — Crafting, Exploration, Interpersonal,
Intrigue, Lore — but `SkillParser` ignores the group. Every skill gets a flat
SCC code `skill/<item>` and lands in a single flat `Browse/skill/` directory
rendered as one 57-card grid. The five-group taxonomy that the rulebook uses
(and that careers, cultures, classes, and domains constantly reference — *"one
skill from the interpersonal skill group"*) is invisible in the data and the
site.

This is the same shape already solved for **rules** (`rule.<group>/<term>`) and
**monster groups** (`monster.<category>/<category>`). We mirror those precedents.

### Per-group inventory

| Group | Skills |
|-------|--------|
| Crafting | 10 |
| Exploration | 10 |
| Interpersonal | 13 |
| Intrigue | 12 |
| Lore | 12 |
| **Total** | **57** |

## Decisions (settled during brainstorm)

1. **SCC shape: `skill.<group>/<item>`** (e.g. `skill.crafting/alchemy`). Group is
   a dotted *type qualifier*, mirroring `rule.<group>/<term>` — skills, like
   rules, are a flat glossary grouped one level deep. (Rejected: the
   `skill/<group>/<item>` path-segment shape used by treasure, which reads as a
   deeper hierarchy than skills actually have.)
2. **Skill groups are linkable.** The ~118 in-prose `<group> skill group`
   phrases get swept to point at each group's landing page. Each group gets a
   real SCC code via the self-named-leaf pattern: `skill.<group>/<group>`
   (e.g. `skill.crafting/crafting`), exactly like `monster.goblin/goblin`.
3. **New index pages use the redesign UI** (`.sc-folder` index-of-indexes +
   `.sc-card` per-group grids), **not** the legacy `<details>` bullet lists that
   `rule` currently falls back to.

## Design

### 1. Source annotations (`input/heroes/Draw Steel Heroes.md`)

- Each of the 57 `<!-- @type: skill | @id: x -->` comments gains `| @group: <g>`,
  where `<g>` is the enclosing H5 group, lowercase
  (`crafting`/`exploration`/`interpersonal`/`intrigue`/`lore`).
- Each of the five H5 group sections (`##### Crafting Skills`, …) — which already
  carry intro prose ("Skills from the crafting skill group are used in…") — gains
  a self-named container annotation: `<!-- @type: skill-group | @id: crafting -->`
  immediately before the heading. This produces the linkable group landing page
  `skill.crafting/crafting`.

### 2. Parsers (`internal/content/`)

- **`SkillParser`** (`skill.go`): read the optional `@group` annotation; when
  present, `TypePath = []string{"skill", group}`. Backward-compatible — no group
  ⇒ flat `["skill"]`. Direct mirror of `RuleParser`.
- **`skill-group`** container (`skill_group.go`, new — small): emits the section's
  intro prose as the page `Body`, `TypePath = []string{"skill", id}`,
  `ItemID = id` ⇒ self-named leaf `skill.<group>/<group>`. Mirrors the
  monster-group self-named-leaf in `monster.go` (`TypePath{"monster", category}`,
  `ItemID: category`). Register it in `registry.go`.
- `Classify` joins `TypePath` with `.` then appends `/ItemID`, so
  `["skill","crafting"] + "alchemy"` → `skill.crafting/alchemy` and
  `["skill","crafting"] + "crafting"` → `skill.crafting/crafting`. No classifier
  change needed.

### 3. Index rendering (`internal/site/`) — redesign UI

`buildIndexContent` dispatches: `buildCardsContent` → `buildFeatureIndexContent`
→ legacy `<details>` fallback. Two extensions, each mirroring an existing branch:

- **Skill root → folder cards.** Extend the index-of-indexes gate so the
  `Browse/skill/` landing (subdirs only) renders `.sc-folder` cards. Today the
  gate is `underFeatureOrTreasure(dir)` in `feature_index.go`; generalize it to
  also accept `skill` (rename to `usesFolderIndex` for clarity). The five folder
  cards show crest · group name · skill count · chevron, with the group intro as
  the optional folder intro blurb.
- **Per-group dir → skill cards.** Extend `buildCardsContent` so a nested skill
  leaf (`Browse/skill/<group>/`, files only) renders the existing `skill` card
  style. Exact mirror of the treasure-leaf branch:
  `if len(subdirs)==0 && len(files)>0 && pathHasSegment(dir, "skill") { cardType = "skill" }`.

The flat-skill `cardFor` "skill" case (220-char blurb budget, scroll crest) is
reused unchanged for per-group cards.

### 4. Linkable group page

The self-named `skill.<group>/<group>` leaf is an ordinary content page, so the
existing permalink-stub generator emits `scc/{code}/index.html` for it with no
extra work. It carries the group's intro prose (the "what is this group"
explainer). The per-group `index.md` is the navigational card grid. Each group
therefore has two pages with distinct jobs:

- `skill/<group>/<group>` — prose explainer, the link target for "<group> skill group".
- `skill/<group>/index.md` — auto-generated `.sc-card` grid of the group's skills,
  the folder-card destination from the root.

The `skill-group` self-named leaf is a sibling file inside its own group dir, so
the per-group card grid will also include a card for it. The per-group
`buildCardsContent` branch must **exclude the self-named `<group>.md` file** from
the skill-card grid (it is the container, not a skill).

### 5. Link sweep + docs

- **Rewrite in-prose skill links:** all 177 `scc:mcdm.heroes.v1/skill/<item>`
  occurrences → `skill.<group>/<item>`. Mechanical; each skill id maps to exactly
  one group (table built from the H5 sections).
- **New group-phrase links:** sweep the ~118 `<group> skill group` phrases →
  `[<group> skill group](scc:mcdm.heroes.v1/skill.<group>/<group>)`. Follow
  `linking-guide.md` conventions (skip the group's own defining section heading
  to avoid self-reference; skip plural "skill groups" enumerations where a single
  target is ambiguous — link only single-group phrases).
- **Docs:**
  - `docs/linking-reference.md` — re-point the 57 skill entries to their grouped
    codes; add 5 new group terms.
  - `docs/linking-guide.md` — note the skill-group code shape + the group-phrase
    rule.
  - `steel-etl/CLAUDE.md` — add `skill-group` to the parser/type list; note the
    `skill.<group>/<item>` shape.
  - workspace `CLAUDE.md` — update the SCC-registry paragraph (skills now grouped;
    new `skill-group` type) and the link-count figures.

### 6. Verification

- `steel-etl validate` (coverage, unknown types) and `classify --diff` confirm
  the new `skill.<group>/<item>` + `skill.<group>/<group>` codes and that no
  unintended codes moved.
- Build the site; eyeball: the redesigned skill root (5 folder cards), one
  per-group index (skill card grid, no stray container card), and one group
  explainer page (`skill.crafting/crafting`).
- `grep` confirms **zero** remaining flat `scc:mcdm.heroes.v1/skill/<item>` links
  and zero `Browse/skill/<item>.md` flat leaves.

## Out of scope

- Re-pointing skill links in *other* books (none reference hero skills today).
- Plural "skill groups" enumeration phrases that name two groups at once — left
  unlinked (ambiguous single target).
- Any change to the skill card visual design itself (reused as-is).

## Affected files

- `input/heroes/Draw Steel Heroes.md` (annotations + link sweeps)
- `internal/content/skill.go`, `internal/content/skill_group.go` (new),
  `internal/content/registry.go`
- `internal/site/feature_index.go` (folder-index gate), `internal/site/cards.go`
  (nested skill leaf + container exclusion)
- Tests alongside the above
- `docs/linking-reference.md`, `docs/linking-guide.md`, `steel-etl/CLAUDE.md`,
  workspace `CLAUDE.md`
