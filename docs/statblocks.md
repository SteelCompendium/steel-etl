# Statblocks (Monsters & Summoner books)

Deep reference for statblock parsing, SCC hierarchy, and site routing. The headline
footguns (H7/H9 headings, code≠path hoist) are summarized in `CLAUDE.md` → "Monsters
book (statblocks)"; this file holds the full detail.

## Deep headings (H7/H9)

The Monsters book uses **H7 for statblocks and H9 for malice/terrain blocks** — heading
levels goldmark doesn't parse (CommonMark caps at H6). `collectDeepHeadings`
(`internal/parser/document.go`) captures these at level 6; **H8 is intentionally not
collected** so retainer advancement sub-blocks fold into their parent statblock's body.
Those folded H8 lines (`######## Level N … Advancement Ability`) would otherwise render
as literal `########` (CommonMark caps at H6), so `RenderSubtree`'s
`demoteOverflowHeadings` rewrites any 7+-hash body line to a **bold** label.

Parsers: `monster` (group lore page + `category` context), `statblock`, `featureblock`
(malice), `dynamic-terrain`, and the non-code `monster-group` container
(`internal/content/monster.go`).

## SCC hierarchy

Nested like treasure: a group is `monster.group/<category>` (lore page; relocated to
the Bestiary group index `monster/<category>/index.md` by the site builder — see
`docs/site-builder.md` → "Group-landing relocation"); statblocks are
`monster.<category>[.<subcategory>].statblock/<id>`; malice featureblocks are
`monster.<category>[.<subcategory>]/<id>`. `<subcategory>` is an echelon
(`1st-echelon`…) for Rivals/Demons/Undead/War Dogs whose statblock names repeat per
echelon. Retainers are `retainer.statblock/<id>`; terrain is
`dynamic-terrain.<category>/<id>`.

**Note (code≠path):** the SCC codes keep their `.statblock` segment, but the **site URL
hoists `statblock/` away** (2026-06-10) so Browse pages sit directly under the group —
`monster/<group>/<id>`, `monster/<group>/<echelon>/<id>`, `retainer/<id>` (via
`hoistStatblockPath` in `internal/site/build.go`; the group-landing assembler splits
statblocks vs. featureblocks by frontmatter `type`).

## Site placement

Monster pages live on the **Browse** tab (`monster/`, `dynamic-terrain/`, `retainer/`;
moved there from the old Bestiary browser 2026-06-10 — presentation/URL only, no SCC
re-mint). The pipeline still skips `RenderSubtree` for `@type: monster`, so the lore
`Body` is the group page's prose; the **site builder** then assembles the group landing
(lore + featureblock cards + statblock preview cards) via
`internal/site/bestiary_cards.go`. The roots render `.sc-folder` cards. The
book-faithful everything-inline view lives on the Read tab's `chapter/monsters` page.
(The Bestiary tab itself is a client-side Search & Filter utility — Plan B, shipped
2026-06-10: `bestiary_search.go` emits the `.sc-bestiary-mount` data island, mounted by
`SCBestiary`; see `docs/site-builder.md` and
`docs/superpowers/specs/2026-06-10-bestiary-restructure-and-search-design.md`.)

## Parsing (stat grid, features, power rolls)

`StatblockParser` parses the stat grid + embedded ability/trait blockquotes;
`ParseStatblockFeatures` + `transformStatblock` build the SDK `statblock.schema.json`
JSON with a `features[]` array.

**Header-row layout is fixed (canonical, 5 columns):**
`| keywords | - | Level N | Org Role | EV/cost |`. The grid's label/value rows
(`**value**<br>Label`) are position-independent so `parseStatGrid` reads them anywhere,
but the **header row is positional** — keywords come from cell 0, the trailing cell is
the EV/cost. The Summoner book was originally transcribed with a different column order
(name in cell 0, keywords in cell 1, level+role combined, no `EV` prefix); that was
**corrected in the input** (`input/summoner/Draw Steel Summoner.md`) rather than teaching
the parser a second layout. Keep new statblock headers in the canonical order.

**`ev` vs `cost` (the trailing header cell).** The last column is either an Encounter
Value — written with a literal `EV` prefix (`EV 3 for 4 minions`, `EV 156`, `EV -`),
parsed into `ev` — or, first seen in the Summoner book, a **gametime summon cost** in
plain language (`3 essence for two minions`, `2 Malice for two minions`, `9 essence for
one champion`), parsed into the separate `cost` field. The distinction is deterministic
(the `EV` marker), so Monsters parse byte-for-byte as before. Both are optional strings in
`statblock.schema.json` (both copies); the v2 card renders `cost` in the EV slot without
the `EV` prefix (`.sb__cost`), and a level-less statblock (summoner minions/champions)
omits the `Level` line entirely. `Champion` is a known organization.

**Site rendering (build-time HTML, 2026-06-14).** `type: statblock` pages no longer
emit a JSON island. `buildStatblockIslandPage` (`internal/site/statblock_page.go`,
the parse stage) hands the `sbIsland` to `renderStatblockCard`
(`internal/site/statblock_card.go`), which emits the finished `.sb-wrap` DOM at build
time — the same DOM `v2/docs/javascripts/steel-statblock.js` used to build client-side
(now slimmed to wire-only: collapsible bands + sticky header). Equivalence is locked
by `TestStatblockCard_GoldenEquivalence` (golden HTML captured from the old JS
renderer; inputs under `internal/site/testdata/statblock_golden/`). This is the
`featureblock_page.go` model and unblocks embedding statblock cards inline on any page.
Design: workspace `docs/superpowers/specs/2026-06-14-statblock-build-time-render-design.md`
(cross-repo spec, so it lives at the workspace root, not under `steel-etl/`).

Power rolls come in **two forms**, both extracted to `effects.roll/tier1-3`: the
Monsters labeled form (`**Power Roll + N:**` + `- **≤11:** …` bullets) and the
**summoner dice-in-title form** (`🏹 **Name Nd10 + <char>**` followed by three bare
digit-led tier lines) — `sbDiceRe` lifts the dice from the title into `roll` and cleans
`name`, and the bare lines map to `tier1/2/3` by position.

**The statblock regexes are hardened against scc link-wrapping** (2026-06-11, when the
Summoner book became the first link-swept statblock source): `sbDiceRe` accepts a
link-wrapped characteristic (`2d10 + [R](scc:…)`) and a `linkDisplay` helper strips
link markup from the structured `roll` and from stat-grid cell values, while
tier/effect values keep their `[x](scc:…)` links verbatim (the data-field convention).

**Stat-grid cells honor escaped pipes** (`splitTableCells`, `statblock_parse.go`).
Summoner minion/fixture/champion Stamina is a multi-value cell — `**4 \| 4 \| 4**<br>Stamina`
— where `\|` is a literal pipe, not a column separator. Splitting the row naively on `|`
shattered the cell so `cellRe` never matched and Stamina rendered as `—`; the row splitter
treats `\|` as literal (unescaped to `|`) so the value survives as `4 | 4 | 4`.

**Fixture** entities use a non-standard 2-column `| **Stamina:** … | **Size:** … |`
grid plus an italic `*Hazard Support*` role line. Although they sit in `@type:
statblock` source sections, `StatblockParser` reclassifies them as **featureblocks**
when the SCC `domain == "fixture"` (Plan 5c): it returns early as `type: featureblock`
with code `monster.fixture.<element>.featureblock/<id>` (mirroring beastheart
companions, which moved into the `monster.*` family). `applyFixtureGrid`
(`internal/content/monster.go`) lifts the role line into `role`/`terrain_type` and drops
the garbage `keywords` the standard grid parse derives from the 2-column header; the
`fixtureStats` helper maps `Stamina`/`Size` into the loose `stats[]` ({name, value})
header (keeping the `+ your level` expression as the value), and `ParseRichFeatures`
populates the base (Level-0) `features[]`. The Level-5/9 advancement tiers live in a
sibling `monster.fixture.<element>.advancement-features/<id>` featureblock (source-split
into a `@type: featureblock | @id: <fixture-id>` section; `FeatureblockParser` fixture
branch). Plan: `docs/superpowers/plans/2026-06-14-fixture-featureblock-restructure.md`.

**Summoner minions / champions / rivals** are plain statblocks (no featureblock
machinery) that `StatblockParser` likewise folds into the `monster.*` family via a
`switch domain` after the normal `typePath` is built (2026-06-15):

- `domain == "minion"` → `monster.minion.summoner.<portfolio>.statblock/<id>`
- `domain == "champion"` → `monster.champion.summoner.<portfolio>.statblock/<id>`
- `domain == "rival"` → the Rival Summoner NPC (any `organization` other than
  `Minion`) → `monster.rival.<echelon>.statblock/<id>` (sits beside the Monsters-book
  rivals); its summoned creatures (`organization == Minion`) →
  `monster.rival.<echelon>.summoner.minion/<id>`. The source `@category: summoner` is
  dropped; `@subcategory` is the echelon.

The `summoner` class segment is hardcoded (these `@domain` values appear only in the
Summoner book). Site-side, `isBestiaryGroupDir` (`internal/site/bestiary_cards.go`) was
extended so the deeper `monster/<domain>/summoner/<portfolio>` portfolio dir is still
recognized as a group landing (its grandparent — not its immediate parent — is the type
root) and renders rich statblock cards. Plan:
`docs/superpowers/plans/2026-06-14-summoner-statblocks-into-monster-family.md`.

**Rival Summoner ⇄ summons cross-references (site-only).** On the v2 site, each Rival
Summoner page renders a `## Summons` card grid of its `summoner/minion/*` siblings, and
each summon links back to its conjurer via a `.sb-backlink` ("Summoned by …") line. This
is added by the `augmentRivalSummonerPages` post-write pass (`internal/site/rival_summons.go`)
purely from the on-disk Browse tree — the conjurer is the summoner-book statblock whose
`organization != Minion`, so the co-located Monsters-book rivals are skipped. No
SCC/schema/data change. See [`site-builder.md`](site-builder.md) → "Rival Summoner ⇄
summons cross-references".

## Featureblocks & dynamic terrain (structured fields)

`FeatureblockParser` and `DynamicTerrainParser` extract structured frontmatter that
validates against **`featureblock.schema.json`** (both schema copies; covers `type:
featureblock` and `type: dynamic-terrain`): `kind` (malice/feature), `level`, `flavor`,
terrain `terrain_type`/`role`, a loose ordered `stats[]` ({name, value}) for the terrain
header, and a non-lossy `features[]` array. The features are parsed by the shared
`ParseRichFeatures`/`RichFeature` in `internal/content/featureparse.go` — a port of the
statblock island's feature parser that additionally keeps the source emoji `icon` and
labeled `sections`/`enhancements` for table-less features. `transformFeatureblock`
(`internal/output/`) emits the SDK JSON/YAML straight from this frontmatter (no body
re-parse).

⚠️ **Bare prose has three position-dependent homes — `intro` / `body` / `trailing` —
and they render in different places.** The parser splits a feature's unstructured
prose by where it sits relative to the first structured block (power roll or spec
table): prose **before** it → `intro` (a test's lead-in, e.g. Pavise Shield's "As a
maneuver, … make a **Might test**.", rendered *above* the power roll); prose **after**
it → `trailing` (post-table notes, joined with spaces). A feature with **no** power
roll and no table is a plain passive, so all its prose is the `body` (rendered below
the card). Source order alone decides which field a paragraph lands in — a lead-in
mis-stored as `body` renders below the tiers instead of above (the bug fixed
2026-06-15). `intro` is in both `featureblock.schema.json` copies.

**Beastheart companion advancement-features** are a second `features[]` source. When a
`@type: featureblock` section sits in companion context (a `##### <C> Advancement
Features` block under a companion), `FeatureblockParser` takes a companion branch:
classifies as `monster.companion.<class>.advancement-features/<species>` and embeds its
**child** `@type:feature` sections (the Level-3/6/10 advancement features, which keep their
own `feature.companion.<class>.<species>.level-N/<id>` codes) into `features[]` via
`collectChildFeatures` — `{name, body, level}` per feature, render-only. Unlike malice
blocks (features from body blockquotes via `ParseRichFeatures`), these come from child
sub-sections, so the standalone card groups them into `.fb__band--adv` level tiers. Plan
5b: `docs/superpowers/plans/2026-06-13-companion-advancement-featureblocks.md`.

### Featureblock site rendering (Plan 2)

`internal/site/featureblock_page.go` turns `type: featureblock` and `type:
dynamic-terrain` pages into the `.fb-wrap` **Forged Band** card at build time. It
`yaml.Unmarshal`s the structured frontmatter produced by the parsers above (no body
re-parse), reuses the ability-card grammar (each feature becomes
`article.sc-ability.fb__feat`), and is dispatched in `build.go` `buildSection`
alongside the statblock/ability rewriters. See the design spec: workspace
`docs/superpowers/specs/2026-06-12-featureblock-cards-design.md` (cross-repo spec at
the workspace root, not under `steel-etl/`).

**Plan 2 scope:** featureblock and dynamic-terrain pages only.

### Fixture rendering (Plan 5c — `fixture_page.go` retired)

Fixtures are now `type: featureblock` (see the data-layer note above), so they render
through the **same** `buildFeatureblockPage` path as malice/terrain/companion-advancement
cards — no fixture-specific site adapter. The Plan 3 `internal/site/fixture_page.go`
statblock→fbDoc adapter (`buildFixturePage`) and the `statblock_kind: fixture` early
return in `buildStatblockIslandPage` were **deleted** in Plan 5c; the shared
`fbFeaturesFromRich` helper it owned moved to `featureblock_page.go`. `renderFbFeats`
still groups `Level > 0` features into `.fb__band--adv` bands, so the advancement-features
page renders its Level-5/9 tiers as bands.

**Site placement (Plan 5c):** `hoistStatblockPath` drops the non-leaf `featureblock/`
segment for the fixture sub-tree, so a base fixture sits at
`Browse/monster/fixture/<element>/<id>` (parallel to hoisted statblock bases), with the
sibling at `…/<element>/advancement-features/<id>`. `bestiaryItemType` indexes the base as
a searchable `"fixture"` facet (the advancement-features sibling is excluded). Plan:
`docs/superpowers/plans/2026-06-14-fixture-featureblock-restructure.md`.

**Plan 4 (retainer advancement split) — shipped.** `internal/site/retainer_page.go`
`splitRetainerAdvancement` cuts a retainer statblock body at the
`**Level N Retainer Advancement Ability**` bold-label separators (these are the H8
headings demoted by `demoteOverflowHeadings`; the regex also tolerates the `######`
heading form the md-dse-linked variant keeps). `buildStatblockIslandPage` rebuilds the
creature JSON island from the pre-advancement base (so advancement abilities no longer
pollute its feature list) and appends a single "Advancement Abilities" Forged Band card
(`renderRetainerAdvancement` → `renderFeatureblockCard`, with each tier stamped onto a
`Level > 0` `.fb__band--adv` band). Role accent + eyebrow read the per-item `role:` /
`organization:` scalars. Non-retainer statblocks are a no-op. Plan 6 (retainer rework —
own `advancement-features` codes, mirroring companions/fixtures) remains.

## Summoner book reuse

The **Summoner book** adds its own statblock-typed trees that reuse this machinery:
`minion.<portfolio>.statblock/<id>`, `champion.<portfolio>.statblock/<id>`
(demon/elemental/fey/undead portfolios), `retainer.summoner.statblock/<id>`, and
echelon-versioned `rival.summoner.<echelon>.statblock/<id>`. (Fixtures are the
exception — Plan 5c moved them out of this statblock family into
`monster.fixture.<element>.featureblock/<id>`; see "Fixture rendering" above.) They route
to the same bestiary cards
(`isBestiaryGroupDir`/`usesFolderIndex`/`hoistStatblockPath` were generalized from
`monster`-only to all statblock roots) and into the Bestiary search index
(`bestiary_search.go`). `bestiarySource`/`withSource` (`bestiary_cards.go`) mark them
**"Summoner · &lt;label&gt;"** on the card — derived from the `scc:` book prefix
(`mcdm.summoner.`), so no data/schema change — to distinguish them from Monsters-book
creatures. The mixed `retainer/` root (monster retainers + the summoner subgroup)
renders monster retainer cards plus a `Summoner` folder card.

**Statblock head eyebrow (provenance).** The `sb__kw` line above the creature
name (rendered by `statblock_card.go`) is `—` or junk for these summoner
statblocks — their source tables put `—`, "Humanoid, Rival", or the creature's
own name where `keywords` normally sit. `summonerProvenanceEyebrow`
(`summoner_provenance.go`) overrides `sbIsland.Ancestry` in `buildStatblockIsland`
with a label derived from the page's `scc` code:

| SCC type-path (under `mcdm.summoner.v1/`) | Eyebrow |
|---|---|
| `monster.rival.{ech}.summoner.minion` | `Rival Summoner Summon · Echelon N` |
| `monster.rival.{ech}.statblock` | `Rival Summoner · Echelon N` |
| `monster.minion.summoner.{circle}.statblock` | `Summoner Minion · {Circle}` |
| `monster.champion.summoner.{circle}.statblock` | `Summoner Champion · {Circle}` |

⚠️ Gated on the `mcdm.summoner.` **source** prefix so the look-alike Monsters-book
`mcdm.monsters.v1/monster.rival.{ech}.statblock` tree — which keeps its real
"Humanoid, Rival" keywords — is untouched. Like the bestiary-card label it is
`scc`-derived, so no data/schema change. Spec:
`docs/superpowers/specs/2026-06-15-summoner-statblock-provenance-eyebrow-design.md`.

The Summoner source is **fully link-swept** (2026-06-11): 1,464 inline `scc:` links
(1,292 cross-book to Heroes, 172 internal), including all 80 statblocks — the first
statblock source to be linked. Cross-book links to Heroes resolve to relative paths
that point outside the per-book `data/data-summoner` repo (the heroes pages live in
`data/data-rules`) but resolve correctly on the unified v2 site where all books share
one Browse tree. See `docs/linking-guide.md` (2026-06-11 note) and
`docs/superpowers/plans/2026-06-10-summoner-content-linking.md`.
