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
| `internal/site/cards.go` | Rich `.sc-card` index cards for Browse-tab type indexes; `buildCardsContent` dispatch (routes bestiary leaves to `bestiary_cards.go`) |
| `internal/site/bestiary_cards.go` | Bestiary entity cards + the monster group-landing assembler |
| `internal/site/bestiary_search.go` | Bestiary tab's Search & Filter data island (`.sc-bestiary-mount`) |
| `internal/site/ability_cards.go` | Renders `type: ability` page bodies into `.sc-ability` cards; shared power-roll tier helpers |
| `internal/site/statblock_page.go` | Parses `type: statblock` page bodies → the `sbIsland` model (stat grid + blockquote features) |
| `internal/site/statblock_card.go` | Renders an `sbIsland` into the build-time `.sb-wrap` HTML card (Go port of `steel-statblock.js`) |
| `internal/site/trait_cards.go` | Renders `type: trait` page bodies into recessed `.sc-trait` niches |
| `internal/site/feature_index.go` | Folder/preview-card index pages for the nested feature/treasure/rule trees |
| `internal/site/cards_book.go` | `.sc-card` index cards for the Books tab (`bookCard`, `chapterCard`) |
| `internal/site/permalinks.go` | SCC permalink redirect-stub generator |

Per-file mechanics for everything under `internal/site/`: [`docs/site-builder.md`](docs/site-builder.md).

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

Features (full mechanics: [`docs/site-builder.md`](docs/site-builder.md)):
- **Section mapping**: copies ETL md-linked output into MkDocs tab directories (Browse, Read)
- **Book-faithful pages**: each `md-linked` page is a full book-order render of its source subtree via `RenderSubtree` (`internal/content/render_subtree.go`) → `ParsedContent.PageBody`. The `md-linked` generator emits `PageBody`; the site builder maps pages directly (no composite reassembly). Ability statblocks are un-blockquoted, headings normalized, document order preserved. `RenderSubtree` also stamps `{data-scc="<code>"}` (attr_list) onto descendant headings that have an SCC code (for the v2 per-heading permalink icons); the section→SCC map is built during the pipeline walk and `PageBody` render + writes are **deferred to a post-walk pass** so the map is complete (a parent renders before its children are classified). See `internal/pipeline/pipeline.go`.
- **Cross-section link rewriting** (`rewriteSectionLinks`): relative `.md` links are re-pointed across section boundaries **and** through the same destination-path relocations `buildSection` applies to every page (group-landing index, kit flatten, statblock hoist) — otherwise links to relocated pages 404
- **Group remapping** (kit abilities under "Kits"), **per-book Read grouping** (`group_by_book: true` → `Read/<book-folder>/` with `.sc-card` book/chapter indexes), **natural sort**, **H1 injection**, **SCC permalink stubs** (`scc/{code}/index.html` redirects), **search exclusion** for Read pages, **static content overrides** (hand-authored pages copied last)

## SCC classification

The SCC registry (`classification.json`) is generated by the pipeline and gitignored. The `classification.freeze` setting in `pipeline.yaml` controls whether existing codes can change between runs.

Use `steel-etl validate --scc-stable` to verify no codes have changed.
Use `steel-etl classify --diff` to see what would change.

⚠️ **Never put the PDF errata printing in the `book:` frontmatter.** `book:` becomes the
SCC source segment of every code minted from that document (`internal/pipeline/pipeline.go`),
so `mcdm.heroes.v1` → `mcdm.heroes.v1_01b` re-mints all ~1,915 heroes codes and dangles
~19k `scc:` links (tried and reverted 2026-06-11). The `.v1` is the SCC *namespace*
version (breaking redefinitions only); the printing is recorded in the separate `printing:` frontmatter field, which flows as a
non-identity build stamp: registry `books` map → SCC API (`index.json`/`scc.json` `books`,
per-entry `printing`) → site page frontmatter (`printing`/`printing_book`, injected by
`applyPrintingStamps` when `site.yaml` sets `registry:`; rendered by the v2 content partial).
**When ingesting a new errata printing:** update `printing:` in the book's frontmatter,
apply the content edits, then tag the commit `<book>-printing-<version>`
(e.g. `heroes-printing-1.01b`) so the exact source is recoverable via `git show`.
Design + the removal/tombstone lifecycle model:
`docs/superpowers/specs/2026-06-11-printing-provenance-and-code-lifecycle-design.md`.

**Scheme version (`scheme_version`).** As of SCC scheme **v1.1** (2026-06-09) the registry records a `scheme_version` int (default `1`) — the SCC *grammar* version, distinct from the registry-file `version`. The link resolver (`internal/scc/resolver.go`) tolerates an optional `scc.vN:` prefix on `scc:` links (bare `scc:` ≡ `scc.v1`) and a reserved trailing `#format` qualifier: it strips `#format` to the canonical identity before lookup, and resolves **only** links whose scheme version matches the registry's (`schemeVersionFromTag` vs `Registry.SchemeVersion()`) — a `scc.v2:` link against a v1 registry is reported unresolvable (a `gen`-time stderr `WARN:`, `resolver.go`) and left as plain text, never silently bound to v1 content. Format is never part of identity; see `reference/scc-specification.md` §2.0/§8/§9 and `docs/superpowers/specs/2026-06-09-scc-scheme-versioning-and-format-design.md`. Restamping bare `scc:` → explicit `scc.v1:` across inputs is deferred (workspace `FOLLOWUPS.md` #4).

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
(`SkillParser`, added 2026-06-08). The Heroes book's `rule.<group>` covers dice /
character / health / resource / combat / damage / test / downtime / negotiation /
treasure / world / general; the **Monsters book** adds its own
`rule.{monster,role,organization,keyword}` glossary (minted 2026-06-12 from the
Monster Basics chapter — see `docs/monster-rule-mapping.md`). Each skill group also has a **group-landing
page `skill.group/<group>`** (e.g. `skill.group/crafting`) emitted by the
**`skill-group`** parser (`internal/content/skill_group.go`) so prose can link to
"the <group> skill group". The container pushes no path context; child skills get
their group from their own `@group`.

The unified `<type>.group/<member>` landing shape (also used by monster groups,
`monster.group/<category>`) replaced the old self-named-leaf form
(`skill.<g>/<g>`, `monster.<cat>/<cat>`) on **2026-06-09** — see
`docs/superpowers/specs/2026-06-09-group-landing-scc-migration-design.md`.
Site-side, the landing is **relocated to the group index**
(`<root>/group/<member>.md` → `<root>/<member>/index.md`) with its intro lore folded
above the listing — mechanics in [`docs/site-builder.md`](docs/site-builder.md) →
"Group-landing relocation".

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

## Statblocks (Monsters & Summoner books)

Full reference — deep headings, SCC hierarchy, parsing, summoner reuse: [`docs/statblocks.md`](docs/statblocks.md). The headline gotchas:

- ⚠️ **H7/H9 headings.** The Monsters book uses H7 for statblocks and H9 for malice/terrain blocks (beyond CommonMark's H6 cap); `collectDeepHeadings` captures them at level 6, **H8 is deliberately not collected** (retainer advancement folds into the parent body, demoted to bold labels by `demoteOverflowHeadings`).
- ⚠️ **Code≠path.** SCC codes keep their `.statblock` segment (`monster.<category>.statblock/<id>`), but the site URL **hoists `statblock/` away** (`hoistStatblockPath`) so Browse pages sit directly under the group. Advancement-features pages additionally **flatten** beside their base entity (`flattenAdvancementFeaturesPath`: `…/advancement-features/<id>` → `…/<id>-advancement-features`) for beastheart companions + summoner fixtures — nav-only, SCC code/permalink unchanged — and the group index pairs base+advancement cards via `buildAdvancementPairContent` (see `docs/site-builder.md` → "Advancement-features flatten").
- Both the **Monsters** book (link-swept 2026-06-12) and the Summoner statblock trees (`minion`/`fixture`/`champion`/`rival`/`retainer.summoner`) are fully link-swept. The statblock parser is hardened against `scc:` link-wrapping on **every** structured field: `sbPowerRollRe` (labeled power-roll header), the title `name`/`cost`/`ability_type` split, and the ability-table cells (`keywords`/`usage`/`distance`/`target`, via `stripBold`) — effect/tier VALUES keep their links verbatim, all structured fields are stored link-free. Never link the `**Power Roll + N:**` header line or the 4-row creature stat-grid label cells in source.
- Fixtures' 2-column stat grid + italic role line are parsed (`applyFixtureGrid`, `monster.go`) into loose `stats[]` + `role`/`terrain_type`. Featureblocks/terrain emit structured `kind`/`level`/`flavor`/`stats[]`/`features[]` validated by `featureblock.schema.json` (both copies); build-time site rendering via `internal/site/featureblock_page.go` (Plan 2 — featureblock/dynamic-terrain scope); retainer-advancement split ships in Plan 4 (`internal/site/retainer_page.go` — splits the body at the `**Level N Retainer Advancement Ability**` bold labels, keeps the island base-only, appends an "Advancement Abilities" card). **Plan 5** restructures companions into `monster.companion.beastheart.*` (5a) + embeddable `monster.companion.beastheart.advancement-features/<species>` entities (5b — `FeatureblockParser` companion branch embeds the Level-3/6/10 child features via `collectChildFeatures`), and restructures **fixtures** into `monster.fixture.<element>.featureblock/<id>` + sibling `…advancement-features/<id>` (5c — `StatblockParser` reclassifies `@domain:fixture` as featureblock; **Plan 3's `fixture_page.go` adapter retired** — fixtures now render through the shared `buildFeatureblockPage`; `hoistStatblockPath` drops the `featureblock/` segment, `bestiaryItemType` indexes the base as a `"fixture"` facet). All of 5a–5c shipped. **Plan 6** = retainer rework. Spec: `docs/superpowers/specs/2026-06-13-companion-restructure-advancement-featureblocks-design.md`; 5c plan: `docs/superpowers/plans/2026-06-14-fixture-featureblock-restructure.md`. See [`docs/statblocks.md`](docs/statblocks.md).

## Architecture

See the workspace-level [ARCHITECTURE.md](../ARCHITECTURE.md) for the full pipeline data flow.

Key references:
- `ANNOTATION-GUIDE.md` -- Annotation format and conventions
- `pipeline.yaml` -- Pipeline configuration
- `../v2/site.yaml` -- Site builder configuration
- `docs/` -- Deep references, one topic per file: see [`docs/index.md`](docs/index.md). Notably `linking-guide.md` (rules for adding SCC cross-reference links), `linking-reference.md` (all linkable terms), `site-builder.md`, `statblocks.md`, `card-data-parity.md`.

## Keeping docs in sync

This file is a **router: current state + pointers only, never dated history.** Deep
subsystem detail goes in `docs/<topic>.md` (add it to `docs/index.md`); SCC
scheme/registry/linking changes get a dated entry in the workspace `docs/scc-log.md`;
per-effort plans/specs live in `docs/superpowers/`. Follow-ups and roadmap items go in
the **workspace** `FOLLOWUPS.md`/`ROADMAP.md` (this repo has none of its own). If a
section here needs a second dated sentence, it has become a log — move the entries out
and leave a summary + pointer.
