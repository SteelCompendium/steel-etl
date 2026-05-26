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
| Introduction | 7-589 | done | done (28) | done (22) | - | - | - | - | - | - | done (3) |
| The Basics | 590-1055 | done | done (17) | done (36) | - | - | - | - | - | - | done (11) |
| Making a Hero | 1056-1263 | done | done (12) | done (18) | - | - | - | - | - | - | done (15) |
| Ancestries | 1264-3199 | done | done (22) | done (324) | - | - | - | - | - | - | done (11) |
| Background | 3200-3206 | done | - | - | - | - | - | - | - | - | - |
| Cultures | 3207-3493 | done | - | done (47) | - | - | - | - | - | - | done (2) |
| Careers | 3494-4065 | done | done (4) | - | - | - | - | - | - | - | done (5) |
| Classes | 4066-17606 | done | done (316) | done (4) | - | - | - | - | - | - | done (67) |
| Kits | 17607-18580 | done | done (5) | - | - | - | - | - | - | - | done (4) |
| Perks | 18581-18946 | done | - | - | - | - | - | - | - | - | done (2) |
| Complications | 18947-20167 | done | done (12) | done (8) | - | - | - | - | - | - | done (18) |
| Tests | 20168-20408 | done | - | done (1) | - | - | - | - | - | - | done (2) |
| Skills | 20409-20856 | done | done (12) | done (3) | - | - | - | - | - | - | done (3) |
| Combat | 20857-21636 | done | done (11) | done (9) | - | - | - | - | - | - | done (13) |
| Negotiation | 21637-22187 | done | done (4) | done (7) | - | - | - | - | - | - | done (2) |
| Downtime Projects | 22188-23215 | done | done (12) | done (14) | - | - | - | - | - | - | done (5) |
| Rewards | 23216-23220 | done | - | - | - | - | - | - | - | - | - |
| Treasures | 23221-25258 | done | done (5) | done (8) | - | - | - | - | - | - | done (9) |
| Titles | 25259-26339 | done | done (12) | done (21) | - | - | - | - | - | - | done (16) |
| Gods and Religion | 26340-27294 | done | done (48) | done (123) | - | - | - | - | - | - | - |
| For the Director | 27295-28721 | done | done (11) | done (8) | - | - | - | - | - | - | done (14) |
