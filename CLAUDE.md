# steel-etl

Go CLI tool that processes annotated Draw Steel TTRPG markdown into structured, multi-format output.

## Dev environment

This project uses **devbox** for toolchain management. Go is NOT on the system PATH -- you must activate devbox first.

```bash
# From the workspace root (parent of steel-etl/):
devbox run --

# Then Go commands work:
cd steel-etl
go build ./...
go test ./...
go run ./cmd/steel-etl gen --config pipeline.yaml
```

For example, `devbox run -- go build ./...`

The `justfile` has convenience recipes that assume Go is on PATH (they work inside `devbox shell`):

```bash
just build    # Build binary
just test     # Run tests with -race
just cover    # Coverage report
just vet      # go vet
just fmt      # gofmt + goimports
just run gen --config pipeline.yaml  # Run with args
```

## Key files

| File | Purpose |
|------|---------|
| `pipeline.yaml` | Pipeline config: input, output formats, classification settings |
| `classification.json` | SCC registry (generated, gitignored) |
| `input/heroes/Draw Steel Heroes.md` | Annotated source — hand-maintained, canonical (the former `annotate_heroes.py` generator was retired; the `.md` holds ~4,055 cross-reference links and annotations that live only here) |
| `input/monsters/Draw Steel Monsters.md` | Annotated Monsters book source. Hand-maintained going forward; the initial annotation pass was bootstrapped by a since-removed `annotate_monsters.pl` (by-heading-level: H2→monster group, H7→statblock, H9→featureblock/terrain). |
| `internal/cli/*.go` | CLI commands: gen, validate, classify, strip, site |
| `internal/content/registry.go` | Content parser registry (25 parsers) |
| `internal/pipeline/pipeline.go` | Main pipeline: parse -> classify -> generate |
| `internal/scc/registry.go` | SCC registry with freeze enforcement |
| `internal/site/build.go` | Site builder: maps ETL output to MkDocs structure |
| `internal/site/config.go` | Site builder config types (sections, groups, books) |
| `internal/site/cards.go` | Rich `.sc-card` index cards for Browse-tab type indexes (kit, class, ancestry, …, plus `rule` glossary-term leaves under `rule/<group>/`, labelled by their topic group) + shared `card()`/`crestSVG`/`iconPaths`. `buildCardsContent`'s switch also routes the bestiary leaves (`statblock`/`dynamic-terrain`/`retainer` — `cardFor` dispatches to `bestiary_cards.go`). |
| `internal/site/bestiary_cards.go` | Bestiary entity cards + the monster group-landing assembler (added 2026-06-10 when the monster/terrain/retainer trees moved into Browse). Leaf cards: `statblockCard` (org+role label, Level/EV/Size/Speed), `terrainCard`, `retainerCard`. `buildMonsterGroupContent` (hooked in `buildIndexContent` after `buildFeatureIndexContent`) renders a `monster/<group>/` landing's Malice/Tactical-Stance featureblock cards + statblock preview cards, splitting demons/undead/rivals/war-dogs under `## <Echelon>` sub-headers; the group lore is folded on top by `mergeGroupLanding`. `isMonsterGroupDir` also guards `feature_index.go`'s folder branch so echelon group dirs reach this assembler. Site-only. |
| `internal/site/ability_cards.go` | `buildAbilityCardPage` dispatch + `renderAbilityCard`: rewrites each standalone `type: ability` *page body* into the raised high-fantasy steel `.sc-ability` card (crest keyed to action type, power-roll panel, Effect/Trigger panels). Hooked in `buildSection` before `injectH1`; site-only — built from the page body since frontmatter is lossy for power rolls. Styled by v2 `docs/stylesheets/steel-ability-cards.css`. |
| `internal/site/trait_cards.go` | `renderTraitCard`: rewrites each `type: trait` *page body* into the recessed `.sc-trait` "codex niche" (colored left spine, embossed heading, level pill). Rebuilds the book-faithful subtree render's H2–H6 heading tree by level (typed by `{data-scc}`: `feature.ability.*` → nested ability plate via `renderAbilityCard`; else → recursive nested sub-trait niche). Routed from `buildAbilityCardPage`; styled by v2 `docs/stylesheets/steel-traits.css`. |
| `internal/site/feature_index.go` | Index pages for the nested **feature, treasure & rule** trees (the levels between the Browse landing and the leaf cards). `buildFeatureIndexContent` (hooked in `buildIndexContent` after `buildCardsContent`): emits **folder cards** (`.sc-folder`) for index-of-indexes nodes whose children are directories (`usesFolderIndex` scopes this to feature/treasure/skill/rule + the bestiary roots monster/dynamic-terrain/retainer — monster GROUP dirs are excluded via `!isMonsterGroupDir` so they reach the group-landing assembler; e.g. the `rule/` glossary landing's 12 topic groups), and **trait/ability preview cards** (`.sc-prev`, mirroring `SCBrowse.card()` in `steel-feature-browser.js`) for parent-of-leaves nodes under `feature/trait` & `feature/ability`. Ability data is read from preserved frontmatter; trait flavor + the "Grants …" marker are parsed back out of the already-rendered `.sc-trait` HTML body. The `feature/` landing also gets the **Search & Filter** `.sc-browse-mount` JSON data island (one object per leaf, dir-URL hrefs). Site-only; styled by v2 `docs/stylesheets/steel-indexes.css`. |
| `internal/site/cards_book.go` | `.sc-card` index cards for the Books tab (`bookCard`, `chapterCard`) |
| `internal/site/permalinks.go` | SCC permalink redirect-stub generator |

## CLI commands

- `gen` -- Run full pipeline (parse + classify + output). **Processes only the primary book unless you pass `--all` (every book) or `--book <id>`.** `pipeline.yaml`'s `books:` list (beastheart, monsters) is skipped by a bare `gen`, so their `data/data-*` output goes stale; the `just deploy*` recipes pass `--all`. See `selectBookConfigs` in `internal/cli/gen.go`.
- `validate` -- Check annotation coverage, unknown types, SCC stability
- `classify` -- Display/export SCC codes, diff against registry
- `strip` -- Remove annotations from markdown (also `--for-translation`)
- `site` -- Build MkDocs site structure from ETL output (see below)

## Site builder (`steel-etl site`)

The `site` command replaces the old bash-based justfile pipeline for building the v2 MkDocs site. Config lives in `v2/site.yaml`.

```bash
steel-etl site --config v2/site.yaml
```

Features:
- **Section mapping**: copies ETL md-linked output into MkDocs tab directories (Browse, Read)
- **Book-faithful pages**: each `md-linked` page is a full book-order render of its source subtree via `RenderSubtree` (`internal/content/render_subtree.go`) → `ParsedContent.PageBody`. The `md-linked` generator emits `PageBody`; the site builder maps pages directly (no composite reassembly). Ability statblocks are un-blockquoted, headings normalized, document order preserved. `RenderSubtree` also stamps `{data-scc="<code>"}` (attr_list) onto descendant headings that have an SCC code (for the v2 per-heading permalink icons); the section→SCC map is built during the pipeline walk and `PageBody` render + writes are **deferred to a post-walk pass** so the map is complete (a parent renders before its children are classified). See `internal/pipeline/pipeline.go`.
- **Group remapping**: nests kit abilities under a "Kits" subdirectory by cross-referencing the `kit/` source directory
- **Per-book Read grouping**: when a section sets `group_by_book: true`, pages are placed under `Read/<book-folder>/` (folder/label/order from the `books:` list in `v2/site.yaml`, keyed by SCC prefix — the substring before the first `/`). Each book gets a source-ordered `.nav.yml` + `index.md`, and the section gets a landing `index.md`. Both index types are rendered as `.sc-card` grids (`cards_book.go`): the landing shows one `bookCard` per book (per-book `icon` + `description` from `site.yaml`, falling back to the generic `book` glyph), and each book index shows one `chapterCard` per chapter (shared `chapter` glyph + a blurb auto-extracted from the chapter's first prose paragraph). Chapter order comes from the `order:` frontmatter field the pipeline assigns in document order. Intra-book links are rewritten to the per-book folder.
- **Natural sort**: numeric-aware ordering in generated index pages (Level 1, 2, ... 10)
- **H1 injection**: adds `# Name` headers from frontmatter when the body lacks one
- **SCC permalink stubs**: generates `scc/{code}/index.html` redirect stubs for every page with an `scc` frontmatter field. The SCC URL is a stable, shareable redirect entry point; the friendly Browse page is the canonical, indexable location. (The client-side address-bar rewrite and its `scc-manifest.js` map were retired 2026-05-31 — see `v2/.repo-docs/decisions/2026-05-23-scc-permalink-system.md`.)
- **Search exclusion**: injects `search: exclude: true` frontmatter into Read section pages
- **Static content overrides**: copies hand-authored pages last, overriding generated content

## SCC classification

The SCC registry (`classification.json`) is generated by the pipeline and gitignored. The `classification.freeze` setting in `pipeline.yaml` controls whether existing codes can change between runs.

Use `steel-etl validate --scc-stable` to verify no codes have changed.
Use `steel-etl classify --diff` to see what would change.

**Scheme version (`scheme_version`).** As of SCC scheme **v1.1** (2026-06-09) the registry records a `scheme_version` int (default `1`) — the SCC *grammar* version, distinct from the registry-file `version`. The link resolver (`internal/scc/resolver.go`) tolerates an optional `scc.vN:` prefix on `scc:` links (bare `scc:` ≡ `scc.v1`) and a reserved trailing `#format` qualifier: it strips `#format` to the canonical identity before lookup, and resolves **only** links whose scheme version matches the registry's (`schemeVersionFromTag` vs `Registry.SchemeVersion()`) — a `scc.v2:` link against a v1 registry is reported unresolvable and left as plain text, never silently bound to v1 content. Format is never part of identity; see `reference/scc-specification.md` §2.0/§8/§9 and `docs/superpowers/specs/2026-06-09-scc-scheme-versioning-and-format-design.md`. Restamping bare `scc:` → explicit `scc.v1:` across inputs is deferred (workspace `FOLLOWUPS.md` #8).

⚠️ **A single-book `gen` accumulates codes — it does not prune.** Because the pipeline
merges into the existing registry (so a lone-book run preserves *other* books' codes),
a bare `gen` leaves orphaned entries for codes no longer emitted (e.g. after a
rename/removal). **`gen --all` handles this automatically:** it resets the registry up
front (`resetRegistryForRebuild` in `internal/cli/gen.go`) so the per-book accumulation
rebuilds a clean, orphan-free set — so every deploy (all `just deploy*` recipes pass
`--all`) self-prunes. A **frozen** registry is preserved instead (freeze means codes are
permanent; `ValidateAgainstFrozen` enforces stability). The live site + permalink stubs
are built from page frontmatter, not this list, so stale codes never reached production
regardless.

### Grouped types (rule / skill)

Some flat glossaries nest one level via an `@group` annotation that `Classify`
joins with a dot: `rule.<group>/<term>` (`RuleParser`) and **`skill.<group>/<item>`**
(`SkillParser`, added 2026-06-08). Each skill group also has a **group-landing
page `skill.group/<group>`** (e.g. `skill.group/crafting`) emitted by the
**`skill-group`** parser (`internal/content/skill_group.go`) so prose can link to
"the <group> skill group". The container pushes no path context; child skills get
their group from their own `@group`.

The unified `<type>.group/<member>` landing shape (also used by monster groups,
`monster.group/<category>`) replaced the old self-named-leaf form
(`skill.<g>/<g>`, `monster.<cat>/<cat>`) on **2026-06-09** — see
`docs/superpowers/specs/2026-06-09-group-landing-scc-migration-design.md`.
Site-side, the landing is **relocated to the group index** by `buildSection`
(`<root>/group/<member>.md` → `<root>/<member>/index.md` via `groupLandingIndexDest`),
and `mergeGroupLanding` folds its intro lore above the listing (`loreIntro` keeps
only the H1 + lead prose up to the first `## `, since a skill-group page body is a
full RenderSubtree dump of every child skill — that inline dump would double-list
against the card grid). There is **no phantom `group/` folder card** (the
relocation means `<root>/group/` never exists in Browse). The `skill/` Browse root
still renders `.sc-folder` group cards (`usesFolderIndex`) and each group dir a
`.sc-card` skill grid (`buildCardsContent`).

## Feature taxonomy (feature / ability / trait)

A **feature** is the umbrella type (`type: feature`); an **ability** is a feature plus combat rigor. `feature_type` has **three** values and the SCC path is **hub-and-spoke** (base case unmarked, specializations marked):

| `feature_type` | Home | SCC path |
|----------------|------|----------|
| `ability` | any (`@type: ability`) | `feature.ability.<entity>…` |
| `trait` | ancestry or monster only (the books that say "trait") | `feature.trait.<entity>…` |
| `feature` | class/domain/college/kit/companion/common | `feature.<entity>…` |

⚠️ **`trait` was narrowed 2026-06-07.** It used to mean *any* non-ability feature; now it is reserved for ancestry traits + monster statblock passives. `FeatureParser` emits `trait` only when the home is an ancestry (monster passives come from `statblock_parse.go`); everything else is a plain `feature`. `kit` and `companion` are **not** trait homes. See `docs/superpowers/specs/2026-06-07-feature-taxonomy-design.md` and the implementation plan alongside it.

⚠️ **Schemas live in two hand-synced copies.** `schemas/*.schema.json` here is steel-etl's own copy; the published contract is `../data-sdk-npm/src/schema/*.schema.json`. steel-etl is Go and does **not** import the npm SDK — it emits SDK-shaped JSON by convention, and `internal/output/schema_validation_test.go` validates against hand-maintained allowlists, not the schema files. **Any schema edit must land in BOTH copies.** (See workspace `ARCHITECTURE.md` → "Schemas: two hand-synced copies".)

## Content embedding patterns

Kits and non-ability features can embed child abilities as structured nested objects in JSON/YAML output:

- **Kits**: `signature_ability` field — KitParser finds child `@type: ability | @subtype: signature` sections via `findSignatureAbilityChild()`
- **Traits**: `ability` field — FeatureParser finds child `@type: ability` sections via `findAbilityChild()`

Both patterns: the child ability is parsed by `AbilityParser`, stored in `ParsedContent.Children`, and embedded by the SDK transformer. The child ability also gets its own standalone output file when the pipeline walks the section tree.

Blockquote headings (`> ######`) get context-aware tree levels (previous regular heading + 1, capped at 6) so they nest as proper children of their parent sections.

## Card ⇄ data field parity

Index/preview cards (`internal/site/cards.go`, `feature_index.go`, …) are
site-only and may read the page **body**, but the body is **not** a data
contract — only frontmatter flows into JSON/YAML + the schemas. When you upgrade
a parser to surface a new field on a card, promote it into the data formats too:
emit `fm["<field>"]` in the parser (share extractors like `firstFlavorParagraph`
in `internal/content/flavor.go`), declare it in BOTH schema copies, update the
`schema_validation_test.go` allowlist, and have the card read the field (with a
body fallback, e.g. `cardFlavor`). Full checklist: `docs/card-data-parity.md`.

## Monsters book (statblocks)

The Monsters book uses **H7 for statblocks and H9 for malice/terrain blocks** — heading levels goldmark doesn't parse (CommonMark caps at H6). `collectDeepHeadings` (`internal/parser/document.go`) captures these at level 6; **H8 is intentionally not collected** so retainer advancement sub-blocks fold into their parent statblock's body. Those folded H8 lines (`######## Level N … Advancement Ability`) would otherwise render as literal `########` (CommonMark caps at H6), so `RenderSubtree`'s `demoteOverflowHeadings` rewrites any 7+-hash body line to a **bold** label. Parsers: `monster` (group lore page + `category` context), `statblock`, `featureblock` (malice), `dynamic-terrain`, and the non-code `monster-group` container (`internal/content/monster.go`).

SCC hierarchy (nested like treasure): a group is `monster.group/<category>` (lore page; relocated to the Bestiary group index `monster/<category>/index.md` by the site builder — see the Grouped types section); statblocks are `monster.<category>[.<subcategory>].statblock/<id>`; malice featureblocks are `monster.<category>[.<subcategory>]/<id>`. `<subcategory>` is an echelon (`1st-echelon`…) for Rivals/Demons/Undead/War Dogs whose statblock names repeat per echelon. Retainers are `retainer.statblock/<id>`; terrain is `dynamic-terrain.<category>/<id>`. **Note (code≠path):** the SCC codes keep their `.statblock` segment, but the **site URL hoists `statblock/` away** (2026-06-10) so Browse pages sit directly under the group — `monster/<group>/<id>`, `monster/<group>/<echelon>/<id>`, `retainer/<id>` (via `hoistStatblockPath` in `build.go`; the group-landing assembler splits statblocks vs. featureblocks by frontmatter `type`).

Monster pages live on the **Browse** tab (`monster/`, `dynamic-terrain/`, `retainer/`; moved there from the old Bestiary browser 2026-06-10 — presentation/URL only, no SCC re-mint). The pipeline still skips `RenderSubtree` for `@type: monster`, so the lore `Body` is the group page's prose; the **site builder** then assembles the group landing (lore + featureblock cards + statblock preview cards) via `bestiary_cards.go`. The roots render `.sc-folder` cards. The book-faithful everything-inline view lives on the Read tab's `chapter/monsters` page. (The Bestiary tab itself is being repurposed into a Search & Filter utility — Plan B; see `docs/superpowers/specs/2026-06-10-bestiary-restructure-and-search-design.md`.) `StatblockParser` parses the stat grid + embedded ability/trait blockquotes; `ParseStatblockFeatures` + `transformStatblock` build the SDK `statblock.schema.json` JSON with a `features[]` array. Power rolls come in **two forms**, both extracted to `effects.roll/tier1-3`: the Monsters labeled form (`**Power Roll + N:**` + `- **≤11:** …` bullets) and the **summoner dice-in-title form** (`🏹 **Name Nd10 + <char>**` followed by three bare digit-led tier lines) — `sbDiceRe` lifts the dice from the title into `roll` and cleans `name`, and the bare lines map to `tier1/2/3` by position.

The **Summoner book** adds its own statblock-typed trees that reuse this machinery: `minion.<portfolio>.statblock/<id>`, `fixture.<portfolio>.statblock/<id>`, `champion.<portfolio>.statblock/<id>` (demon/elemental/fey/undead portfolios), `retainer.summoner.statblock/<id>`, and echelon-versioned `rival.summoner.<echelon>.statblock/<id>`. They route to the same bestiary cards (`isBestiaryGroupDir`/`usesFolderIndex`/`hoistStatblockPath` were generalized from `monster`-only to all statblock roots). `bestiarySource`/`withSource` (`bestiary_cards.go`) mark them **"Summoner · &lt;label&gt;"** on the card — derived from the `scc:` book prefix (`mcdm.summoner.`), so no data/schema change — to distinguish them from Monsters-book creatures. The mixed `retainer/` root (monster retainers + the summoner subgroup) renders monster retainer cards plus a `Summoner` folder card.

## Architecture

See the workspace-level [ARCHITECTURE.md](../ARCHITECTURE.md) for the full pipeline data flow.

Key references:
- `ANNOTATION-GUIDE.md` -- Annotation format and conventions
- `pipeline.yaml` -- Pipeline configuration
- `../v2/site.yaml` -- Site builder configuration
- `docs/linking-guide.md` -- Rules for adding SCC cross-reference links (conditions, skills, movement, negotiation, culture disambiguation)
- `docs/linking-reference.md` -- All 582 linkable terms with display names, variants, and SCC codes
