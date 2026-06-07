# steel-etl

Go CLI tool that processes annotated Draw Steel TTRPG markdown into structured, multi-format output for the [Steel Compendium](https://steelcompendium.io).

**Status:** Phases 0-5 complete (3.6 i18n deferred). Three books integrated — heroes, beastheart, monsters (**2,627 SCC codes**). All 9 output generators, 24 content parsers, `validate`/`classify`/`gen`/`strip`/`site` CLI commands implemented. The SCC registry is currently **unfrozen** (`freeze: false`) while multi-book content is still being added; existing codes are stable, and re-freezing is a flip of `classification.freeze` once content settles.

## What it does

```
Annotated Markdown ──► steel-etl ──► Structured Output
                                      ├── md/          (per-section markdown + frontmatter)
                                      ├── md-linked/   (cross-referenced)
                                      ├── json/        (SDK-compatible)
                                      ├── yaml/
                                      ├── md-dse/      (Obsidian Draw Steel Elements)
                                      └── md-dse-linked/
```

The source is one annotated markdown file per book (`Draw Steel Heroes.md`, `Draw Steel Beastheart.md`, `Draw Steel Monsters.md`) with HTML comment annotations that tell the parser what each section is. The tool parses each in a single pass and produces per-section output files in multiple formats with full YAML frontmatter. A bare `gen` processes only the primary book; pass `--all` (every book) or `--book <id>` for the rest.

## Project layout

```
steel-etl/
├── cmd/steel-etl/main.go         # CLI entrypoint
├── internal/
│   ├── cli/                       # Cobra commands (gen, validate, classify, strip, site)
│   ├── parser/                    # Markdown parser: annotations, document, sections
│   ├── context/                   # Hierarchical annotation context stack
│   ├── content/                   # 24 content parsers (ability, class, kit, statblock, monster, etc.)
│   ├── scc/                       # SCC classifier and registry
│   ├── output/                    # 9 output generators (md, json, yaml, linked, dse, etc.)
│   ├── pipeline/                  # Orchestrates parse → classify → generate
│   └── site/                      # MkDocs site builder from steel-etl output
├── testdata/fixtures/             # Test fixtures (simple_class.md)
├── pipeline.yaml                  # Pipeline configuration (heroes + beastheart + monsters)
├── Makefile                       # Build, test, lint targets
├── scripts/                       # Reusable SCC link-audit/linking tooling (link_audit*.py, link_apply.py)
└── input/                         # Annotated sources (hand-maintained, canonical)
    ├── heroes/Draw Steel Heroes.md
    ├── beastheart/Draw Steel Beastheart.md
    └── monsters/Draw Steel Monsters.md
```

## Annotation scheme

Source markdown is annotated with HTML comments before headings:

```markdown
<!-- @type: class | @id: fury -->
## Fury

<!-- @type: feature-group | @level: 1 -->
### 1st-Level Features

<!-- @type: feature -->
#### Growing Ferocity

<!-- @type: ability | @cost: 3 Ferocity -->
> ######## Tide of Death
```

Annotations classify what each section IS (ability, feature, kit, ancestry, etc.). Content parsers then extract structured data (keywords, power rolls, effects) from the body text automatically. See [ANNOTATION-GUIDE.md](ANNOTATION-GUIDE.md) for the full reference.

### SCC (Steel Compendium Classification)

Every classified item gets a permanent SCC identifier:

```
{source}/{type}/{item}

mcdm.heroes.v1/class/fury
mcdm.heroes.v1/feature.ability.fury.level-1/brutal-slam
mcdm.heroes.v1/feature.trait.fury.level-1/growing-ferocity
mcdm.heroes.v1/feature.ability.common/grab
mcdm.heroes.v1/kit/panther
mcdm.heroes.v1/ancestry/dwarf
mcdm.heroes.v1/condition/dazed

mcdm.monsters.v1/monster.goblins/goblins                       (group lore page)
mcdm.monsters.v1/monster.goblins.statblock/goblin-warrior      (statblock)
mcdm.monsters.v1/monster.goblins/goblin-malice                 (malice featureblock)
mcdm.monsters.v1/dynamic-terrain.environmental-hazards/lava
mcdm.monsters.v1/retainer.statblock/goblin-guide
```

Type names are singular. Features use `feature.ability` / `feature.trait` with class, level, and kit context in the type path (e.g., `feature.trait.fury.level-1.boren/kit-bonuses`). Monster content uses a nested `monster.<category>[.<subcategory>].statblock/<item>` hierarchy (malice featureblocks are siblings of the `statblock/` folder); see the Monsters section of [ANNOTATION-GUIDE.md](ANNOTATION-GUIDE.md).

SCCs become permanent URLs (`steelcompendium.io/mcdm.heroes.v1/feature.ability.fury.level-1/brutal-slam`) and are immutable once frozen. See `plans/architecture-redesign/scc-taxonomy.md` for the full taxonomy.

## Annotation coverage

The annotated source carries **1,523 annotations** across the full Draw Steel Heroes book:

| Type | Count |
|------|-------|
| ability | 507 |
| feature | 599 |
| feature-group | 104 |
| complication | 100 |
| title | 61 |
| perk | 47 |
| kit | 25 |
| chapter | 21 |
| treasure | 19 |
| career | 18 |
| ancestry | 12 |
| class | 9 |
| movement | 1 |

Auto-detected metadata per ability: signature/triggered subtype, heroic resource cost, ID overrides for special characters.

The **Monsters** book adds 437 statblocks, 64 malice featureblocks, 35 dynamic-terrain objects, and 52 monster groups (validated against the legacy `data-bestiary-*` repos: exact match on names and file counts). Statblocks are H7 and malice/terrain are H9 — heading levels above goldmark's H6 cap, captured by `collectDeepHeadings`.

## The annotated input

`input/heroes/Draw Steel Heroes.md`, `input/beastheart/Draw Steel Beastheart.md`, and `input/monsters/Draw Steel Monsters.md` are **hand-maintained and canonical**. The heroes file holds ~4,055 SCC cross-reference links and all annotations directly. (Heroes was originally bootstrapped by a since-removed `annotate_heroes.py` script plus one-off link-adder scripts; the Monsters annotation pass was bootstrapped by a since-removed `annotate_monsters.pl`. Those one-time bootstrap scripts have been deleted — the sources are now hand-maintained in place.) See `docs/linking-guide.md` for the rules on adding cross-reference links.

## Pipeline configuration

`pipeline.yaml` defines the input source, output formats, and classification settings:

```yaml
book: mcdm.heroes.v1
input: ./input/heroes/Draw Steel Heroes.md

classification:
  registry: ./classification.json
  freeze: false          # currently unfrozen while multi-book content is added

output:
  base_dir: ../data/data-rules
  formats: [md, json, yaml]
  variants:
    linked: true
    dse: true
    dse_linked: true

books:                   # additional books beyond the primary; each overrides any top-level setting
  - book: mcdm.monsters.v1
    input: ./input/monsters/Draw Steel Monsters.md
    output: { base_dir: ../data/data-bestiary }
  - book: mcdm.beastheart.v1
    input: ./input/beastheart/Draw Steel Beastheart.md
    output: { base_dir: ../data/data-beastheart }
```

Multi-book support is built in: the beastheart and monsters books are configured as entries under `books:`. A bare `gen` processes only the primary (heroes) book — pass `--all` or `--book <id>` to regenerate the others (the `just deploy*` recipes pass `--all`).

## Design documents

The full architecture redesign lives in `plans/architecture-redesign/`:

| Document | Content |
|----------|---------|
| `README.md` | Overview and reading guide |
| `scc-taxonomy.md` | SCC type taxonomy and classification rules |
| `annotation-spec.md` | Annotation format specification |
| `content-parsers.md` | What each parser extracts from content |
| `context-stack.md` | Hierarchical annotation context stack |
| `go-project-structure.md` | Go module layout |
| `pipeline-config.md` | Pipeline configuration spec |
| `phases.md` | Phased implementation plan |
| `decisions.md` | Architectural decision log |

## CLI commands

```bash
# Run from the steel-etl directory. Requires devbox (see workspace CLAUDE.md).

steel-etl gen --config pipeline.yaml              # Run full pipeline
steel-etl gen --format json                        # Generate only JSON output
steel-etl gen --locale es                          # Generate for a specific locale

steel-etl validate --config pipeline.yaml          # Check annotations and coverage
steel-etl validate --scc-stable                    # Also verify SCC codes haven't changed

steel-etl classify --config pipeline.yaml          # Show all SCC codes by type
steel-etl classify --diff                          # Show changes vs. registry
steel-etl classify --export-map scc.json           # Export SCC-to-type mapping

steel-etl strip input.md -o clean.md               # Remove annotations
steel-etl strip --for-translation -o template.md   # Create translation template

steel-etl site --config v2/site.yaml               # Build MkDocs site from output
```

## Roadmap

- **Phase 0** -- Design & Annotate ✓
- **Phase 1** -- Core Go CLI ✓
- **Phase 2** -- Full output pipeline (JSON, YAML, linked/DSE, aggregation) ✓
- **Phase 3** -- Translation + SCC URLs ✓ (3.6 i18n deferred, awaiting translated content)
- **SCC Freeze** -- validate/classify commands ✓ (registry currently reopened for multi-book expansion)
- **Phase 4** -- Data repo consolidation ✓ + homebrew content registry (4.4-4.5 open)
- **Phase 5** -- Monsters book integration (multi-book proof) ✓ *(2026-06-05; 2,627 codes across 3 books)*
