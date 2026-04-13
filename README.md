# steel-etl

Go CLI tool that processes annotated Draw Steel TTRPG markdown into structured, multi-format output for the [Steel Compendium](https://steelcompendium.io).

**Status:** Phase 0 (Design & Annotate) complete. Phase 1 (Go CLI build) is next.

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
├── annotate_heroes.py     # Annotation script (Phase 0 tooling)
├── annotate_fury_pilot.py # Original Fury-only pilot (kept as reference)
├── ANNOTATION-GUIDE.md    # Quick reference for annotation syntax
├── pipeline.yaml          # Pipeline configuration
├── input/
│   └── heroes/
│       └── Draw Steel Heroes.md   # Annotated source (generated)
└── cmd/                   # (Phase 1: Go CLI, not yet built)
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

mcdm.heroes.v1/classes/fury
mcdm.heroes.v1/abilities.fury/brutal-slam
mcdm.heroes.v1/kits/panther
mcdm.heroes.v1/ancestries/dwarf
mcdm.heroes.v1/conditions/dazed
```

SCCs become permanent URLs (`steelcompendium.io/mcdm.heroes.v1/abilities.fury/brutal-slam`) and are immutable once frozen. See `plans/architecture-redesign/scc-taxonomy.md` for the full taxonomy.

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
  freeze: false          # set to true after freezing SCC codes

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

## Roadmap

- **Phase 0** -- Design & Annotate (done)
- **Phase 1** -- Core Go CLI: parse annotated markdown, produce per-section markdown output
- **Phase 2** -- Full output pipeline: JSON, YAML, linked/DSE variants, aggregation
- **Phase 3** -- Translation support + SCC-based website URLs
- **Phase 4** -- Data repo consolidation + homebrew content registry
- **Phase 5** -- Monsters book integration (multi-book proof)
