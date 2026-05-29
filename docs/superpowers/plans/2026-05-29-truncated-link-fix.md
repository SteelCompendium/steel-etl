# Truncated / Mis-Targeted SCC Link Fix — Implementation Plan

> **Status:** Draft, awaiting execution (2026-05-29). Created after the cross-linking and content-linking efforts completed. Depends on no other in-flight work except the uncommitted `internal/site/build.go` change (unrelated; owner committing separately).

**Goal:** Repair SCC links that were truncated or pointed at the wrong code because the iterative linking process linked broad terms (classes/ancestries/careers) *before* narrow ones (features/abilities/complications/skills). Two coupled root causes:

- **Bucket A — stale truncated references:** the longer code already exists, but the reference links only the prefix to the wrong (shorter) code. 95 known sites → 12 target codes, several a *different type* (e.g. `class/shadow` → `complication/shadow-born`, `career/criminal` → `skill/criminal-underworld`).
- **Bucket B — missing annotations:** a header that should be a feature/trait (e.g. `#### Fury Abilities`) is missing its `<!-- @type: feature -->` annotation, so no code was generated, so the feature-linker couldn't link it and the stale prefix link survived. `validate` reports 352 unannotated `####` (H4) sections; the genuine missing-annotation features must be adjudicated out of that pool (most H4s are legitimately structural).

Fixing Bucket B **creates new codes**, which converts the corresponding sites into Bucket-A re-links. So Phase 1 (annotations) precedes Phase 2 (re-link).

**Tech Stack:** Markdown content editing (`input/heroes/Draw Steel Heroes.md`), `devbox run -- bash -c 'cd steel-etl && go ...'` for pipeline validation.

## Decisions locked in (2026-05-29)

1. **Source of truth = `input/heroes/Draw Steel Heroes.md`, edited directly.** It already holds 4,060 hand-added links. Do **not** re-run `annotate_heroes.py` (it would clobber the links). The `.py` is treated as an out-of-sync historical generator; note divergence but do not maintain it in this effort.
2. **Exhaustive annotation sweep:** adjudicate every one of the 352 unannotated H4 headers (and any H5 feature headers spotted alongside), document-wide.
3. **Full per-site AI adjudication** for re-linking — no blind regex. Every site is read in context before changing.

## How the system works (verified)

- `<!-- @type: feature -->` immediately above a `####` header → code `feature.trait.{class}.level-{N}/{slug}`. Class & level come from document context (ancestor `@type: class` + enclosing `@type: feature-group | @level: N`); slug from the heading (or explicit `@id:`). See `internal/content/feature.go`.
- Annotation may be single-line (`<!-- @type: feature -->`) or multi-line (`<!--\n@type: ability\n@id: gouge\n@cost: 3 Ferocity\n-->`). The header follows on the next line (no blank line between). Detection relies on: the line before a header ends with `-->`.
- `classification.json` is **regenerated from annotations every `gen` run**. Never hand-edit it. `frozen: false` today; `ValidateAgainstFrozen` only fails on *removed* codes, so **adding codes is safe**.
- Pipeline prints `WARN: unresolved scc link "..."` for any `scc:` link whose code is not in the registry. Zero WARN is the bar.

## Required reading (every worker, before any task)

1. `ANNOTATION-GUIDE.md` — annotation syntax, required fields, conventions.
2. `plans/architecture-redesign/annotation-spec.md` — full annotation spec.
3. `docs/linking-guide.md` — linking rules + progress matrix.
4. `docs/linking-reference.md` — the 416 linkable terms with codes.
5. `reference/draw-steel-agent-reference.md` (workspace) — to adjudicate whether a header is a real game feature/trait vs. structural prose, and which class/level it belongs to.

## Detection commands (reusable)

**Unannotated H4 headers (Bucket B candidate pool):**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
F="input/heroes/Draw Steel Heroes.md"
awk '/^#### / { if (prev !~ /-->[[:space:]]*$/) print NR": "$0 } { if ($0 !~ /^[[:space:]]*$/) prev=$0 }' "$F"
```

**Bucket A truncation sites (combined phrase resolves to a real code):**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
jq -r '.codes[]' classification.json > /tmp/codes.txt
F="input/heroes/Draw Steel Heroes.md"
grep -oE "\[[A-Za-z'’]+\]\(scc:mcdm\.heroes\.v1/[a-z.0-9-]+/[a-z0-9-]+\)( [A-Za-z'’]+){1,3}" "$F" > /tmp/cand.txt
# (adjudicate each /tmp/cand.txt line: slugify display+following words, check `grep -iE "/<slug>$" /tmp/codes.txt`)
```

**Validate + WARN gate:**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate'
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml' 2>&1 | grep WARN
```

## Unannotated-H4 distribution (Phase 1 sizing)

| Chapter | Unannotated H4 | Notes |
|---|---|---|
| The Basics | 29 | Mostly structural (Might, Agility, Power Rolls) — expect few real features |
| Making a Hero | 12 | |
| Ancestries | 29 | Ancestry traits — likely several real features |
| Cultures | 5 | |
| Careers | 7 | |
| **Classes** | **88** | Richest vein of real missing feature/trait annotations (e.g. Fury Abilities) |
| Kits | 9 | |
| Tests | 8 | Likely structural |
| Skills | 3 | |
| Combat | 29 | Mostly structural rules; watch for the basic-action features |
| Negotiation | 23 | Mostly structural |
| Downtime Projects | 30 | Mostly structural; watch for project features |
| Treasures | 10 | |
| Gods and Religion | 8 | |
| **Total** | **352** | |

---

## Phase 1 — Missing-annotation audit & repair (Bucket B)

For each task: run the unannotated-H4 detector for the chapter's line range, read each header in context, and decide:
- **Real feature/trait** (a discrete game entity with its own rules block, siblings are annotated features) → add the correct annotation (`<!-- @type: feature -->` for traits; `@type: ability` etc. where appropriate). Use explicit `@id:` only if the slug from the heading would be wrong/ambiguous.
- **Structural / prose header** (e.g. "Might", "Power Rolls", "1st Echelon") → leave unannotated.
Use `reference/draw-steel-agent-reference.md` to confirm class/level placement. After each chapter: regenerate, confirm the expected new codes appear, and that `validate` coverage rose with **0 unknown @type**.

- [ ] **Task 1.1 — Classes chapter, intro + Censor + Conduit** (lines 4066–7793; ~subset of the 88)
- [ ] **Task 1.2 — Classes chapter, Elementalist + Fury + Null** (7794–12210). Includes the canonical `#### Fury Abilities` → `<!-- @type: feature -->` → `feature.trait.fury.level-1/fury-abilities`.
- [ ] **Task 1.3 — Classes chapter, Shadow + Tactician + Talent + Troubadour** (12211–17606)
- [ ] **Task 1.4 — Ancestries chapter** (1264–3199; 29 candidates)
- [ ] **Task 1.5 — The Basics + Making a Hero** (590–1263; 41 candidates, mostly structural — expect to leave most alone)
- [ ] **Task 1.6 — Cultures + Careers + Kits** (3207–4065, 17607–18580; 21 candidates)
- [ ] **Task 1.7 — Combat + Negotiation + Downtime** (20857–23215; 82 candidates, mostly structural — careful adjudication)
- [ ] **Task 1.8 — Tests + Skills + Treasures + Gods** (20168–20856, 23221–25258, 26340–27294; 29 candidates)
- [ ] **Task 1.9 — Phase 1 checkpoint:** full `validate` + `gen`; record new code count delta in `classification.json`; commit. Regenerate `/tmp/codes.txt` for Phase 2.

Each task commits: `link: annotate missing <chapter> feature headers`.

## Phase 2 — Re-link truncated references (Bucket A, expanded by Phase 1)

Re-run the Bucket A detector against the **post-Phase-1** registry. For every truncation site, read it in context and replace `[Short](scc:wrong) Rest…` with `[Short Rest](scc:correct-code)`, removing the stale prefix link. Per-site adjudication required for:
- **Multi-target codes:** `null-field` exists as **both** `feature.ability.null.level-1/null-field` and `feature.trait.null.level-1/null-field` — choose by context (the surrounding text/level table).
- **Wrong-type fixes:** `shadow-born` → `complication/shadow-born`; `criminal-underworld` → `skill/criminal-underworld`. Verify the context genuinely refers to that entity (not the class/career).
- Confirm the *whole* phrase is the entity name (don't over-extend into following words).

Suggested grouping (commit per group): Null family (`null-field` ×29, `null-speed`, `null-tradition`) · Ward family (`talent/conduit/elementalist-ward`) · `*-tradition`/`censor-order`/`shadow-college`/`troubadour-class-act` · wrong-type (`shadow-born`, `criminal-underworld`) · any new Phase-1-created targets (e.g. `fury-abilities`).

- [ ] **Task 2.1 — Null family re-links**
- [ ] **Task 2.2 — Ward + tradition + class-specific feature re-links**
- [ ] **Task 2.3 — Wrong-type re-links (shadow-born → complication, criminal-underworld → skill)**
- [ ] **Task 2.4 — Phase-1-created targets (fury-abilities and any siblings) re-links**
- [ ] **Task 2.5 — Re-run detector; confirm 0 remaining combined-phrase→code truncations**

## Phase 3 — Unlink genuine false positives (Bucket B residue)

Capitalized-continuation sites that resolve to **no** code and are **not** a real entity (proper names like "Polder Jackson", ordinary usage like "Human Culture"/"Human Languages" where it's a section label, "Conduit Domain"/"Censor Domain"/"Elementalist Specialization" if not adjudicated as features in Phase 1). Strip the partial link to plain text. **Do not** touch legitimate references such as "the [fury's](scc:.../class/fury) Heroic Resource" where the class link is correct.

- [ ] **Task 3.1 — Adjudicate & unlink false-positive prefix links** (commit: `fix: unlink false-positive truncated class/career links`)

## Phase 4 — Final validation

- [ ] **Step 1:** `go test ./... -race` → all pass.
- [ ] **Step 2:** `gen` → **0 WARN**.
- [ ] **Step 3:** `validate` → coverage improved, 0 unknown @type, 0 duplicate codes.
- [ ] **Step 4:** Confirm 0 remaining Bucket A truncations (detector clean) and no legacy colon links.
- [ ] **Step 5:** Spot-check linked output for the fixed targets (e.g. `null-field`, `shadow-born`, `fury-abilities`) in `../data/data-rules/en/md-linked/`.
- [ ] **Step 6:** Update `docs/linking-guide.md` progress matrix / add a note about the annotation+re-link pass.
- [ ] **Step 7:** Final commit.

## File Map

| File | Action | Responsibility |
|---|---|---|
| `input/heroes/Draw Steel Heroes.md` | Modify (all phases) | Add annotations (P1), re-link truncations (P2), unlink false positives (P3) |
| `classification.json` | Regenerated | Never hand-edited; verify new codes appear |
| `docs/linking-guide.md` | Modify (P4) | Progress note |

## Risks

- **`annotate_heroes.py` divergence:** per decision, `.md` is canonical; the `.py` is not updated, so a future re-run would regress. Flag prominently; out of scope here.
- **Over-annotation:** annotating a structural header creates a bogus code. Adjudicate conservatively; when unsure, leave unannotated and add a `<!-- REVIEW: -->` note.
- **Level/class mis-derivation:** a feature annotation placed outside the correct `feature-group`/class context yields a wrong code. Verify each new code in the regenerated registry.
- **Link-density downstream:** re-targeted links change which survive `--link-mode=first`; acceptable, but spot-check.
