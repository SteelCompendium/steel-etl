# Site builder internals (`steel-etl site`)

Deep reference for the v2 site builder (`internal/site/`). The command overview and
config pointers live in `CLAUDE.md` → "Site builder"; this file holds the per-file
detail and the relocation/link-rewriting mechanics.

## Per-file map

### `internal/site/build.go`

Site builder entry point: maps ETL output to the MkDocs structure. Key mechanics:

- **Section mapping**: copies ETL md-linked output into MkDocs tab directories (Browse, Read).
- **Cross-section link rewriting** (`rewriteSectionLinks`): relative `.md` links are
  re-pointed across section boundaries (Browse↔Read) **and** through the same
  destination-path relocations `buildSection` applies to every page — otherwise a link
  to a page that got relocated 404s. The link *target* runs the identical
  mutually-exclusive ladder as the destination (`groupLandingIndexDest` → else
  `applyGroups` flatten/remap → then `hoistStatblockPath`), so links to a skill-group
  landing resolve to `skill/<member>/index.md`, kit-ability links to
  `feature/ability/Kits/<kit>-<ability>.md`, and statblock links drop the `statblock/`
  segment. `applyGroups` stats `<sourceDir>/<match_type>/<x>.md` to confirm a flatten
  target, so `cfg.SourceDirList()` is threaded in (each source root is tried). Fixed
  the 2026-06-11 mkdocs-warning sweep (workspace
  `docs/followups-archive/2026-06-11-completed.md`, was FOLLOWUPS #4/#14).
- **Group remapping**: nests kit abilities under a "Kits" subdirectory by
  cross-referencing the `kit/` source directory.
- **Per-book Read grouping**: when a section sets `group_by_book: true`, pages are
  placed under `Read/<book-folder>/` (folder/label/order from the `books:` list in
  `v2/site.yaml`, keyed by SCC prefix — the substring before the first `/`). Each book
  gets a source-ordered `.nav.yml` + `index.md`, and the section gets a landing
  `index.md`. Both index types are rendered as `.sc-card` grids (`cards_book.go`): the
  landing shows one `bookCard` per book (per-book `icon` + `description` from
  `site.yaml`, falling back to the generic `book` glyph), and each book index shows one
  `chapterCard` per chapter (shared `chapter` glyph + a blurb auto-extracted from the
  chapter's first prose paragraph). Chapter order comes from the `order:` frontmatter
  field the pipeline assigns in document order. Intra-book links are rewritten to the
  per-book folder.
- **Natural sort**: numeric-aware ordering in generated index pages (Level 1, 2, ... 10).
- **Nav titles**: each generated type-directory index (`generateIndexesRecursive`) also
  writes a sibling `.nav.yml` with `title: <dirToTitle(name)>`, so awesome-nav labels the
  Browse nav with the same display title as the index H1 (the pluralized `typeTitles`
  map — "Ancestries", "Careers", "Classes") instead of title-casing the singular SCC type
  slug. `title:` only; sort/other options inherit from the section-root `.nav.yml`.
- **H1 injection**: adds `# Name` headers from frontmatter when the body lacks one.
- **Search exclusion**: injects `search: exclude: true` frontmatter into Read section pages.
- **Static content overrides**: copies hand-authored pages last, overriding generated content.
- **Printing provenance stamps** (`applyPrintingStamps`, final pass — after static
  overrides so every page is covered): when `site.yaml` sets `registry:` (the
  pipeline's `classification.json`), injects non-identity `printing` /
  `printing_book` frontmatter into every page whose `scc:` book prefix has a
  recorded printing in the registry's `books` map. Rendered as a muted "Source:
  Heroes · printing 1.01b" line by `v2/overrides/partials/content.html`
  (`.sc-provenance`, styled in `v2/docs/stylesheets/extra.css`). Books without a
  `printing:` frontmatter field are skipped. No SCC/URL impact. Design:
  `docs/superpowers/specs/2026-06-11-printing-provenance-and-code-lifecycle-design.md`.

### `internal/site/cards.go`

Rich `.sc-card` index cards for Browse-tab type indexes (kit, class, ancestry, …, plus
`rule` glossary-term leaves under `rule/<group>/`, labelled by their topic group) +
shared `card()`/`crestSVG`/`iconPaths`. `buildCardsContent`'s switch also routes the
bestiary leaves (`statblock`/`dynamic-terrain`/`retainer` — `cardFor` dispatches to
`bestiary_cards.go`).

### `internal/site/bestiary_cards.go`

Bestiary entity cards + the monster group-landing assembler (added 2026-06-10 when the
monster/terrain/retainer trees moved into Browse). Leaf cards: `statblockCard`
(org+role label, Level/EV/Size/Speed), `terrainCard`, `retainerCard`.
`buildMonsterGroupContent` (hooked in `buildIndexContent` after
`buildFeatureIndexContent`) renders a `monster/<group>/` landing's
Malice/Tactical-Stance featureblock cards + statblock preview cards, splitting
demons/undead/rivals/war-dogs under `## <Echelon>` sub-headers; the group lore is
folded on top by `mergeGroupLanding`. `isBestiaryGroupDir` (generalized 2026-06-10 from
`monster`-only to all statblock roots, incl. the summoner
minion/fixture/champion/rival/retainer trees) also guards `feature_index.go`'s folder
branch so group dirs reach this assembler; `buildMonsterGroupContent` also handles the
mixed `retainer/` root (monster retainers + summoner subgroup folder card) and marks
summoner cards via `bestiarySource`/`withSource`. Site-only.

### `internal/site/bestiary_search.go`

Bestiary **Search & Filter** landing (Plan B, 2026-06-10). `collectBestiaryItems` walks
the Browse `monster`/`dynamic-terrain`/`retainer` frontmatter → one `bestiaryItem`
record each (classified by `type` + tree, since `statblock/` is hoisted away);
`buildBestiarySearchPage` (hooked in `Build()` after `generateIndexPages`) emits
`docs/Bestiary/index.md` with a `.sc-bestiary-mount` JSON data island, consumed
client-side by `v2/docs/javascripts/steel-bestiary-browser.js` (`window.SCBestiary`).
No-op when the Monsters book is absent. Site-only — no SCC re-mint, no data-repo
change. The advanced "inflicts <condition>" data seam (spec §B5) is **reserved, not
built**.

### `internal/site/ability_cards.go`

`buildAbilityCardPage` dispatch + `renderAbilityCard`: rewrites each standalone
`type: ability` *page body* into the raised high-fantasy steel `.sc-ability` card
(crest keyed to action type, power-roll panel, Effect/Trigger panels). Hooked in
`buildSection` before `injectH1`; site-only — built from the page body since
frontmatter is lossy for power rolls. Styled by v2
`docs/stylesheets/steel-ability-cards.css`. Shared tier helpers live here:
`tierPanelHTML` (emits the `.sc-ability__pr` panel; head only when a characteristic is
present) and `isTierListBlock` (the airtight ≤11/12-16/17+ bullet-triple signature — a
list block parsing into ≥2 tiers). A **header-less tier triple** (a "test" reusing the
tiers with no `**Power Roll +**` header) is detected too and rendered as a bare tier
panel (no synthesized header).

### `internal/site/statblock_page.go`

`buildStatblockIslandPage`: rewrites each `type: statblock` *page body* into a
`<script class="sc-statblock-data">` JSON island (stats from frontmatter; features
parsed from the body blockquotes — richer than the SDK statblock JSON: keeps
Effect/Trigger sections, Malice enhancements, trailing notes). Hooked in
`buildSection` next to `buildAbilityCardPage` (before `injectH1`). The island is
mounted client-side by `v2/docs/javascripts/steel-statblock.js` (`window.SCStatblock`)
into the High-Fantasy Steel `.sb-wrap` DOM styled by
`v2/docs/stylesheets/steel-statblock.css` — per the approved 2026-06-11 design handoff,
archived at workspace `reference/design-system/handoff/redesign/statblocks/README.md`
(DOM + `data-sb-*` preference contract). Reuses ability-card body helpers
(`parseAbilityTable`/`parseTiers`/`prHeadRe`/`labelRe`/`paraSplitRe`) + `cardHref`
(link-target resolution). Handles labeled tiers (Monsters) and dice-in-title/bare
tiers (Summoner). Family Malice band deferred (workspace `FOLLOWUPS.md` #7). Site-only.

### `internal/site/trait_cards.go`

`renderTraitCard`: rewrites each `type: trait` *page body* into the recessed
`.sc-trait` "codex niche" (colored left spine, embossed heading, level pill). Rebuilds
the book-faithful subtree render's H2–H6 heading tree by level (typed by `{data-scc}`:
`feature.ability.*` → nested ability plate via `renderAbilityCard`; else → recursive
nested sub-trait niche). A body block matching `isTierListBlock` renders via the shared
`tierPanelHTML` (the `tiers` `classifyTraitBlock` kind) instead of a plain `<ul>`, so a
plain feature's embedded test (e.g. the Summoner's Fairy Whispers) shows the
glyph-badged power-roll panel. Routed from `buildAbilityCardPage`; styled by v2
`docs/stylesheets/steel-traits.css`.

### `internal/site/feature_index.go`

Index pages for the nested **feature, treasure & rule** trees (the levels between the
Browse landing and the leaf cards). `buildFeatureIndexContent` (hooked in
`buildIndexContent` after `buildCardsContent`): emits **folder cards** (`.sc-folder`)
for index-of-indexes nodes whose children are directories (`usesFolderIndex` scopes
this to feature/treasure/skill/rule + the bestiary roots
monster/dynamic-terrain/retainer — bestiary GROUP dirs are excluded via
`!isBestiaryGroupDir` so they reach the group-landing assembler; e.g. the `rule/`
glossary landing's 12 topic groups), and **feature/trait/ability preview cards**
(`.sc-prev`, mirroring `SCBrowse.card()` in `steel-feature-browser.js`) for
parent-of-leaves nodes. Ability data is read from preserved frontmatter; trait/feature
flavor + the "Grants …" marker are parsed back out of the already-rendered `.sc-trait`
HTML body. The recessed `.sc-trait` niche is **shared** by plain features (`type:
feature`) and the narrowed ancestry/monster traits (`type: trait`), but the eyebrow
*noun* reflects the real type — `featureNoun` (`trait_cards.go`) renders "<Source>
Feature" vs "<Source> Trait" (regression once mislabelled every feature "Trait"). The
`feature/` landing also gets the **Search & Filter** `.sc-browse-mount` JSON data island
(one object per leaf, dir-URL hrefs; `kind` ∈ feature/ability/trait drives the Type
facet's three buckets). Site-only; styled by v2 `docs/stylesheets/steel-indexes.css`.

### `internal/site/cards_book.go`

`.sc-card` index cards for the Books tab (`bookCard`, `chapterCard`).

### `internal/site/permalinks.go`

SCC permalink redirect-stub generator: emits `scc/{code}/index.html` redirect stubs for
every page with an `scc` frontmatter field. The SCC URL is a stable, shareable redirect
entry point; the friendly Browse page is the canonical, indexable location. (The
client-side address-bar rewrite and its `scc-manifest.js` map were retired 2026-05-31 —
see `v2/.repo-docs/decisions/2026-05-23-scc-permalink-system.md`.)

## Group-landing relocation (rule / skill / monster groups)

The unified `<type>.group/<member>` landing shape (`skill.group/<group>`,
`monster.group/<category>`) replaced the old self-named-leaf form on **2026-06-09** —
see `docs/superpowers/specs/2026-06-09-group-landing-scc-migration-design.md`.
Site-side, the landing is **relocated to the group index** by `buildSection`
(`<root>/group/<member>.md` → `<root>/<member>/index.md` via `groupLandingIndexDest`),
and `mergeGroupLanding` folds its intro lore above the listing (`loreIntro` keeps only
the H1 + lead prose up to the first `## `, since a skill-group page body is a full
RenderSubtree dump of every child skill — that inline dump would double-list against
the card grid). There is **no phantom `group/` folder card** (the relocation means
`<root>/group/` never exists in Browse). The `skill/` Browse root still renders
`.sc-folder` group cards (`usesFolderIndex`) and each group dir a `.sc-card` skill grid
(`buildCardsContent`).

## Advancement-features flatten (companions / fixtures)

`flattenAdvancementFeaturesPath` (`build.go`) collapses a non-leaf
`advancement-features/` folder in the bestiary tree, folding its name into the leaf
(`…/advancement-features/<id>.md` → `…/<id>-advancement-features.md`) so a
beastheart-companion or summoner-fixture advancement page sits beside its base entity
instead of in a sub-folder. Like `hoistStatblockPath` it is a deliberate **code≠path**
divergence — the SCC code keeps its `.advancement-features` segment; only the Browse
URL/sidebar changes (the `/scc/…advancement-features/<id>/` permalink stub still exists,
now redirecting to the flattened page) — and it is applied in the **same two places**:
the dest-path computation in `buildSection` (right after `hoistStatblockPath`) and the
inbound-link mirror in `rewriteSectionLinks`. The flattened group dir is then rendered
by `buildAdvancementPairContent` (`advancement_pairs.go`) as a 2-up
`.sc-cards.sc-cards--pairs` grid pairing each base card (eyebrow "Companion"/`paw` or
"Fixture"/`skull`) with its advancement card (eyebrow "Advancement Features", sharing the
base's name), base-first; it is the **first** check in `buildIndexContent` so it
intercepts these dirs before the bestiary group-landing / plain-list builders. Styled by
`v2/docs/stylesheets/steel-redesign.css` (`.sc-cards--pairs`). The left-sidebar order is
pinned base-first to match: `advancementPairNavOrder` makes `generateIndexesRecursive`
write an explicit `nav:` list (`index.md`, base, advancement, …) into the dir's `.nav.yml`,
otherwise the advancement page (`<id>-advancement-features.md`) filename-sorts ahead of its
base (`<id>.md`).
