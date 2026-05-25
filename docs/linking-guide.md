# Linking Guide

Instructions for adding scc: cross-reference links to the input document
(`input/heroes/Draw Steel Heroes.md`). Designed to be picked up by any AI
session and followed step by step.

## Reference

- **Reference table:** `docs/linking-reference.md` — all linkable terms with display names, variants, and SCC codes
- **Link format:** `[Display Text](scc:mcdm.heroes.v1/type/id)`
- **Input document:** `input/heroes/Draw Steel Heroes.md`

## Linking Rules

### Link when

- The term refers to a game mechanic (a class, career, ancestry, kit, etc.)
- Link ALL instances of the term — density filtering is handled by the pipeline at build time
- Bolded terms that reference game mechanics (e.g., glossary: `**Criminal:** a career choice...` should become `**[Criminal](scc:mcdm.heroes.v1/career/criminal):** a career choice...`)
- Terms inside nested child sections of their own parent definition (e.g., "Fury" mentioned in a Fury ability description — when extracted, the ability page needs a link back to its class)

### Don't link when

- The term is used as ordinary English, not referencing the game mechanic ("fighting criminals" ≠ the Criminal career)
- The term appears in its own section heading (`## Fury` does not link to itself)
- The text is inside an annotation comment (`<!-- @type: ... -->`)

### Case and variants

- Match case-insensitively: "fury", "Fury", and "FURY" all match
- Handle plurals: "criminals" should link with display text "criminals" to the Criminal career SCC code
- Handle possessives: "Fury's" should link "Fury's" to the Fury class SCC code (include the possessive in the display text)
- Use the reference table for known plural forms; use judgment for unlisted variants

### Pre-existing links

- **First pass (current):** Strip ALL pre-existing links before adding scc: links. Both old scc: links and PDF-origin links are stale.
- **Future passes:** When re-running after a PDF update, preserve existing scc: links and only add new ones.

### Uncertainty marker

When unsure whether a term is a game reference or flavor text:

```
<!-- REVIEW: is this a game reference? -->[Criminal](scc:mcdm.heroes.v1/career/criminal)<!-- /REVIEW -->
```

Find flagged cases: `grep -n "<!-- REVIEW:" input/heroes/Draw\ Steel\ Heroes.md`

## Workflow

### For each chapter

1. Find the chapter in the progress matrix below
2. Read the chapter text (between its `<!-- @type: chapter -->` marker and the next chapter marker)
3. If the "Strip Links" column is not done for this chapter, strip all pre-existing links first
4. Read the full reference table (`docs/linking-reference.md`)
5. Add scc: links for ALL game mechanic references across all types in a single pass
6. Use `<!-- REVIEW: -->` markers for uncertain cases
7. Update the progress matrix
8. Commit: `git commit -m "link: add cross-reference links to {chapter} chapter"`

### Validation

After completing all chapters, run the pipeline and check for warnings:

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml' 2>&1 | grep WARN
```

Warnings indicate unresolved SCC codes (typos or missing registry entries).

## Progress Matrix

| Chapter | Lines | Strip Links | Classes | Ancestries | Careers | Kits | Perks | Complications | Titles | Treasures | Chapters |
|---------|-------|------------|---------|-----------|---------|------|-------|--------------|--------|-----------|----------|
| Introduction | 7-589 | - | - | - | - | - | - | - | - | - | - |
| The Basics | 590-1055 | - | - | - | - | - | - | - | - | - | - |
| Making a Hero | 1056-1263 | - | - | - | - | - | - | - | - | - | - |
| Ancestries | 1264-3199 | - | - | - | - | - | - | - | - | - | - |
| Background | 3200-3206 | - | - | - | - | - | - | - | - | - | - |
| Cultures | 3207-3493 | - | - | - | - | - | - | - | - | - | - |
| Careers | 3494-4065 | - | - | - | - | - | - | - | - | - | - |
| Classes | 4066-17606 | - | - | - | - | - | - | - | - | - | - |
| Kits | 17607-18580 | - | - | - | - | - | - | - | - | - | - |
| Perks | 18581-18946 | - | - | - | - | - | - | - | - | - | - |
| Complications | 18947-20167 | - | - | - | - | - | - | - | - | - | - |
| Tests | 20168-20408 | - | - | - | - | - | - | - | - | - | - |
| Skills | 20409-20856 | - | - | - | - | - | - | - | - | - | - |
| Combat | 20857-21636 | - | - | - | - | - | - | - | - | - | - |
| Negotiation | 21637-22187 | - | - | - | - | - | - | - | - | - | - |
| Downtime Projects | 22188-23215 | - | - | - | - | - | - | - | - | - | - |
| Rewards | 23216-23220 | - | - | - | - | - | - | - | - | - | - |
| Treasures | 23221-25258 | - | - | - | - | - | - | - | - | - | - |
| Titles | 25259-26339 | - | - | - | - | - | - | - | - | - | - |
| Gods and Religion | 26340-27294 | - | - | - | - | - | - | - | - | - | - |
| For the Director | 27295-28721 | - | - | - | - | - | - | - | - | - | - |
