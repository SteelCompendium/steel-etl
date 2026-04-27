# steel-etl

Go CLI tool that processes annotated Draw Steel TTRPG markdown into structured, multi-format output for the [Steel Compendium](https://steelcompendium.io).

**Status:** Phases 0-3 complete (3.6 i18n deferred). SCC taxonomy frozen (1,432 codes). All 9 output generators, 14 content parsers, `validate`/`classify`/`gen`/`strip`/`site` CLI commands implemented.

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

The source is a single annotated markdown file (e.g., `Draw Steel Heroes.md`) with HTML comment annotations that tell the parser what each section is. The tool parses it in a single pass and produces per-section output files in multiple formats with full YAML frontmatter.

## Project layout

```
steel-etl/
├── cmd/steel-etl/main.go         # CLI entrypoint
├── internal/
│   ├── cli/                       # Cobra commands (gen, validate, classify, strip, site)
│   ├── parser/                    # Markdown parser: annotations, document, sections
│   ├── context/                   # Hierarchical annotation context stack
│   ├── content/                   # 14 content parsers (ability, class, kit, ancestry, etc.)
│   ├── scc/                       # SCC classifier and registry (frozen)
│   ├── output/                    # 9 output generators (md, json, yaml, linked, dse, etc.)
│   ├── pipeline/                  # Orchestrates parse → classify → generate
│   └── site/                      # MkDocs site builder from steel-etl output
├── testdata/fixtures/             # Test fixtures (simple_class.md)
├── annotate_heroes.py             # Annotation script (Phase 0 tooling)
├── pipeline.yaml                  # Pipeline configuration
├── Makefile                       # Build, test, lint targets
└── input/heroes/                  # Annotated source (generated)
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
```

Type names are singular. Features use `feature.ability` / `feature.trait` with class, level, and kit context in the type path (e.g., `feature.trait.fury.level-1.boren/kit-bonuses`).

SCCs become permanent URLs (`steelcompendium.io/mcdm.heroes.v1/feature.ability.fury.level-1/brutal-slam`) and are immutable once frozen. See `plans/architecture-redesign/scc-taxonomy.md` for the full taxonomy.

## Annotation coverage

The `annotate_heroes.py` script produces **1,523 annotations** across the full Draw Steel Heroes book:

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

## Generating the annotated input

```bash
# Requires the source file at ../data-gen/input/heroes/Draw Steel Heroes.md
python3 annotate_heroes.py            # generates input/heroes/Draw Steel Heroes.md
python3 annotate_heroes.py --dry-run  # preview without writing
```

## Pipeline configuration

`pipeline.yaml` defines the input source, output formats, and classification settings:

```yaml
book: mcdm.heroes.v1
input: ./input/heroes/Draw Steel Heroes.md

classification:
  registry: ./classification.json
  freeze: true           # frozen 2026-04-26 — existing codes cannot be removed

output:
  base_dir: ../data/data-rules
  formats: [md, json, yaml]
  variants:
    linked: true
    dse: true
    dse_linked: true
```

Multi-book support is built in. The monsters book is configured as an additional entry under `books:`.

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
- **SCC Freeze** -- 1,432 codes frozen, validate/classify commands ✓
- **Phase 4** -- Data repo consolidation + homebrew content registry
- **Phase 5** -- Monsters book integration (multi-book proof)
