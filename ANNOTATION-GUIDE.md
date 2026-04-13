# Annotation Guide

Quick reference for annotating Draw Steel source markdown. For the full spec, see `plans/architecture-redesign/annotation-spec.md`.

## Document Frontmatter

Every annotated input file starts with YAML frontmatter:

```yaml
---
book: mcdm.heroes.v1
source: MCDM
title: Draw Steel Heroes
---
```

## Annotation Syntax

HTML comments placed immediately before a heading. The annotation applies to that heading's section.

**Single-line** (few fields):
```markdown
<!-- @type: class | @id: fury -->
## Fury
```

**Multi-line** (many fields):
```markdown
<!--
@type: ability
@id: gouge
@cost: 3 Ferocity
-->
#### Gouge
```

## Required Fields

| Field | Description |
|-------|-------------|
| `@type` | Content type (see table below). Determines which parser processes the section. |

## Optional Fields

| Field | Description | When to use |
|-------|-------------|-------------|
| `@id` | Slug identifier. If omitted, derived from heading text. | When heading text would produce a bad slug |
| `@cost` | Heroic resource cost | When cost is NOT in the heading text |
| `@action` | Action type override | For triggered actions (non-standard layout) |
| `@distance` | Distance override | When parser can't extract from content |
| `@target` | Target override | When parser can't extract from content |
| `@keywords` | Keyword list override | When parser can't extract from content |
| `@level` | Feature level | On `feature-group` containers |
| `@subtype` | Classification hint | `signature`, `heroic`, `triggered` |
| `@trigger` | Trigger condition override | Only when parser can't extract trigger from body text |
| `@note` | Free-text note | Metadata-only, no parser runs |

### SCC Override Fields

| Field | Description | When to use |
|-------|-------------|-------------|
| `@scc` | Replace the auto-derived canonical SCC entirely | When auto-classification is wrong or ambiguous |
| `@scc-alias` | Add an additional lookup alias for this item | Cross-references, redirects, shared abilities |

See `plans/architecture-redesign/scc-taxonomy.md` for the full SCC taxonomy and classification rules.

Any `@`-prefixed key is captured. Unknown keys pass through to frontmatter output.

## Body-Parsed vs Annotation Fields

The parser extracts most structured data from the **body text** of each section — not from annotations. Annotations are for classification and overrides only.

| Data | Parsed from body? | Annotation override? |
|------|-------------------|---------------------|
| Trigger condition | **Yes** — extracted from "**Trigger:**" line | `@trigger` (only if body parse fails) |
| Keywords | **Yes** — extracted from keyword line | `@keywords` (only if body parse fails) |
| Distance/range | **Yes** — extracted from distance line | `@distance` (only if body parse fails) |
| Target | **Yes** — extracted from target line | `@target` (only if body parse fails) |
| Power roll tiers | **Yes** — extracted from table | *(no override)* |
| Action type | **Yes** — extracted from action line | `@action` (only if body parse fails) |
| Cost | Partially — from heading or body | `@cost` (when not in heading text) |

**Rule of thumb:** Don't annotate what the parser can extract from the content. Use annotation fields only as overrides for exceptional cases where the body structure is non-standard.

## Content Types

### Structural

| @type | Use for | Parser extracts |
|-------|---------|-----------------|
| `chapter` | H1 top-level chapters | Title, content passthrough |
| `class` | H2 class sections (Fury, Shadow...) | Overview, heroic resource |
| `feature-group` | H3 "Nth-Level Features" containers | Level context for children |

### Class Content

| @type | Use for | Parser extracts |
|-------|---------|-----------------|
| `ability` | Individual abilities (Gouge, Brutal Slam...) | Keywords, action, distance, target, power roll, effect, flavor |
| `feature` | Non-ability features (Growing Ferocity...) | Name, description |

### Character Creation

| @type | Use for | Parser extracts |
|-------|---------|-----------------|
| `ancestry` | Ancestry sections (Dwarf, Human...) | Traits, ancestry points |
| `kit` | Kit sections (Panther, Shining Armor...) | Stat bonuses, equipment, signature ability |
| `perk` | Individual perks | Prerequisites, description |
| `career` | Career sections | Grants (skills, languages, etc.) |
| `culture` | Culture benefit options | Environment/organization/upbringing |
| `complication` | Complications | Description |
| `skill` | Individual skills | Associated characteristic |

### Rewards

| @type | Use for | Parser extracts |
|-------|---------|-----------------|
| `title` | Title entries | Echelon, benefits |
| `treasure` | Treasure entries | Treasure type, properties |

### Monsters (future)

| @type | Use for |
|-------|---------|
| `monster` | Monster entries (container for lore + statblock) |
| `statblock` | Stat blocks within a monster entry |
| `dynamic-terrain` | Dynamic terrain features |
| `retainer` | Retainer NPCs |

## Annotation Patterns

### Class section (Fury as example)

```markdown
<!-- @type: chapter | @id: classes -->
# Classes

<!-- @type: class | @id: fury -->
## Fury

...overview text...

<!-- @type: feature-group | @level: 1 -->
### 1st-Level Features

<!-- @type: feature -->
#### Primordial Aspect
...

<!-- @type: feature -->
#### Ferocity
...

<!-- @type: feature -->
#### Growing Ferocity
...

<!-- @type: ability | @subtype: signature -->
#### Brutal Slam
...

<!--
@type: ability
@cost: 3 Ferocity
-->
#### Gouge
...

<!-- @type: feature -->
#### Mighty Leaps
...

<!-- @type: feature-group | @level: 2 -->
### 2nd-Level Features

<!-- @type: ability | @cost: 5 Ferocity -->
#### Blood for Blood!
...
```

### Triggered actions

The trigger condition is parsed from the body text (the "**Trigger:**" line). Only annotate `@subtype: triggered` to signal the parser:

```markdown
<!-- @type: ability | @subtype: triggered -->
#### Reactive Strike
```

If the body structure is non-standard and the parser can't extract the trigger, use the `@trigger` override:

```markdown
<!--
@type: ability
@subtype: triggered
@trigger: A creature within your reach makes an attack against one of your allies
-->
#### Reactive Strike
```

### SCC overrides

Use `@scc` to replace the auto-derived classification, and `@scc-alias` to add redirects:

```markdown
<!--
@type: ability
@scc: mcdm.heroes.v1/abilities.fury/reactive-strike
@scc-alias: mcdm.heroes.v1/abilities.common/reactive-strike
-->
#### Reactive Strike
```

### Ancestry section

```markdown
<!-- @type: ancestry | @id: dwarf -->
## Dwarf
...
```

### Kit section

```markdown
<!-- @type: kit | @id: panther -->
## Panther
...
```

### Simple types

```markdown
<!-- @type: perk | @id: alert -->
#### Alert
...

<!-- @type: condition | @id: dazed -->
### Dazed
...

<!-- @type: career | @id: artisan -->
## Artisan
...
```

## End Markers

Most sections are delimited by the next annotation or heading at the same/higher level. For edge cases where the parser can't determine where a section ends, use an `@end` comment:

```markdown
<!-- @type: ability | @subtype: signature -->
#### Brutal Slam
...ability content...
<!-- @end: brutal-slam -->
```

The id after `@end:` must match the section's `@id` (explicit or auto-derived from the heading slug). The parser verifies this — a mismatch is a build error.

**When to use `@end`:**
- Adjacent sections at the same heading level where content bleeds across (rare)
- Sections followed by non-heading content that belongs to a *different* parent
- Any case where the "next heading/annotation closes the previous section" heuristic fails

**When NOT to use `@end`:**
- Normal sequential sections — the next annotation/heading closes the previous one automatically
- Parent-child nesting — heading levels already establish hierarchy

End markers are rare. If you need many of them, the source structure likely needs cleanup instead.

## What NOT to Annotate

- **Headings without meaningful content to extract** -- e.g., "Basics" under a class (just structural grouping, the parser handles it as part of the class section)
- **Tables that are part of a parent section** -- e.g., the advancement table under a class, Growing Ferocity tables
- **Sub-sections of abilities** -- power roll tiers, effect text, etc. are extracted by the ability parser from content structure
- **Index files** -- generated, not classified

## Source Normalization

The original source markdown uses blockquoted H8 headers (`> ########`) to delimit abilities. These are a non-standard artifact of the old ETL and should be **normalized** during annotation:

- Remove the `> ` blockquote wrapper
- Replace `########` with a standard heading level appropriate to the nesting (typically `####`)
- Add an `<!-- @type: ability ... -->` annotation before the heading

The annotation provides the section boundary that the blockquote/H8 combo previously served.

## Heading Level Guidelines

In the source markdown, heading levels vary. The annotation-to-heading association is purely positional (the annotation goes immediately before the heading it describes). The heading level itself doesn't affect what `@type` you use.

## Validating Annotations

```bash
# Check annotation syntax and coverage (once steel-etl is built)
steel-etl validate input/heroes/Draw\ Steel\ Heroes.md
```
