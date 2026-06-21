# Summoner Retainer modeled like the Rival Summoner

**Date:** 2026-06-21
**Status:** Design — approved, pending spec review
**Scope:** steel-etl (source annotation + parser + site builder + tests + docs)

## Problem

The Summoner book's "Retainer Summoner" section is **one** retainer — **Devil
Detective** (`organization: Retainer`) — plus three statblocks it *summons*:
**Razor** (`Signature Minion`), **Violent** (`Minion`), **Gorrre** (`Minion`).
Today's `fa288c6` re-mint classified **all four** as `monster.retainer.statblock/*`,
so the Browse → Monsters → Retainer index shows four "retainer" cards, three of
which are really the detective's summoned minions.

Separately, the detective's advancement abilities — the `######## Level 4 / 7 / 10
Retainer Advancement Ability` blocks — are authored as H8 headings. The parser does
**not** capture H8 as a featureblock; it demotes them to bold text, so they leak
inline onto whichever statblock page they happen to fall under in source order
(e.g. the Level 4 abilities currently render on **Razor's** page). There is no
shared advancement-features entity for the retainer, unlike the Monsters-book
retainers, which each carry a real `monster.retainer.advancement-features/<id>`
sibling.

## Goals

1. **Browse → Monsters → Retainer index** shows, for the summoner retainer, exactly
   two cards: the **Devil Detective** statblock preview card and its **Advancement
   Features** featureblock preview card. The three minion cards do **not** appear.
2. **Devil Detective's page** shows its statblock, an **Advancement Features**
   preview card, and a **Summons** grid of its three minions underneath — mirroring
   the Rival Summoner page. Minion pages get a "Summoned by" back-link.
3. Mirror the existing **Rival Summoner** pattern as closely as possible, and bring
   the rival summon SCC codes into line with it (see "Rival parity" below).

## Non-goals

- **Per-ability SCC coding** of the advancement abilities. The advancement-features
  entity is a single container code with members rendered inline/uncoded — exactly
  as the Monsters-book retainers do (per-ability coding stays deferred, ROADMAP #15).
- Touching the Monsters-book retainers, companions, or fixtures.
- Any schema/JSON data-contract change. All new behavior is parser classification +
  source content + site-only rendering; card content reads existing frontmatter.

## Design

### A. Minions leave the retainer index (parser / SCC)

`StatblockParser.Classify` (`internal/content/monster.go`), `retainer` branch: when
`category == "summoner"` **and** `organization == "Minion"`, classify the statblock
as a nested summon instead of a first-class retainer — directly parallel to the
existing `rival` branch's `organization == "Minion"` check.

```go
case "retainer":
    if category == "summoner" {
        if org, _ := fm["organization"].(string); org == "Minion" {
            typePath = compactPath("monster", "retainer", "summoner", "minion", "statblock")
        } else {
            typePath = compactPath("monster", "retainer", "statblock")
        }
    } else {
        typePath = compactPath("monster", "retainer", category, subcategory, "statblock")
    }
```

The detective (`organization: Retainer`) keeps `monster.retainer.statblock/devil-detective`.
The three minions move to `monster.retainer.summoner.minion.statblock/<id>`.

### B. Rival parity — add `.statblock` to the rival summons

The rival summons currently mint `monster.rival.<echelon>.summoner.minion/<id>`
(no `.statblock`). Per the decision to make every statblock code terminate in
`.statblock`, the `rival` branch gains the segment too:

```go
case "rival":
    if org, _ := fm["organization"].(string); org == "Minion" {
        typePath = compactPath("monster", "rival", subcategory, "summoner", "minion", "statblock")
    } else {
        typePath = compactPath("monster", "rival", subcategory, "statblock")
    }
```

Because `hoistStatblockPath` (`internal/site/build.go`) drops a **non-leaf**
`statblock` segment under the bestiary roots, both the retainer and rival summon
**browse URLs are unchanged** — only the SCC code gains the segment. This is the
same deliberate code≠path divergence the trees already use.

### C. Advancement Features featureblock (source annotation)

Mirror the Monsters-book retainer source pattern. Add a sibling section after Devil
Detective's statblock:

```markdown
<!-- @type: featureblock | @id: devil-detective -->
####### Devil Detective Advancement Features

> **Level 4 Retainer Advancement Ability**

> 🏹 **Soul Sleuth …**
> …

> **Level 7 Retainer Advancement Ability**
> …
```

- Convert each `######## Level N Retainer Advancement Ability` H8 heading into a
  `> **Level N Retainer Advancement Ability**` **blockquote label** (the only form
  `ParseRichFeatures` collects), keeping the ability blockquotes beneath it.
- **Reorder** the scattered L4/L7/L10 blocks out from between the minion statblocks
  into this single featureblock section, so the section order becomes: Devil
  Detective statblock → Devil Detective Advancement Features featureblock → Razor →
  Violent → Gorrre. (The ability prose that references summoning violents/gorrres is
  unchanged; only block position moves.)

This mints exactly one new code:
`mcdm.summoner.v1/monster.retainer.advancement-features/devil-detective`.

### D. Index pairing — free

`buildAdvancementPairContent` (`internal/site/advancement_pairs.go`) already pairs
`<id>.md` with `<id>-advancement-features.md` into the 2-up grid, and
`flattenAdvancementFeaturesPath` already places the advancement page beside its base
(`devil-detective-advancement-features.md`). Once the featureblock entity (C) exists
and the minions are gone from the flat dir (A), the retainer index automatically
shows the Devil Detective card paired with its Advancement Features card and nothing
else from the summoner book. The summoner retainer base card keeps its
`Summoner · Retainer` eyebrow (existing `withSource` tagging).

### E. Detective's page: advancement card + summons (site augment)

Add `augmentSummonerRetainerPages(sectionDir)` (`internal/site/`, parallel to
`augmentRivalSummonerPages`), wired in `build.go` beside the rival call. After pages
are written, for each flat statblock page under `monster/retainer/` whose `scc`
begins `mcdm.summoner.` and `organization != "Minion"` (the detective):

1. **Advancement card** — append a `## Advancement Features` section embedding the
   same preview card the index builds, reusing `card(...)` +
   `advancementCardInner(...)` over the sibling
   `<id>-advancement-features.md`. (Monsters-book retainer pages do **not** embed
   theirs today; this is new and retainer-scoped.)
2. **Summons grid** — append a `## Summons` `.sb-cards` block of the minion cards
   read from `monster/retainer/summoner/minion/`, via the existing
   `rivalSummonsCards(readDir, hrefBase, files)` helper (read dir separate from the
   href base, e.g. `hrefBase = "summoner/minion"`).
3. **Back-links** — prepend a `Summoned by <detective>` `sb-backlink` to each minion
   page above its `.sb-wrap`, exactly as the rival back-link does.

The function is idempotent (guards on `## Summons` / `sb-backlink` already present),
matching `augmentRivalSummonerPages`.

> **Simplifying assumption:** there is exactly one summoner retainer, so all files
> in `monster/retainer/summoner/minion/` belong to Devil Detective. If a second
> summoner retainer is ever added, the minion subtree would need per-retainer
> association (the rival pattern gets this for free via per-echelon dirs). Noted, not
> built.

## SCC / registry impact

| Entity | Before | After |
|---|---|---|
| Devil Detective | `monster.retainer.statblock/devil-detective` | *(unchanged)* |
| Razor | `monster.retainer.statblock/razor` | `monster.retainer.summoner.minion.statblock/razor` |
| Violent | `monster.retainer.statblock/violent` | `monster.retainer.summoner.minion.statblock/violent` |
| Gorrre | `monster.retainer.statblock/gorrre` | `monster.retainer.summoner.minion.statblock/gorrre` |
| Devil Detective advancement | *(none — H8, uncoded)* | `monster.retainer.advancement-features/devil-detective` **(new)** |
| Rival summons (×17) | `monster.rival.<ech>.summoner.minion/<id>` | `monster.rival.<ech>.summoner.minion.statblock/<id>` |

- **Browse URLs unchanged** for every row (the new `.statblock` segments are
  non-leaf and hoisted away).
- **Net registry count: +1** (the new advancement featureblock). The minion/rival
  rows are renames, not additions.
- **Link safety:** 0 inbound `scc:` links reference the rival summon codes or the
  retainer minion ids (verified by grep over `input/`), so the re-mints dangle
  nothing. The retainer minion codes were only minted earlier today (`fa288c6`).

## Testing

- `monster_test.go`: extend the retainer test — `organization: Minion` summoner
  statblock → `monster/retainer/summoner/minion/statblock`; `organization: Retainer`
  → `monster/retainer/statblock`. Update the rival minion test to expect the new
  `.statblock` segment.
- `advancement_pairs_test.go`: the existing summoner-retainer provenance test still
  passes (detective base + advancement pair, `Summoner · Retainer` eyebrow); confirm
  the minions no longer contribute index cards.
- New `augmentSummonerRetainerPages` test (model on `rival_summons_test.go`):
  detective page gains `## Advancement Features` + `## Summons` cards; minion pages
  gain a `Summoned by` back-link; idempotent on re-run.
- `validate --scc-stable` clean after a fresh `gen --all`; `classify --diff` shows
  exactly the expected renames + the one new code.

## Docs to update (part of "done")

- `CLAUDE.md` + `docs/statblocks.md` — retainer summons now nest under
  `monster.retainer.summoner.minion.statblock`; rival summons gain `.statblock`;
  the summoner retainer has a real advancement-features featureblock.
- `docs/site-builder.md` — the new `augmentSummonerRetainerPages` pass.
- `docs/summoner-linking-reference.md` — minion code table + the new advancement
  code; rival summon codes gain `.statblock`.
- Workspace `docs/scc-log.md` — dated entry for the re-mints + new code.

## Build / deploy

Implementation runs the parser + site changes, then a `gen --all` + site build to
regenerate `data/` and `v2/docs/`; deploy via the standard `just deploy-v2` flow
(regenerates and commits generated output itself — never hand-commit it).
