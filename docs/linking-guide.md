# Linking Guide

Instructions for adding scc: cross-reference links to the input document
(`input/heroes/Draw Steel Heroes.md`). Designed to be picked up by any AI
session and followed step by step.

> **2026-05-29 — Truncated-link fix pass.** Iterative linking order (broad terms
> linked before narrow) left some links pointing at a truncated/wrong code. Fixed:
> (1) annotated missing feature headers so their codes generate — 9 class "X Abilities"
> traits, 56 per-level/cost ability-groups, 14 ancestry signature traits, 68 purchased
> traits; (2) added new `project` (16) and `god` (9) SCC types + parsers; (3) re-linked
> ~150 truncated references to the correct (often different-type) code — e.g.
> `class/shadow`→`complication/shadow-born`, `career/criminal`→`skill/criminal-underworld`,
> `class/null`→`feature.trait.../null-field` (×32); (4) unlinked false positives where the
> leading word was part of a named benefit/item (Devil's Bargain, Noble Splendor, etc.).
> Registry 1,582→1,754. See `docs/superpowers/plans/2026-05-29-truncated-link-fix.md`.
> **Rule reinforced:** when a short term is the prefix of a longer game entity, link the
> *whole* entity to its own code, not just the prefix.

> **2026-06-07 — New `rule.<group>/<term>` type (rules-glossary linking).** Added a
> grouped `rule` SCC type so every Draw Steel rules/glossary term is linkable. 108 codes
> across 12 groups: `dice`, `character`, `health`, `resource`, `combat`, `damage`, `test`,
> `downtime`, `negotiation`, `treasure`, `world`, `general`. Each code is anchored to the
> existing rules section that defines the term (e.g. `rule.combat/flanking` → `### Flanking`,
> `rule.health/winded` → `#### Winded`); terms with no own heading either map to the nearest
> headed parent (e.g. `square` → `rule.combat/distance`) or already-typed code. The
> Introduction glossary headwords now link to these. Terms already covered by another type
> (conditions, movement, skills, classes, …) keep linking to that type — see
> `docs/rule-term-mapping.md` for every term's decision. **Disambiguation:** most rule terms
> are common English words (edge, bane, cover, size, distance, surge, strike, line, wall,
> creature, hero, level) — link ONLY the game-mechanic use, never ordinary prose. Full term
> list in `docs/linking-reference.md` (`## Rules`); design record in
> `docs/superpowers/plans/2026-06-07-rule-glossary-scc-linking.md`.

> **2026-06-06 — Audit pass (kits, combat mechanics, gods).** A whole-document audit
> (`scripts/link_audit.py`) surfaced gaps the earlier passes missed:
> (1) **Hyphen/spacing variants** — the reference lists "Rapid Fire" (space) but the doc
> writes "Rapid-Fire" (hyphen), so it was skipped in the kit table while every other kit
> was linked. **Always check hyphen↔space and plural/possessive variants of a term.**
> (2) **Combat mechanics were never linkable** — the `feature.{trait,ability}.common.*`
> move actions / maneuvers / free strikes are absent from `linking-reference.md` and were
> never linked. Linked the unambiguous ones (Advance/Disengage/Ride move actions; Catch
> Breath, Escape Grab, Aid Attack, Search for Hidden Creatures, Use Consumable, Stand Up
> maneuvers); the common-verb ones (Hide, Charge, Grab, Knockback, Heal, Defend) and
> Free Strike remain — see `FOLLOWUPS.md` #6. (3) Linked distinctive gods (Cavall, etc.,
> own-section excluded) and the Templar `Domain Piety and Effects` truncation.
> **Tooling:** `scripts/link_audit.py`, `link_audit_category.py`, `link_apply.py`.

> **2026-06-07 — Audit tail closed (FOLLOWUPS #6 A+B+C).** Cleared the deferred tail (+27
> links, ~4,595 → ~4,622; `gen` 0 WARN). (B) Read every remaining unlinked common-verb
> maneuver/action occurrence (Hide/Charge/Grab/Defend/Heal/Knockback/…); only **7** were
> genuine mechanic refs — the rest are mundane, the Hide/Ride *skills*, the "Disengage Bonus"
> stat label, ability *names*, or own-section definitions. (C) Linked each **Heroic Resource**
> (wrath/piety/essence/ferocity/discipline/insight/focus/clarity/drama) only at its genuine
> cross-references — the Introduction glossary defn line + the "Heroic Resources" overview list
> + 2 other-chapter mechanic refs (**20** links) — having confirmed (via the new section-aware
> `scripts/link_audit_sectioned.py`) that each resource sits ≈80–95% **inside its own class
> section** and was already linked once in its class progression table. **Rule clarified:** a
> term that exists only as **per-class codes** (`Triggered Action`, `Signature Ability`,
> `Skill`, `Perk` — every class has its own) has **no single canonical target**, so generic
> prose uses are left unlinked rather than pointed at one arbitrary class. Heroic resources are
> linked **sparingly** (cross-references only, never within-class/flavor). See
> `docs/linking-reference.md` (Heroic Resources + "deliberately NOT linkable" sections).

## Reference

- **Reference table:** `docs/linking-reference.md` — all linkable terms with display names, variants, and SCC codes
- **Link format:** `[Display Text](scc:mcdm.heroes.v1/type/id)`
- **Input document:** `input/heroes/Draw Steel Heroes.md`

## Linking Rules

### Link when

- The term refers to a game mechanic (a class, career, ancestry, kit, condition, skill, etc.)
- Link ALL instances of the term — density filtering is handled by the pipeline at build time
- Bolded terms that reference game mechanics (e.g., glossary: `**Criminal:** a career choice...` should become `**[Criminal](scc:mcdm.heroes.v1/career/criminal):** a career choice...`)
- Terms inside nested child sections of their own parent definition (e.g., "Fury" mentioned in a Fury ability description — when extracted, the ability page needs a link back to its class)

### Don't link when

- The term is used as ordinary English, not referencing the game mechanic ("fighting criminals" ≠ the Criminal career)
- The term appears in its own section heading (`## Fury` does not link to itself)
- The text is inside an annotation comment (`<!-- @type: ... -->`)

### Mundane vs. game-mechanic disambiguation

Many new linkable terms (conditions, skills, negotiation motivations, movement types, cultures) are common English words. **Each instance must be evaluated individually** — scripted regex replacement is not appropriate for these types.

#### Conditions

Conditions (bleeding, dazed, frightened, grabbed, prone, restrained, slowed, taunted, weakened) refer to specific game status effects. Link when the text describes a creature having or gaining the condition as a game mechanic:

- **Link:** "the target is dazed", "deals bleeding damage", "the creature is grabbed", "a prone creature"
- **Don't link:** "she grabbed the sword" (mundane verb), "prone to errors" (adjective), "the frightened villagers fled" (emotion, not the game condition)

Key test: if you could replace the word with "has the [X] condition" and it still makes sense, it's a game reference.

#### Skills

Skills (climb, hide, intimidate, ride, etc.) refer to specific game skills that grant +2 on tests. Link when the text names the skill as a game mechanic:

- **Link:** "a Might test using the Climb skill", "the Hide skill", "using Intimidate"
- **Don't link:** "climb the wall" (mundane verb), "hide behind a barrel" (mundane verb), "ride a horse" (mundane verb, unless specifically referencing the Ride skill)

Key test: is the text talking about the named skill mechanic, or just using the word as a verb/noun?

#### Negotiation motivations/pitfalls

Motivations (benevolence, discovery, freedom, greed, higher authority, justice, legacy, peace, power, protection, revelry, vengeance) are negotiation system terms. Link when the text references them as NPC traits in the negotiation system:

- **Link:** "an NPC with the benevolence motivation", "the discovery pitfall", "motivations and pitfalls"
- **Don't link:** "the power of the gods" (general noun), "in the interest of peace" (general concept), "a legacy of war" (general noun)

#### Movement types

Movement terms (forced movement, shifting, teleport, fly, burrow, etc.) are game mechanics. Link when used as mechanical game terms:

- **Link:** "a creature who is force moved", "the creature can shift 1 square", "teleport 5 squares"
- **Don't link:** "shift your weight" (mundane), "fly into a rage" (metaphorical)

#### Culture types

Culture terms (nomadic, rural, urban, bureaucratic, etc.) refer to the culture benefit system. Link when referencing the specific culture type in the character creation system:

- **Link:** "a nomadic culture", "the bureaucratic organization", "the martial upbringing"
- **Don't link:** "the nomadic tribes" (flavor text), "urban sprawl" (mundane adjective)

#### Projects and gods

Downtime projects (`project/*`) and deities (`god/*`) became linkable 2026-05-29. Link when a project or god is named in prose outside its own section:

- **Link:** "complete the Build Airship project", "a temple to Cavall", "blessed by Val"
- **Don't link:** the project's/god's own section heading; short god names (Val, Ord, Kul) used as ordinary words

### Case and variants

- Match case-insensitively: "fury", "Fury", and "FURY" all match
- Handle plurals: "criminals" should link with display text "criminals" to the Criminal career SCC code
- Handle possessives: "Fury's" should link "Fury's" to the Fury class SCC code (include the possessive in the display text)
- Use the reference table for known plural forms; use judgment for unlisted variants

### Pre-existing links

- **First pass (complete):** All chapters have been stripped of stale links and re-linked for classes, ancestries, and chapters.
- **Second pass (complete, 2026-05-28):** Added new type links (conditions, skills, negotiations, movements, cultures) to all 18 chapters. 1,229 new links added; 2,818 total SCC references at that point. Each instance was AI-evaluated for mundane vs. game-mechanic disambiguation.
- **Truncated-link fix (complete, 2026-05-29):** Re-pointed ~150 truncated references to the correct (often different-type) code, annotated missing feature headers (+147 trait codes), and added `project`/`god` types. Current total: ~4,055 SCC references. See the note at the top of this file.

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
3. Read the full reference table (`docs/linking-reference.md`)
4. Add scc: links for game mechanic references, evaluating each instance individually
5. Use `<!-- REVIEW: -->` markers for uncertain cases
6. Update the progress matrix
7. Commit: `git commit -m "link: add {type} cross-reference links to {chapter} chapter"`

### Validation

After completing all chapters, run the pipeline and check for warnings:

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml' 2>&1 | grep WARN
```

Warnings indicate unresolved SCC codes (typos or missing registry entries).

## Progress Matrix

| Chapter | Lines | Conditions | Skills | Negotiations | Movements | Cultures | Classes | Ancestries | Careers | Kits | Perks | Complications | Titles | Treasures | Chapters |
|---------|-------|-----------|--------|-------------|-----------|----------|---------|-----------|---------|------|-------|--------------|--------|-----------|----------|
| Introduction | 7-589 | done (18) | done (0) | done (0) | done (17) | done (0) | done (28) | done (22) | - | done (1) | - | - | - | - | done (3) |
| The Basics | 590-1055 | done (5) | done (0) | done (0) | done (1) | done (0) | done (17) | done (36) | - | - | - | - | - | - | done (11) |
| Making a Hero | 1056-1263 | done (0) | done (0) | done (0) | done (0) | done (0) | done (12) | done (18) | - | - | - | - | - | - | done (15) |
| Ancestries | 1264-3199 | done (16) | done (1) | done (0) | done (7) | done (0) | done (22) | done (324) | - | - | - | - | - | - | done (11) |
| Background | 3200-3206 | done (0) | done (0) | done (0) | done (0) | done (0) | - | - | - | - | - | - | - | - | - |
| Cultures | 3207-3493 | done (0) | done (13) | done (0) | done (0) | done (21) | - | done (47) | - | - | - | - | - | - | done (2) |
| Careers | 3494-4065 | done (0) | done (17) | done (0) | done (0) | done (0) | done (4) | - | done (25) | - | done (13) | - | - | - | done (5) |
| Classes | 4066-17606 | done (434) | done (34) | done (0) | done (257) | done (0) | done (316) | done (4) | done (2) | done (18) | done (1) | done (14) | - | - | done (67) |
| Kits | 17607-18580 | done (8) | done (0) | done (0) | done (7) | done (0) | done (5) | - | - | done (32) | - | - | - | - | done (4) |
| Perks | 18581-18946 | done (0) | done (1) | done (0) | done (1) | done (0) | - | - | - | done (19) | done (11) | - | - | - | done (2) |
| Complications | 18947-20167 | done (14) | done (2) | done (0) | done (2) | done (0) | done (12) | done (8) | - | - | - | done (233) | done (1) | - | done (18) |
| Tests | 20168-20408 | done (1) | done (0) | done (0) | done (1) | done (0) | - | done (1) | - | - | - | - | - | - | done (2) |
| Skills | 20409-20856 | done (0) | done (14) | done (0) | done (0) | done (0) | done (12) | done (3) | - | - | - | - | - | - | done (3) |
| Combat | 20857-21636 | done (28) | done (0) | done (0) | done (62) | done (0) | done (11) | done (9) | - | done (1) | - | - | - | - | done (13) |
| Negotiation | 21637-22187 | done (1) | done (4) | done (32) | done (1) | done (0) | done (4) | done (7) | - | - | - | - | - | - | done (2) |
| Downtime Projects | 22188-23215 | done (16) | done (1) | done (0) | done (15) | done (0) | done (12) | done (14) | - | done (3) | - | - | done (1) | - | done (5) |
| Rewards | 23216-23220 | done (0) | done (0) | done (0) | done (0) | done (0) | - | - | - | - | - | - | - | - | - |
| Treasures | 23221-25258 | done (42) | done (1) | done (0) | done (28) | done (0) | done (5) | done (8) | - | done (1) | - | - | - | done (4) | done (9) |
| Titles | 25259-26339 | done (22) | done (3) | done (0) | done (7) | done (0) | done (12) | done (21) | - | done (1) | - | - | done (22) | done (1) | done (16) |
| Gods and Religion | 26340-27294 | done (4) | done (0) | done (0) | done (1) | done (0) | done (48) | done (123) | - | - | - | - | done (20) | - | - |
| For the Director | 27295-28721 | done (0) | done (0) | done (0) | done (4) | done (0) | done (11) | done (8) | - | - | - | done (1) | done (1) | - | done (14) |
