# Companion Statblock Grid-Card Format — Design

**Date:** 2026-06-02
**Status:** Approved (design)
**Scope:** `steel-etl/input/beastheart/Draw Steel Beastheart.md` — the 14 companion stat blocks under `### Companion Stat Blocks`.

## Goal

Reformat the 14 Beastheart companion stat blocks into the legacy "grid-card" visual
(`**value**<br>Label` multi-column table) so they read as proper stat-block cards and stay
visually consistent with the future **Monster** book cards.

## Why this is safe (visual-only)

- The companion stat fields (Size/Speed/Stability/Free Strike, Immunity/Movement/Skills,
  characteristics) are **markdown passthrough** — `internal/content/feature.go` does not parse
  them into structured fields. Reformatting the table therefore has **no effect on SCC codes,
  JSON/SDK output, or classification**.
- `v2/docs/stylesheets/tables.css` ("Monster Statblock Tables" section) already styles
  multi-column `:not([class])` tables, including `<br>` line breaks inside cells and rounded
  card borders. **No CSS change required.**
- Abilities, headings, and all `<!-- @type: ... -->` annotations are **untouched** — only the
  stat-block header (keyword line + stat table + Immunity line + characteristics line) is replaced.

## Card shape (monster-parallel 5-stat)

A single 5-column `:not([class])` table. The 5-column width is forced by the characteristics row
(Might/Agility/Reason/Intuition/Presence). The defensive row mirrors the monster layout
(Size/Speed/**Stamina**/Stability/Free Strike); companions have no fixed Stamina (it equals the
beastheart's maximum), so the Stamina cell reads `= yours`. Short rows (header, Immunity) leave
trailing cells blank.

### Transformation template

**Before:**
```markdown
<!-- @type: feature-group | @companion: basilisk | @level: 1 -->
#### Basilisk

*Beast, Companion*

| Size | Speed | Stability | Free Strike |
|------|-------|-----------|-------------|
| 1L   | 5     | 2         | 1 + M       |

**Immunity:** Poison 3 **Movement:** — **Skills:** [Alertness](scc:mcdm.heroes.v1/skill/alertness)

**Might** +2 **Agility** +1 **Reason** −1 **Intuition** +2 **Presence** +2
```

**After:**
```markdown
<!-- @type: feature-group | @companion: basilisk | @level: 1 -->
#### Basilisk

| Beast, Companion |  | Level 1 |  |  |
|:--:|:--:|:--:|:--:|:--:|
| **1L**<br>Size | **5**<br>Speed | **= yours**<br>Stamina | **2**<br>Stability | **1 + M**<br>Free Strike |
| **Poison 3**<br>Immunity | **—**<br>Movement | **[Alertness](scc:mcdm.heroes.v1/skill/alertness)**<br>Skills |  |  |
| **+2**<br>Might | **+1**<br>Agility | **−1**<br>Reason | **+2**<br>Intuition | **+2**<br>Presence |
```

### Field-mapping rules

- **Header row, cell 1:** the keyword/type line verbatim (varies: `Beast, Companion`,
  `Animal, Companion`, `Companion, Dragon`, `Companion, Elemental`, `Companion, Ooze`,
  `Companion, Infernal`). Cell 3: `Level N` from `@level`. Cells 2/4/5 blank.
- **Defensive row:** `Size | Speed | Stamina(= yours) | Stability | Free Strike`, each as
  `**value**<br>Label`. Values pulled from the existing 1-row stat table.
- **Meta row:** `Immunity | Movement | Skills`, each `**value**<br>Label`; cells 4/5 blank.
  Skills value keeps its `[...](scc:...)` link inside the bold. Long Immunity values
  (e.g. Drake's "Attuned damage type 3 (see …)") are kept verbatim.
- **Characteristics row:** `Might | Agility | Reason | Intuition | Presence`, each
  `**value**<br>Label`. Uses the en-dash `−` for negatives, matching source.

## Out of scope

- No `monster`/`statblock` parser work (still "future" in ANNOTATION-GUIDE.md).
- No changes to ability cards, advancement features, or any non-companion content.
- No CSS changes.

## Verification

1. `devbox run -- go build ./...` (sanity).
2. `devbox run -- steel-etl validate --scc-stable` — confirm **no SCC codes changed**
   (proves the reformat is structurally inert).
3. Build the v2 site (`steel-etl site` / `just deploy-v2` dry path) and eyeball a couple of
   companion pages (Basilisk + Drake for the long-immunity case) to confirm the cards render.
