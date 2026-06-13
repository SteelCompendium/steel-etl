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

**Fixture** statblocks (`fixture.<portfolio>.statblock/*`) use a non-standard
2-column `| **Stamina:** … | **Size:** … |` grid plus an italic `*Hazard Support*`
role line. `StatblockParser` recognizes these when the SCC `domain == "fixture"`
(`applyFixtureGrid`, `internal/content/monster.go`): it maps `Stamina`/`Size` into the
structured fields (keeping the `+ your level` expression as the value), lifts the role
line into `role`/`terrain_type`, drops the garbage `keywords` the standard grid parse
derives from the 2-column header, and stamps `statblock_kind: fixture`. (Resolved the
former 2-column-grid gap, archived as workspace FOLLOWUPS "was #6".)

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
re-parse). Site rendering of these as High-Fantasy Steel cards is a later effort
(`docs/superpowers/specs/2026-06-12-featureblock-cards-design.md`, Plan 2+).

## Summoner book reuse

The **Summoner book** adds its own statblock-typed trees that reuse this machinery:
`minion.<portfolio>.statblock/<id>`, `fixture.<portfolio>.statblock/<id>`,
`champion.<portfolio>.statblock/<id>` (demon/elemental/fey/undead portfolios),
`retainer.summoner.statblock/<id>`, and echelon-versioned
`rival.summoner.<echelon>.statblock/<id>`. They route to the same bestiary cards
(`isBestiaryGroupDir`/`usesFolderIndex`/`hoistStatblockPath` were generalized from
`monster`-only to all statblock roots) and into the Bestiary search index
(`bestiary_search.go`). `bestiarySource`/`withSource` (`bestiary_cards.go`) mark them
**"Summoner · &lt;label&gt;"** on the card — derived from the `scc:` book prefix
(`mcdm.summoner.`), so no data/schema change — to distinguish them from Monsters-book
creatures. The mixed `retainer/` root (monster retainers + the summoner subgroup)
renders monster retainer cards plus a `Summoner` folder card.

The Summoner source is **fully link-swept** (2026-06-11): 1,464 inline `scc:` links
(1,292 cross-book to Heroes, 172 internal), including all 80 statblocks — the first
statblock source to be linked. Cross-book links to Heroes resolve to relative paths
that point outside the per-book `data/data-summoner` repo (the heroes pages live in
`data/data-rules`) but resolve correctly on the unified v2 site where all books share
one Browse tree. See `docs/linking-guide.md` (2026-06-11 note) and
`docs/superpowers/plans/2026-06-10-summoner-content-linking.md`.
