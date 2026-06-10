# Bestiary Restructure + Search Utility — Design

**Date:** 2026-06-10
**Status:** Approved (brainstorm), pending implementation plan

## Scope & decomposition

Two parts ship from this spec; a third is explicitly deferred:

- **Part A — Restructure (now):** move the browsable monster / dynamic-terrain /
  retainer content into the **Browse** tab, clean up monster-group presentation,
  and define the card **data contract**.
- **Part B-shell — Search & Filter utility (now):** repurpose the **Bestiary**
  tab into a client-side faceted finder over a build-time JSON index, with a
  clean seam for future advanced data.
- **Deferred:** card **CSS / visual design** (handed off to Claude Design
  later), and **advanced condition queries** (blocked on community-sourced
  effect data).

Guiding constraint throughout: **SCC codes are not re-minted.** Monster grouping
already exists as `monster.group/<category>` (migrated 2026-06-09); every change
here is site-presentation + URL path, honoring the code≠path principle
(`scc-code-vs-path-principle`). The Read tab is untouched — the Monsters book
keeps its book-faithful chapter form there.

## Background / current state

- `v2/site.yaml` defines three tab sections: **Browse** (modular pages by type),
  **Read** (`group_by_book`, book-faithful chapters), **Bestiary** (currently
  `include: monster/`, `dynamic-terrain/`, `retainer/` — a hierarchical browser
  that mirrors Browse).
- Monster content is already group-based: `monster/<group>/index.md` is the
  group landing (code `monster.group/<category>`), with statblocks at
  `monster/<group>/statblock/<item>` (code `monster.<group>.statblock/<item>`)
  and the group's Malice featureblock(s) as siblings.
- Four groups carry an **extra echelon nesting level**: `demons`, `undead`,
  `rivals`, `war-dogs` use `monster/<group>/<echelon>/…` (per-echelon Malice +
  statblocks), e.g. code `monster.demons.1st-echelon/…`. This asymmetry is the
  bulk of what reads as "confusing" today.
- `monster/index.md` and the group landings currently render the legacy
  `.browse-expand` / `.browse-index` list UI, **not** the rich `.sc-card` /
  `.sc-folder` cards used elsewhere in Browse.
- Statblock frontmatter is rich and filter-ready: `level`, `ev`, `role`
  (Harrier, Defender, …), `organization` (Minion, Horde, Elite, Leader, Solo,
  Retainer, …), `keywords` (Undead, Goblin, Humanoid, …), `size`, characteristics
  (`might`/`agility`/…), `movement`, `speed`, `stamina`, `free_strike`. Retainers
  add `immunities`; dynamic-terrain is sparser (`ev`, `level`, `size`,
  `direction` — no `role`/`keywords`).
- Relevant builder code: `steel-etl/internal/site/cards.go` (rich index cards),
  `build.go` (section placement, group-landing assembly, link rewriting),
  `config.go` (site.yaml schema).

## Part A — Restructure

### A1. Browse placement

Add three new **top-level Browse categories** on the Browse landing card grid,
as peers of Abilities / Ancestries / Classes / etc.: **Monsters**,
**Dynamic Terrain**, **Retainers**. Implemented by moving `monster/`,
`dynamic-terrain/`, `retainer/` from the Bestiary section's `include:` to the
Browse section's `include:` in `v2/site.yaml`, and adding their cards to
`Browse/index.md` (the hand-curated landing grid).

### A2. Monster group presentation

- **`monster/` root index** becomes a **card grid of groups** (goblins, dragons,
  demons, …), replacing today's `.browse-expand` list — reusing the folder /
  `.sc-folder` card pattern from the Skills groups.
- **Group landing** (`monster/<group>/index.md`, code `monster.group/<category>`)
  renders, in order:
  1. intro **lore** (the existing group-page prose);
  2. the group's **featureblock(s) folded inline** — the single Malice block
     (and, for Ajax, the additional Tactical Stance block). Featureblocks are
     short, so inlining gives at-a-glance value on the landing;
  3. a grid of **statblock preview cards**.
- Featureblocks **retain their own dedicated page + SCC** for direct linking;
  the inline copy on the landing is a convenience view, not a replacement.
- **Echelon groups** (`demons`, `undead`, `rivals`, `war-dogs`): statblock
  preview cards are split under small **"1st Echelon" … "4th Echelon"**
  sub-headers, and the per-echelon Malice features fold in under the lore the
  same way. No code re-mint — the existing `monster.<group>.<echelon>/…` codes
  and paths stand; only the index rendering groups cards by echelon.

### A3. Statblock paths

Keep `monster/<group>/statblock/<item>` and
`monster/<group>/<echelon>/statblock/<item>`. This is the most
code-mirrors-path-consistent option (the `statblock` segment mirrors the
`.statblock` SCC segment) and avoids URL churn.

### A4. Card data contract (data only; CSS deferred)

All fields come from **existing frontmatter** — no data-repo changes — matching
how `cards.go` already sources card data. CSS / "high-fantasy steel" visual
polish is a later Claude-Design pass; this section fixes only *which fields* a
card exposes.

- **Statblock preview card:** name, level, EV, role, organization, size,
  keywords (as chips). Optional: top characteristics. Links to the full
  statblock page.
- **Statblock full page:** unchanged content (the existing rendered statblock
  table + body). "Full card" styling is deferred.
- **Featureblock card** (Malice / Tactical Stance): name + type label + link.
- **Dynamic-terrain card:** name, level, EV, size.
- **Retainer card:** name, level, role, keywords, immunities.

### A5. steel-etl changes

- **`v2/site.yaml`:** move the three includes from Bestiary → Browse (A1); the
  Bestiary section becomes static-content-backed (see B6).
- **`internal/site/cards.go`:** register `monster`, `dynamic-terrain`, and
  `retainer` (plus the nested group/echelon shapes) as card-producing index
  types; add the group-landing assembly (lore + inline featureblock(s) +
  statblock preview cards, echelon-grouped where applicable). This generalizes
  the existing Skills-group-landing pattern.
- **Tests** alongside the existing `cards_test.go` / `build_test.go`.

## Part B-shell — Search & Filter utility

### B1. Architecture

Static-site friendly, no backend:

1. **Build-time export:** a new `steel-etl site` step walks statblock /
   dynamic-terrain / retainer frontmatter and emits a JSON index at
   `v2/docs/Bestiary/data/index.json`. Each record is keyed by **SCC code** and
   carries: `type`, `name`, `level`, `ev`, `role`, `organization`, `keywords`,
   `size`, `url` (the Browse permalink). Missing fields (e.g. terrain `role`) are
   omitted/null.
2. **Client widget:** vanilla JS on the Bestiary landing reads that JSON and
   performs client-side faceting + sorting. Widget JS/CSS live under
   `v2/overrides` / `v2/docs/javascripts`; the landing page itself is shipped via
   `static_content`.

### B2. Facets (v1)

`level` (range), `ev` (range), `role` (multi-select), `organization`
(multi-select), `keywords` (multi-select, e.g. Undead / Goblin), `size`, and a
free-text **name** search. This covers the motivating query — "undead minions in
the EV 3–6 range" — directly.

### B3. Results layout

A **dense sortable table** (sortable by EV, level, name, …); each row links to
the full Browse page for that entity. A card-grid toggle is a possible later
enhancement, **not** part of v1 — the utility framing favors scannable density.

### B4. Entity scope

A **unified** view with a **type filter** (Statblock / Dynamic Terrain /
Retainer). Facets that don't apply to a given type (e.g. `role`/`keywords` on
terrain) are simply inert for those rows; selecting such a facet effectively
narrows results to the types that carry it.

### B5. Advanced-data seam (future, not built now)

The design reserves an **optional second JSON** at
`v2/docs/Bestiary/data/conditions.json`, keyed by SCC code. If present at build
time, the widget left-joins it to enable "inflicts *poisoned*"-style facets; if
absent, v1 behaves normally. This is the clean plug for the community-sourced
effect dataset, should it become available — explicitly **out of scope for v1**.

### B6. Bestiary tab during transition

Browsable content moves to Browse immediately. Until the search widget lands,
the Bestiary tab shows a **styled "coming soon" placeholder** page (shipped via
`static_content`). Once the widget is ready, the placeholder is replaced by the
search landing.

## Cross-cutting concerns

- **No SCC re-mint.** Grouping/codes already exist; Part A is presentation + path
  only. (`scc-code-vs-path-principle`)
- **Read tab unchanged** — Monsters book stays in book-faithful chapter form;
  statblock/featureblock stylization there is a separate future effort.
- **Docs to update (part of "done"):**
  - `ARCHITECTURE.md` — new Browse categories; the JSON-export build step and its
    data flow; the Bestiary tab's change from generated browser → static search
    utility.
  - `v2/.repo-docs` — the search widget + JSON contract.
  - Workspace `CLAUDE.md` — Bestiary tab repurposing, monster-group landing
    presentation, the search data files.
- **FOLLOWUP (requested):** add a numbered `FOLLOWUPS.md` item for the **linking
  effort** — wiring monster / dynamic-terrain / retainer / featureblock pages
  into the in-prose SCC cross-reference sweep (heroes-doc style), now that they
  live in Browse.

## Out of scope / deferred

- Card CSS and "high-fantasy steel" visual polish (handed to Claude Design).
- Advanced condition/effect querying and the community effect dataset (B5 seam
  reserved, not implemented).
- Read-tab statblock/featureblock stylization.
- A card-grid toggle on the search results.
