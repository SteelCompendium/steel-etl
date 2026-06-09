# Group-Landing SCC Migration — Design

> **Status:** Approved 2026-06-09. Ready for implementation plan.

## Goal

Replace the **self-named-leaf** group-landing SCC pattern with a unified
**`<type>.group/<member>`** scheme, applied to the two code-producing group landings in one pass:

| Group landing | Today (self-named leaf) | After (unified) |
|---|---|---|
| Skill group (`@type: skill-group`) | `skill.crafting/crafting` | `skill.group/crafting` |
| Monster group (`@type: monster`) | `monster.goblins/goblins` | `monster.group/goblins` |

**Scope:** 5 skill-group codes + 51 monster-group codes = **56 codes re-minted**. Skill-leaf
codes (`skill.<g>/<item>`), statblock codes (`monster.<cat>.statblock/<id>`), and malice
featureblock codes (`monster.<cat>/<id>`) are **untouched**.

## Motivation

The self-named-leaf trick (`skill.crafting/crafting`) is a workaround for "a folder node can't
carry a code" — it doubles the group name and reads awkwardly as a permalink. The
`<type>.group/<member>` form is a cleaner expression of the same idea, and yields a tidy
enumerable bucket: `skill.group/*` = exactly the 5 skill-group landings; `monster.group/*` =
exactly the 51 monster-group landings.

These codes are days old (skills 2026-06-08, monsters 2026-06-05) with ~zero external linkage,
so re-minting now — before they calcify — is the cheapest this correction will ever be. Per the
project SCC principle (**codes are forever; divergence is reserved for backwards-compat**), this
is a young-code correction done at the right time.

## Key principle applied: code == canonical path

In this pipeline `TypePath` drives **both** the SCC code (`scc.Classify` joins it with `.`) and
the canonical output file path (`generator.go` joins it with `/`). The default state is
**alignment** — code == canonical data-repo path. We keep that alignment:

- Group `TypePath` becomes `["skill","group"]` / `["monster","group"]`, so the **code** is
  `skill.group/crafting` and the **canonical file** is `skill/group/crafting.md`. Aligned.
- We do **not** spend a code≠path divergence (e.g. an `SCCOverride` that parks the file at its
  old path) — there is no backwards-compat payoff, because the code itself is changing and the
  old permalink 404s regardless.
- The **only** thing that diverges is the **v2 Browse presentation** (`/Browse/skill/crafting/`
  ≠ code `skill.group/crafting`) — but Browse URLs are *always-already* decoupled from codes
  (the `/scc/{code}/` permalink redirect exists for exactly this). Re-organizing presentation is
  the site builder's steady-state job, not a divergence we're burning.

## Architecture

Three layers change; the SCC classifier, permalink generator, and registry-freeze machinery do
**not**.

### Layer 1 — Parsers (the code change)

**`internal/content/skill_group.go`** (`SkillGroupParser`, `@type: skill-group`):
- `TypePath`: `["skill", id]` → **`["skill", "group"]`**.
- `ItemID`: `id` (unchanged — the group id, e.g. `crafting`).
- Result: code `skill.group/crafting`, file `skill/group/crafting.md`.

**`internal/content/monster.go`** (`MonsterParser`, `@type: monster`):
- `TypePath`: `["monster", category]` → **`["monster", "group"]`**.
- `ItemID`: `category` (unchanged).
- `fm["category"]` and the pipeline's context push (which seeds `category` for descendant
  statblocks/featureblocks) are **unchanged** — descendants read `category` from context, not
  from the group page's `TypePath`. So `monster.<cat>.statblock/*` and `monster.<cat>/<id>`
  (malice) are unaffected.
- Result: code `monster.group/goblins`, file `monster/group/goblins.md`. The group lore page
  now lives in a separate `monster/group/` subtree from its statblocks (`monster/goblins/...`)
  in the canonical output — which is correct/SCC-honest.

### Layer 2 — Browse presentation (v2 site builder, site-only)

The canonical output now contains `skill/group/` and `monster/group/` directories. The site
builder reorganizes them for UX so the landing renders *at the group index*:

1. **Suppress the `group/` folder card.** In the `skill/` (and Bestiary `monster/`) folder-index
   render (`buildFolderIndex` / the subdir collection feeding it), exclude the `group` subdir so
   it does **not** appear as a navigational "Group" card alongside the real groups.
2. **Fold the landing into the group index.** When building each `skill/<g>/` group index (the
   `.sc-card` grid in `buildCardsContent`), prepend the lore/intro body from
   `skill/group/<g>.md` above the card grid, and carry that page's SCC so its permalink stub
   resolves to `/Browse/skill/<g>/`. Monster group lore folds into its Bestiary group landing
   the same way.
3. **Retire `dropSelfNamed` for skills.** There is no longer a self-named `<g>.md` inside
   `skill/<g>/` (it moved to `skill/group/<g>.md`), so the drop is unnecessary; the fold in (2)
   replaces it.
4. **Suppress the standalone `skill/group/` and `monster/group/` Browse pages** (they are
   consumed as parent indexes, not navigable on their own) — while still emitting the
   `skill.group/<g>` / `monster.group/<cat>` **permalink stubs** that redirect into Browse.

Net Browse result: `/Browse/skill/crafting/` *is* the crafting landing (lore + skill cards);
the doubled `/Browse/skill/crafting/crafting/` page disappears.

### Layer 3 — Source prose link repoint

- **`input/heroes/Draw Steel Heroes.md`:** repoint the **164** self-named group links
  `skill.<g>/<g>` → `skill.group/<g>` (crafting 26, exploration 24, interpersonal 38,
  intrigue 32, lore 44). A per-group substring replace is safe — no skill is named after its
  group, so `skill.crafting/crafting` is not a prefix of any leaf code. The 177 leaf
  `skill.<g>/<item>` links are **unchanged**.
- **`input/monsters/Draw Steel Monsters.md`:** **0** changes — there are no in-prose links to
  monster group lore pages.

## What does NOT change

- Skill-leaf codes/paths (`skill.crafting/cooking`), all 177 leaf prose links.
- Statblock codes (`monster.goblins.statblock/cutter`), malice featureblock codes
  (`monster.goblins/<id>`), retainer/terrain codes.
- The SCC classifier (`scc.Classify`), the generator path logic, the permalink-stub generator,
  the registry-freeze mechanism. `classification.freeze` is already `false`.
- The `monster-group` container (`@type: monster-group`) — still non-code-producing; it only
  seeds `domain`/`category`/`subcategory` context.

## Registry / backwards-compat

- 56 codes change; `validate --scc-stable` will flag exactly these. We accept the diff and
  regenerate `classification.json`.
- Old permalinks (`/scc/.../skill.crafting/crafting/`, `/scc/.../monster.goblins/goblins/`) will
  404. Acceptable: days-old codes, ~zero external linkage, no redirect shim warranted (YAGNI).

## Testing strategy

- **`skill_group_test.go`:** `TypePath == ["skill","group"]`, `ItemID == "crafting"`, classified
  code `…/skill.group/crafting`.
- **`monster_test.go`:** `TypePath == ["monster","group"]`, `ItemID == "goblins"`, code
  `…/monster.group/goblins`; and a descendant-context assertion that a child statblock still
  classifies under `monster.goblins.statblock/…` (group TypePath change did not leak).
- **Site-builder tests:** (a) `skill/` folder index has no `group` card; (b) `skill/crafting/`
  index contains the crafting lore above the skill `.sc-card` grid; (c) no standalone
  `/Browse/skill/crafting/crafting/` page; (d) a `skill.group/crafting` permalink stub exists.
- **Prose link guard:** assert 0 remaining `skill.<g>/<g>` occurrences and 164 `skill.group/<g>`
  occurrences in the heroes doc after the repoint.
- Full `go test ./...` (green) + a `gen --all` regen smoke check (codes + Browse render).

## Docs to update (part of "done")

- `steel-etl/CLAUDE.md` — "Grouped types (rule / skill)" + "Monsters book" sections (new
  `<type>.group/<member>` landing shape; retire the self-named-leaf description).
- Workspace `CLAUDE.md` — the SCC registry paragraph (skill-groups + monsters entries).
- This spec's implementation plan; note the prior `2026-06-08-skill-groups-nesting` and
  `2026-06-05-monsters-book` plans are superseded on the group-landing code shape.
