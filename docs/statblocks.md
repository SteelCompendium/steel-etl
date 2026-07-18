# Statblocks (Monsters & Summoner books)

Deep reference for statblock parsing, SCC hierarchy, and site routing. The headline
footguns (H7/H9 headings, code≠path hoist) are summarized in `CLAUDE.md` → "Monsters
book (statblocks)"; this file holds the full detail.

## Deep headings (H7/H9)

The Monsters book uses **H7 for statblocks and H9 for malice/terrain blocks** — heading
levels goldmark doesn't parse (CommonMark caps at H6). `collectDeepHeadings`
(`internal/parser/document.go`) captures these at level 6; **H8 is intentionally not
collected** — any folded H8 body line (`########` …) would otherwise render as literal
`########` (CommonMark caps at H6), so `RenderSubtree`'s `demoteOverflowHeadings` rewrites
any 7+-hash body line to a **bold** label. (Retainer/role advancement abilities used to
rely on this; Plan 6 — 2026-06-18 — instead moved their `######## Level N …` separators
into sibling `@type: featureblock` sections as **blockquote** labels `> **Level N …**`, the
form `ParseRichFeatures` collects. See "Retainers" below.)

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
echelon. Monsters-book retainers are `monster.retainer.statblock/<id>` with coded
`monster.retainer.advancement-features/<id>` + `monster.retainer.role-advancement/<role>`
container siblings (Plan 6 — see "Retainers" below); terrain is `dynamic-terrain.<category>/<id>`.

**Note (code≠path):** the SCC codes keep their `.statblock` segment, but the **site URL
hoists `statblock/` away** (2026-06-10) so Browse pages sit directly under the group —
`monster/<group>/<id>`, `monster/<group>/<echelon>/<id>`, `monster/retainer/<id>`
(via `hoistStatblockPath` in `internal/site/build.go`; the group-landing assembler splits
statblocks vs. featureblocks by frontmatter `type`).

## Site placement

Monster pages live on the **Browse** tab (`monster/` — including `monster/retainer/` since
Plan 6, which the summoner retainer folded into on 2026-06-21 — and `dynamic-terrain/`;
moved there from the old Bestiary browser 2026-06-10 — presentation/URL only). The pipeline still skips `RenderSubtree` for `@type: monster`, so the lore
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

**Keyword domains are distributed (`parseCreatureKeywords`).** Top-level commas in
cell 0 separate distinct keywords (`Abyssal, Demon` → `["Abyssal", "Demon"]`), but a
trailing parenthetical lists *domain qualifiers* on the preceding keyword and is
**distributed** — one keyword per domain — so the Summoner-book elemental cell
`Elemental (Air, Earth)` becomes `["Elemental (Air)", "Elemental (Earth)"]` (each a
separate, independently filterable Bestiary facet) instead of the naive comma-split
`"Elemental (Air"` / `"Earth)"`. The book-faithful `Elemental (Air, Earth)` display is
reconstructed for the statblock head by `collapseKeywords` (`internal/site`), which
recombines same-base entries. `splitTopLevelCommas`/`splitKeywordDomains` are the
paren-aware split helpers; the ability-table keyword cells still use plain
`splitCommaList` (ability keywords carry no domains).

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
time — the DOM `v2/docs/javascripts/steel-statblock.js` once built client-side. That
script is now **retired** (FOLLOWUPS #10.1, 2026-06-18): the collapsible Villain/Malice
bands are native `<details>`/`<summary>` and the sticky mini-header is a CSS
scroll-driven animation, so statblocks need no client JS. Equivalence is locked by
`TestStatblockCard_GoldenEquivalence` — the golden HTML was originally captured from the
old JS renderer and is now a committed snapshot of `renderStatblockCard` (regenerate
with `STEEL_UPDATE_GOLDEN=1` after an intentional DOM change; inputs under
`internal/site/testdata/statblock_golden/`). This is the
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
  `monster.rival.<echelon>.summoner.minion.statblock/<id>`. The source `@category: summoner`
  is dropped; `@subcategory` is the echelon.
- `domain == "retainer"` + `@category: summoner` → the conjurer (Devil Detective,
  `organization` other than `Minion`) → `monster.retainer.statblock/<id>` (flat-merged with
  the Monsters-book retainers); its summons (`organization == Minion`: Razor/Violent/Gorrre) →
  `monster.retainer.summoner.minion.statblock/<id>`, off the retainer index — mirroring the
  rival summons. The detective's shared advancement abilities mint one
  `monster.retainer.advancement-features/<id>` featureblock (`FeatureblockParser` retainer
  branch; the `category != "summoner"` guard was lifted 2026-06-21).

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

**Summoner Retainer ⇄ summons + advancement (site-only).** The summoner retainer (Devil
Detective) is modeled the same way (2026-06-21): `augmentSummonerRetainerPages`
(`internal/site/summoner_retainer.go`) appends a `## Advancement Features` preview card
(from the `<id>-advancement-features` sibling) and a `## Summons` grid of its
`summoner/minion/*` minions to the detective's page, plus a `Summoned by` back-link on each
minion. The `monster/retainer/` Browse index then shows only the detective's statblock
preview + its advancement-features card (the minions live in the `summoner/minion/` subtree),
rendered by the bestiary group assembler like every other retainer. No SCC/schema/data
change beyond the classification above.

**Class-owned back-links (site-only).** Beastheart companions (`monster/companion/beastheart/*`)
and summoner fixtures (`monster/fixture/*/*`) each carry a `.sb-backlink` pointing at their
owning class landing page (`class/beastheart`, `class/summoner`), on both the base page and
its `-advancement-features` sibling. Added by `augmentClassOwnedBackLinks`
(`internal/site/class_backlinks.go`), derived from the page's `scc` type-path — no
SCC/schema/data change. The summoner retainer (Devil Detective) is a different type-path
shape (`monster.retainer.*`) and is **not** covered (FOLLOWUPS #15 scope decision). See
[`site-builder.md`](site-builder.md) → "Class-owned back-links".

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

**Coded advancement members + on-page embed (2026-06-19 — ROADMAP #16).** Each fixture's
Level-5/9 advancement members are now individually coded
`feature.fixture.<element>.<base-id>.level-N/<member-id>` (×12) with their own leaf pages.
Because input headers stay faithful to the PDF (fixture group is H5) and the advancement
featureblock is a parse-**sibling** of the base — not a child — the level-6 cap forbids
nesting members under it. So members keep their `> ⭐️ **Name**` blockquote form (plus a
per-member inline `@type: feature` annotation) and the `FeatureblockParser` fixture branch
emits them as **parser-emitted coded children** (`ParsedContent.CodedChildren`,
`fixtureCodedChildren`/`fixtureMemberAnnotations` in `monster.go`); the pipeline walk **and**
`CollectSCCCodes` classify + write them as leaves. No `collectDeepHeadings`/`ContextStack`
change. The advancement card is **embedded on the base fixture page** at build time via
`embedFixtureAdvancement` (`build.go`) injecting the `{data-scc}` marker the `embed_cards`
post-pass transcludes; the group-index base+advancement pairing is **kept** (full companion
parity). Base/container codes unchanged. Spec:
`docs/superpowers/specs/2026-06-19-fixture-advancement-coded-members-design.md`.

## Retainers (Plan 6 — shipped 2026-06-18)

The Monsters-book retainers joined the `monster.*` family and their advancement/role groups
became **coded container entities** (members inline/uncoded), exactly like fixtures (5c):

- `monster.retainer.statblock/<id>` (×21) — `StatblockParser` `domain == "retainer"` branch.
  The base body keeps only the innate abilities, so `buildStatblockIslandPage` →
  `renderStatblockCard` produces the normal creature `.sb-wrap` card.
- `monster.retainer.advancement-features/<id>` (×21) and
  `monster.retainer.role-advancement/<role>` (×9) — `FeatureblockParser` `domain == "retainer"`
  branch (the role kind fires when the enclosing group carries `@category: role-advancement`).
  Members come from `ParseRichFeatures(body)`; their `Level > 0` `.fb__band--adv` tiers render
  via the shared `buildFeatureblockPage` on each entity's own paired page.

**Source shape:** each retainer's `######## Level N Retainer Advancement Ability` H8 headings
were moved into a sibling `<!-- @type: featureblock | @id: <slug> -->` `####### <Name>
Advancement Features` section and rewritten as **blockquote** labels `> **Level N …**` — the
`@id` must equal the base slug so the codes pair, and the label must be blockquoted because
`ParseRichFeatures`/`splitBlockquoteBlocks` only see `>` lines (a standalone bold line is
invisible). The 9 `##### <Role> Abilities` groups sit under a `@type: monster-group |
@domain: retainer | @category: role-advancement` container, each annotated
`@type: featureblock | @id: <role>`.

**Site:** `monster/retainer/` pairs base+advancement via `buildAdvancementPairContent` (which
tolerates the one `role-advancement/` subdir and surfaces it as a folder card) with a base-first
`.nav.yml`; `bestiaryItemType` keeps the `retainer` facet for the base and excludes the
featureblock siblings. **Plan 4's `internal/site/retainer_page.go` body split is retired.**

**Deferred (ROADMAP #15):** per-ability coding (each base/advancement/role ability its own
`feature.ability.*`) needs the header-levels rework — abilities are H8 siblings of the H7
statblock today, so they cannot nest under it as real sections.

## Summoner book reuse

The **Summoner book** adds its own statblock-typed trees that reuse this machinery:
`minion.<portfolio>.statblock/<id>`, `champion.<portfolio>.statblock/<id>`
(demon/elemental/fey/undead portfolios), the retainer family `monster.retainer.statblock/devil-detective`
(the conjurer) + `monster.retainer.summoner.minion.statblock/<id>` (its summons) +
`monster.retainer.advancement-features/devil-detective`, and echelon-versioned
`monster.rival.<echelon>.statblock/rival-summoner` (NPC) +
`monster.rival.<echelon>.summoner.minion.statblock/<id>` (summons). (Fixtures are the
exception — Plan 5c moved them out of this statblock family into
`monster.fixture.<element>.featureblock/<id>`; see "Fixture rendering" above.) They route
to the same bestiary cards
(`isBestiaryGroupDir`/`usesFolderIndex`/`hoistStatblockPath` were generalized from
`monster`-only to all statblock roots) and into the Bestiary search index
(`bestiary_search.go`). `bestiarySource`/`withSource` (`bestiary_cards.go`) mark them
**"Summoner · &lt;label&gt;"** on the card — derived from the `scc:` book prefix
(`mcdm.summoner.`), so no data/schema change — to distinguish them from Monsters-book
creatures. Plan 6 moved the Monsters-book retainers to `monster/retainer/`; the
2026-06-21 re-mint folded the summoner retainer (Devil Detective) into that same tree
(`monster.retainer.statblock/devil-detective`) and the old top-level `retainer/` root is
gone. Its summons nest under `monster/retainer/summoner/minion/` (off the index) and are
surfaced on the detective's page by `augmentSummonerRetainerPages` — so the
`monster/retainer/` landing shows the 21 Monsters-book retainers + Devil Detective + their
advancement-features cards (and the role-advancement folder), all rendered by the bestiary
group assembler.

**Statblock head provenance (`left-deck`).** The provenance line — `sbIsland.Ancestry`,
rendered as the shared header's `left-deck` slot below the creature name (`statblock_card.go`
→ `renderCardHead`; the kind-noun "Monster"/"Summon"/… is the separate `left-eyebrow`) — is
`—` or junk for these summoner statblocks: their source tables put `—`, "Humanoid, Rival",
or the creature's own name where `keywords` normally sit. `summonerProvenanceEyebrow`
(`summoner_provenance.go`) overrides `sbIsland.Ancestry` in `buildStatblockIsland`
with a label derived from the page's `scc` code:

| SCC type-path (under `mcdm.summoner.v1/`) | Eyebrow |
|---|---|
| `monster.rival.{ech}.summoner.minion.statblock` | `Rival Summoner Summon · Echelon N` |
| `monster.rival.{ech}.statblock` | `Rival Summoner · Echelon N` |
| `monster.minion.summoner.{circle}.statblock` | `Summoner Minion · {Circle}` |
| `monster.champion.summoner.{circle}.statblock` | `Summoner Champion · {Circle}` |

⚠️ Gated on the `mcdm.summoner.` **source** prefix so the look-alike Monsters-book
`mcdm.monsters.v1/monster.rival.{ech}.statblock` tree — which keeps its real
"Humanoid, Rival" keywords — is untouched. Like the bestiary-card label it is
`scc`-derived, so no data/schema change. **Exception — elemental domains:** when the
faithful keyword (`collapseKeywords`) carries a domain parenthetical, it is appended to
the provenance so the head reads like the sourcebook (`Summoner Minion · Elemental (Air,
Earth)`); only summoner elementals carry domains, so the first `(` is theirs. Spec:
`docs/superpowers/specs/2026-06-15-summoner-statblock-provenance-eyebrow-design.md`.

The Summoner source is **fully link-swept** (2026-06-11): 1,464 inline `scc:` links
(1,292 cross-book to Heroes, 172 internal), including all 80 statblocks — the first
statblock source to be linked. Cross-book links to Heroes resolve to relative paths
that point outside the per-book `data/data-summoner` repo (the heroes pages live in
`data/data-rules`) but resolve correctly on the unified v2 site where all books share
one Browse tree. See `docs/linking-guide.md` (2026-06-11 note) and
`docs/superpowers/plans/2026-06-10-summoner-content-linking.md`.

## Family Malice bands + context-driven meta-cell label (FOLLOWUPS #7 — shipped 2026-07-18)

Two pieces of the High-Fantasy Steel statblock card the design handoff marked as
non-blocking nice-to-haves are now wired in.

**Malice bands (piece 1).** Each Monsters-book statblock's own Browse leaf page carries
its family's shared Malice featureblock as a collapsible `<details>` band (open by
default, `.sb__band--malice`) — the same DOM the Villain Actions band already used
(`renderStatblockBand`/`renderStatblockFeature`, statblock_card.go). The family
featureblock keeps rendering as its own Browse page too. Implemented as a `build.go`
post-pass, **not** baked into `renderStatblockCard` alongside the villain band:
`embed_card_sections: [Browse, Read]` means a Read-tab chapter page separately embeds a
family's Malice featureblock as its own `{data-scc}` section right alongside that
family's statblocks (the sourcebook lists Malice once per family, after its
statblocks), so baking the band into the shared renderer would duplicate it once per
embedded statblock plus once standalone per Read chapter. `monster_malice.go`
pre-scans (`buildMaliceBandCache`, before any entry's raw body is transformed) for
`type: featureblock` + `kind: malice` pages, keyed by SCC type-path with any trailing
`.statblock` stripped (a statblock's type-path always ends in `.statblock`; stripping it
gives the exact type-path its family's Malice file is coded under — robust across the
per-echelon Demon/Undead/War Dog Malice tiers, which live at the SAME type-path as that
echelon's own statblocks). One family (Dragons) codes five species' Malice files under
the identical `monster.dragon` type-path (no subcategory distinguishes species there);
`augmentMonsterMaliceBands` disambiguates those by a shared filename slug
(`crucible-dragon.md` ↔ `crucible-dragon-malice.md`). The post-pass runs AFTER
`embedItemCards` (like `augmentClassOwnedBackLinks`) and only ever writes a statblock's
own canonical Browse leaf file, never a transcluded copy, so the band appears exactly
once. Retainers/companions/fixtures/every Summoner statblock have no `kind: malice`
sibling and simply get no band.

**Context-driven 4th meta cell (piece 2).** The `.sb__meta` 2×2's last cell used to be
hardcoded `"With Captain"`, and its value came from a body-prose regex that, in the real
corpus, only ever matches the single `@classify:false` illustrative example — every real
minion's captain bonus lives in the `with_captain` frontmatter field
(`ParseStatblockFields`) and was never read, so a real bonus like "+5 bonus to ranged
distance" silently rendered as "—". `statblockMeta4` (`statblock_page.go`) now derives
the cell per statblock: Summoner-book statblocks (scc source prefix `mcdm.summoner.`)
show "Free Strike Damage Type" instead — a new site-only `free_strike_damage_type`
frontmatter field (`ParseStatblockFields`, mirrors `with_captain`'s parse, deliberately
excluded from `statblockScalarKeys` so it never reaches the SDK JSON schema) — since
every classified Summoner statblock's grid carries that label, never "With Captain".
Monsters-book statblocks with a real bonus keep "With Captain"; everything else (the
cell is blank for solos/leaders/retainers) drops the cell entirely rather than showing a
meaningless blank label — `renderStatblockMeta`/`renderStatblockSticky` reflow to 3
cells. `companion_statblock.go`'s own "Skills" relabeling (a third, pre-existing variant
of this same slot) is untouched.
