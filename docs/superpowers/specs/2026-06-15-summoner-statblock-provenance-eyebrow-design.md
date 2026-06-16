# Summoner-book statblock provenance eyebrow

**Date:** 2026-06-15
**Status:** Designed
**Scope:** `steel-etl` only (`internal/site`)

## Problem

Summoner-book creature statblocks render a useless top-left eyebrow (the
`sb__kw` line above the name). For the rival summoner's minions (e.g. Zombie
Titan) it shows `—`; for heroic-summoner portfolio minions it shows junk (the
portfolio Zombie Titan literally shows its own name there, because those source
tables put the creature name in the first column instead of `—`). None of these
statblocks surface *where the creature comes from* — which summoner context it
belongs to, and (for rivals) at which echelon.

We want each summoner-book statblock to show a provenance label in that eyebrow
so a reader landing on the page (standalone, via SCC permalink, or embedded)
immediately knows what it is.

## Approach

The page's `scc` code already encodes everything we need — the circle
(`…summoner.undead.statblock`) and the echelon (`…rivals.4th-echelon…`). So we
derive the eyebrow purely from the `scc` frontmatter field at statblock render
time and override the `sb__kw` value (`sbIsland.Ancestry`). No tree-walking, no
post-write HTML surgery, no DOM or CSS change.

This is strictly better than the current `sb__kw` content for every affected
statblock, so replacing it (rather than adding a second line) is the right call.

### Label scheme

`·` is the separator already used by other eyebrows in the codebase
(`feature_index.go`). Echelon `4th-echelon` → `Echelon 4`; circle segment
title-cased.

| Statblock | SCC type-path (under `mcdm.summoner.v1/`) | Eyebrow |
|---|---|---|
| Rival summoner minion | `monster.rivals.{ech}.summoner.minion` | `Rival Summoner Summon · Echelon N` |
| Rival summoner (elite) | `monster.rivals.{ech}.statblock` | `Rival Summoner · Echelon N` |
| Portfolio minion | `monster.minion.summoner.{circle}.statblock` | `Summoner Minion · {Circle}` |
| Champion | `monster.champion.summoner.{circle}.statblock` | `Summoner Champion · {Circle}` |

Wording rationale:
- The rival minions read **"Summon"**, not "Minion", so the label can't be
  misread as *a minion that is a rival summoner* — it's a thing the rival
  *summons*. The elite itself stays **"Rival Summoner"** (it is the summoner).
- Portfolio minions / champions keep their `Minion` / `Champion` noun with the
  circle appended. The circle (Undead / Demon / Elemental / Fey) is the key
  differentiator and stays visible even when the page is viewed outside its
  folder.

## Implementation

### `summonerProvenanceEyebrow(scc string) string`

New helper (new file `internal/site/summoner_provenance.go`). Returns the eyebrow
label for a summoner-book statblock, or `""` for anything that doesn't match.

- Returns `""` unless `scc` carries the `mcdm.summoner.` **source** prefix. This
  is what keeps the Monsters-book `mcdm.monsters.v1/monster.rivals.{ech}.statblock`
  tree (regular, non-summoner rivals — 28 of them) untouched.
- Splits the code into `<source>/<type-path>/<item>` and matches the type-path
  against the four shapes above, extracting `{ech}` / `{circle}`.
- Echelon: parse the leading ordinal of `{N}th-echelon` → integer N → `Echelon N`.
  A segment that doesn't parse yields `""` (defensive; no partial label).
- Circle: title-case the `{circle}` segment.

### `buildStatblockIsland` (`statblock_page.go`)

After computing `Ancestry` from `keywords`, override it when the helper returns
non-empty:

```go
ancestry := strings.Join(parseFrontmatterList(fm, "keywords"), ", ")
if eb := summonerProvenanceEyebrow(parseFrontmatterField(fm, "scc")); eb != "" {
    ancestry = eb
}
```

`buildStatblockIsland` already reads `fm`, so `scc` is in hand. Because the
override is keyed on `scc`, it flows consistently to every consumer of the island
(the full `.sb-wrap` page via `buildStatblockIslandPage`, and the bestiary
preview via `bestiary_cards.go`).

## Testing

Unit tests for `summonerProvenanceEyebrow` covering:

1. Rival minion → `Rival Summoner Summon · Echelon 4`
2. Rival elite → `Rival Summoner · Echelon 4`
3. Portfolio minion → `Summoner Minion · Undead`
4. Champion → `Summoner Champion · Undead`
5. Monsters-book rival (`mcdm.monsters.v1/monster.rivals.4th-echelon.statblock`)
   → `""` (the critical non-match)
6. A non-summoner-book code and an empty string → `""`

The existing `.sb-wrap` golden-equivalence test (`TestStatblockCard_…`) covers
only non-summoner fixtures, so it is unaffected; the override never fires for
those inputs.

## Out of scope

- **Fixtures** (`monster.fixture.{circle}.featureblock`) render through the
  featureblock component, not the creature `.sb-wrap` card — different code path,
  not addressed here.
- The existing rival "Summoned by Rival Summoner" backlink (`rival_summons.go`)
  is unchanged. It's navigation; the eyebrow is identity. They coexist.
- No change to SCC codes, frontmatter, JSON/YAML data formats, or schemas — this
  is a site-render-only label derived from the existing `scc` field.
