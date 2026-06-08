# Card ⇄ Data Field Parity — data-sdk-npm Consumer Side Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the `data-sdk-npm` TypeScript SDK into parity with the schema fields added on the steel-etl producer side — `flavor` (culture, perk) and `echelon` (treasure) — and reconcile the `project_goal` type so the SDK round-trips real pipeline output without loss.

**Architecture:** This is the **consumer-side companion** to `2026-06-08-card-data-field-parity.md`. That plan landed the parser fields + both schema copies (the schema copy in *this* repo, `src/schema/*.schema.json`, is updated there). Here we update the TypeScript that mirrors those schemas: the DTO classes (`src/dto/*DTO.ts`), the model classes (`src/model/*.ts`), and tests. Treasure/culture/perk are **not** handled by the per-type markdown readers/writers (those exist only for feature/statblock/featureblock) — they round-trip through the generic DTO-driven JSON/YAML path, so DTO + model changes are sufficient for I/O.

**Tech Stack:** TypeScript, Jest (`jest.config.js`), the repo's `justfile` build/test recipes, Node (devbox).

**Prerequisite:** `2026-06-08-card-data-field-parity.md` (steel-etl) must be merged first — its Task 4 adds `echelon`/`flavor` to `data-sdk-npm/src/schema/*.schema.json`, which this plan's DTOs must match.

---

## Current state (verified 2026-06-08)

| DTO | has `flavor`? | has `echelon`? | notes |
|-----|---------------|----------------|-------|
| `TreasureDTO` | ✅ | ❌ **add** | also has `project_goal?: number` and `project_roll_characteristic?: string` already; **`project_goal` type mismatch** — steel-etl emits `"45"` (string), DTO expects `number` |
| `CultureDTO` | ❌ **add** | n/a | — |
| `PerkDTO` | ❌ **add** | n/a | — |

Every DTO follows the same shape: a public field, a copy line in `static partialFromModel`, and a matching field on the model class (`src/model/Treasure.ts`, `Culture.ts`, `Perk.ts`) with `toDTO()` / `fromDTO()` plumbing.

---

## Task 0: Investigate the model + validation pattern (read-only)

**Files:** none (reading only)

- [ ] **Step 1: Read the three model classes and one as the template**

Run: `cat src/model/Treasure.ts src/model/Culture.ts src/model/Perk.ts`

Confirm for each: (a) the public field list, (b) how `fromDTO(dto)` copies fields onto the model, (c) how `toDTO()` (or `partialFromModel`) serializes them. The new fields follow the exact same three touch-points.

- [ ] **Step 2: Confirm there is no per-type markdown reader/writer for these types**

Run: `ls src/io/markdown && grep -rl "Treasure\|Culture\|Perk" src/io`
Expected: only feature/statblock/featureblock markdown readers exist; treasure/culture/perk appear (if at all) only in the generic JSON/YAML path. This confirms DTO+model changes are sufficient.

- [ ] **Step 3: Decide the `project_goal` reconciliation**

steel-etl emits `project_goal` as a **string** (`extractField` returns the raw `"45"`). The DTO currently types it `number`. Check what `src/schema/treasure.schema.json` declares for `project_goal` (`type: "string"` vs `"number"` vs `["string","number"]`). Pick ONE source of truth:
- **Recommended:** make the SDK tolerant — type `project_goal?: string | number` in the DTO/model, since the schema (post-merge) and real pipeline data use a string, but existing SDK consumers may expect a number. Record the choice in the commit message.

No code in this task — just the reading + decision that the next tasks depend on.

---

## Task 1: TreasureDTO + Treasure model — add `echelon`, reconcile `project_goal`

**Files:**
- Modify: `src/dto/TreasureDTO.ts`, `src/model/Treasure.ts`
- Test: `src/__tests__/` (mirror the nearest existing DTO round-trip test)

- [ ] **Step 1: Write the failing test**

Find the existing treasure DTO/model test (`grep -rl Treasure src/__tests__`) and add a case that constructs a `TreasureDTO` from a plain object including `echelon` and a string `project_goal`, round-trips it through `toModel().toDTO()` (or the JSON reader/writer), and asserts both survive:

```ts
it('round-trips echelon and string project_goal', () => {
    const dto = new TreasureDTO({
        name: 'Bag of Holding',
        echelon: '3',
        project_goal: '45',
        project_roll_characteristic: 'Reason',
        flavor: 'A bag that holds far more than its size suggests.',
    });
    const back = TreasureDTO.fromModel(dto.toModel());
    expect(back.echelon).toBe('3');
    expect(back.project_goal).toBe('45');
    expect(back.flavor).toBe('A bag that holds far more than its size suggests.');
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npx jest -t 'round-trips echelon'` (or the repo's `just test` filtered).
Expected: FAIL — `echelon` is dropped (not on DTO/model) and/or `project_goal` type error.

- [ ] **Step 3: Implement**

In `src/dto/TreasureDTO.ts`:
- Add field `echelon?: string;` (place after `level?: string;`).
- Change `project_goal?: number;` to `project_goal?: string | number;` (per Task 0 Step 3 decision).
- In `static partialFromModel`, add `if (model.echelon !== undefined) data.echelon = model.echelon;` and keep the existing `project_goal` copy line.

In `src/model/Treasure.ts`: add the matching `echelon?: string;` field, adjust `project_goal` type to `string | number`, and add the field to `fromDTO`/`toDTO` (mirror the existing `level` field's three touch-points exactly).

- [ ] **Step 4: Run the test to verify it passes**

Run: `npx jest -t 'round-trips echelon'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/data-sdk-npm
git add src/dto/TreasureDTO.ts src/model/Treasure.ts src/__tests__
git commit -m "feat(treasure): add echelon field and tolerate string project_goal"
```

---

## Task 2: CultureDTO + Culture model — add `flavor`

**Files:**
- Modify: `src/dto/CultureDTO.ts`, `src/model/Culture.ts`
- Test: `src/__tests__/`

- [ ] **Step 1: Write the failing test**

```ts
it('round-trips culture flavor', () => {
    const dto = new CultureDTO({ name: 'Nomadic', flavor: 'A wandering people of the high steppes.' });
    const back = CultureDTO.fromModel(dto.toModel());
    expect(back.flavor).toBe('A wandering people of the high steppes.');
});
```

- [ ] **Step 2: Run to verify it fails**

Run: `npx jest -t 'round-trips culture flavor'`
Expected: FAIL — `flavor` dropped.

- [ ] **Step 3: Implement**

In `src/dto/CultureDTO.ts`: add `flavor?: string;` (after `name!: string;`) and `if (model.flavor !== undefined) data.flavor = model.flavor;` in `partialFromModel`.
In `src/model/Culture.ts`: add the `flavor?: string;` field + `fromDTO`/`toDTO` plumbing (mirror `language`).

- [ ] **Step 4: Run to verify it passes**

Run: `npx jest -t 'round-trips culture flavor'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/data-sdk-npm
git add src/dto/CultureDTO.ts src/model/Culture.ts src/__tests__
git commit -m "feat(culture): add flavor field"
```

---

## Task 3: PerkDTO + Perk model — add `flavor`

**Files:**
- Modify: `src/dto/PerkDTO.ts`, `src/model/Perk.ts`
- Test: `src/__tests__/`

- [ ] **Step 1: Write the failing test**

```ts
it('round-trips perk flavor', () => {
    const dto = new PerkDTO({ name: 'Alert', flavor: 'You always keep one eye on the door.' });
    const back = PerkDTO.fromModel(dto.toModel());
    expect(back.flavor).toBe('You always keep one eye on the door.');
});
```

- [ ] **Step 2: Run to verify it fails**

Run: `npx jest -t 'round-trips perk flavor'`
Expected: FAIL — `flavor` dropped.

- [ ] **Step 3: Implement**

In `src/dto/PerkDTO.ts`: add `flavor?: string;` (after `name!: string;`) and the copy line in `partialFromModel`.
In `src/model/Perk.ts`: add the `flavor?: string;` field + `fromDTO`/`toDTO` plumbing (mirror `prerequisites`).

- [ ] **Step 4: Run to verify it passes**

Run: `npx jest -t 'round-trips perk flavor'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/data-sdk-npm
git add src/dto/PerkDTO.ts src/model/Perk.ts src/__tests__
git commit -m "feat(perk): add flavor field"
```

---

## Task 4: Full build, schema-validation, and DTO/schema drift check

**Files:** none (verification only)

- [ ] **Step 1: Build the SDK**

Run: `npx tsc --noEmit` (or the repo's `just build`).
Expected: no type errors.

- [ ] **Step 2: Run the full Jest suite**

Run: `npx jest` (or `just test`).
Expected: PASS, including any schema-validation tests under `src/validation`.

- [ ] **Step 3: Confirm the schema copy here matches the steel-etl copy**

Run:

```bash
cd /home/vexa/code/steel_compendium/workspace
for s in culture perk treasure; do
  diff -q data-sdk-npm/src/schema/$s.schema.json steel-etl/schemas/$s.schema.json \
    && echo "$s: IN SYNC" || echo "$s: DRIFT"
done
```

Expected: all three `IN SYNC` (the steel-etl plan's Task 4 already updated this repo's schema copy; this is the cross-check).

- [ ] **Step 4: Bump version + changelog if the repo publishes**

If `package.json` is versioned for publish, bump the minor version and add a `CHANGELOG.md` entry: "Add culture/perk `flavor` and treasure `echelon`; treasure `project_goal` accepts string or number." Commit:

```bash
cd /home/vexa/code/steel_compendium/workspace/data-sdk-npm
git add package.json CHANGELOG.md
git commit -m "chore: bump SDK for flavor/echelon field parity"
```

---

## Self-Review notes

- **DTO coverage:** treasure `echelon` (Task 1), culture `flavor` (Task 2), perk `flavor` (Task 3). Treasure `flavor`/`project_*` already existed — only `echelon` + the `project_goal` type were gaps. ✓
- **The model-class plumbing is the one place to read first** (Task 0): the exact `fromDTO`/`toDTO` shape varies, so confirm it against the existing `level`/`language`/`prerequisites` fields before editing.
- **`project_goal` type** is the only semantic decision; defaulting to `string | number` keeps both the real pipeline string output and any existing numeric consumers valid.
- **No markdown reader/writer changes** — treasure/culture/perk use the generic JSON/YAML DTO path (confirmed in Task 0 Step 2).
```
