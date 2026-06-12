# Monsters Content Linking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `scc:` cross-reference links throughout the Monsters book source (`input/monsters/Draw Steel Monsters.md`, ~29,062 lines, currently **0** links) — links *out of* and *within* the Monsters source only (FOLLOWUPS #5 direction 1). First **mint a monster rule-glossary** (`rule.monster/*`, `rule.role/*`, `rule.organization/*`, `rule.keyword/*`) from the Monster Basics chapter so the book's pervasive vocabulary (Malice, minion, squad, captain, Encounter Value, the 9 creature roles, the 6 organizations, the general creature keywords) becomes linkable, then sweep the whole book.

**Architecture:** Linking is a manual, per-instance editing pass on one annotated markdown file, governed by the disambiguation rules in `docs/linking-guide.md`. Three reference tables drive it: the existing `docs/linking-reference.md` (Heroes, cross-book targets — conditions, movement, combat, characteristics, damage immunity/weakness, rules-glossary) for shared terms; a **new** `docs/monsters-linking-reference.md` (the Monsters book's own codes) for internal links; and `docs/linking-guide.md` for the link/don't-link judgment. Phase 0 mints the monster rule-glossary by adding `<!-- @type: rule | @group: … | @id: … -->` annotations to selected Monster Basics headings (the same mechanism the Heroes book used on 2026-06-07; `RuleParser` joins `@group` into a `rule.<group>` SCC type). Phase 1 hardens the one remaining statblock-parser regex (`sbPowerRollRe`, the Monsters labeled power-roll form) against link-wrapping — the other statblock regexes were already hardened for the Summoner book. Phase 2 sweeps the source in 13 section batches, validated with `gen --all` (no-new-WARN gate) plus JSON/card spot-checks. Phase 3 verifies; Phase 4 syncs docs.

**Tech Stack:** Go (steel-etl pipeline + statblock parser, `go test`), annotated Markdown, devbox toolchain. **All Go/just commands MUST be wrapped** `devbox run -- bash -c 'cd steel-etl && go …'` (Go is not on PATH; a bare `devbox run -- go …` runs from the workspace root, which has no `go.mod`, and fails silently — memory `devbox-go-invocation`).

---

## Build & Validation Conventions (READ FIRST)

**⚠️ `devbox run` resets the working directory to the devbox project root (the workspace root, `/home/scott/code/steelCompendium/workspace`), which has no `go.mod`.** A bare `devbox run -- go run ./cmd/steel-etl …` fails with `go: cannot find main module` and **exits 1 but prints no `WARN`** — a silent trap that looks like a clean build. **Always `cd steel-etl` inside the wrapper**, and always check the exit status:

```bash
# Canonical full build (run from anywhere):
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all' > /tmp/gen.log 2>&1
echo "exit=$?"   # MUST be 0. Non-zero = build error (read /tmp/gen.log tail), not a link problem.
```
The same wrapper applies to `go test` / `go build` / `validate`:
`devbox run -- bash -c 'cd steel-etl && go test ./...'`.

**Why `--all`:** a bare `gen` processes only the primary book (heroes) and **skips** the `books:` list (monsters/beastheart/summoner) — `selectBookConfigs` in `internal/cli/gen.go`. The Monsters book is `mcdm.monsters.v1` and outputs to the **workspace-level** `data/data-bestiary/` (per `pipeline.yaml` `books[].base_dir: ../data/data-bestiary`), NOT `steel-etl/data/`. Cross-book links to Heroes resolve only when Heroes is in the same registry, so **always validate with `--all`** (or at minimum `--book monsters`, which merges into the existing registry).

**The WARN gate is NOT "0 WARN".** The real baseline (`gen --all` on a clean tree) has **104 WARN / 12 distinct unresolved codes**, all pre-existing stale flat `mcdm.heroes.v1/skill/<x>` links from `input/beastheart/Draw Steel Beastheart.md` (never repointed after the 2026-06-08 skill-group nesting — out of scope here). Establish the baseline once at the start (Task 0.1) and save the distinct-code set to `/tmp/scc-warn-baseline-codes.txt`.

**Validation gate = "no NEW unresolved code vs. baseline":**
```bash
cd /home/scott/code/steelCompendium/workspace
grep "WARN: unresolved scc link" /tmp/gen.log | sed -E 's/.*link "([^"]+)".*/\1/' | sort -u > /tmp/warn-now.txt
comm -13 /tmp/scc-warn-baseline-codes.txt /tmp/warn-now.txt   # MUST print nothing
```
Any line printed is a NEW unresolved code introduced by this pass (a typo'd target or a missing registry entry) — **fix before committing.** Note: links to nested skills MUST use the grouped form `skill.<group>/<item>` (e.g. `skill.intrigue/hide`); the flat `skill/<x>` form is exactly the stale baseline bug.

**Code-count guard.** Record the Monsters code count after Phase 0 and assert it never changes during the Phase 2 link sweep (links never mint codes):
```bash
grep -oE "mcdm\.monsters\.v1/[a-z0-9./-]+" steel-etl/classification.json | sort -u | wc -l
```
Before Phase 0 this is **591**. After Phase 0 it rises by the number of glossary codes minted (recorded in Task 0.3). During Phase 2 it must stay fixed at the post-Phase-0 number.

---

## Background: what was learned during planning

- **Source:** `input/monsters/Draw Steel Monsters.md`, 29,062 lines, **0** existing `scc:` links. Frontmatter `book: mcdm.monsters.v1`, `printing: "1.01"`.
- **Existing own codes (591):** statblocks `monster.<category>[.<subcategory>].statblock/<id>`; group landings `monster.group/<category>`; malice featureblocks `monster.<category>[.<subcategory>]/<id>`; `retainer.statblock/<id>`; `dynamic-terrain.<category>/<id>`; `chapter/*` (4). `<subcategory>` is an echelon (`1st-echelon`…) for Rivals/Demons/Undead/War Dogs. (Reference: `docs/statblocks.md`.)
- **The Monster Basics chapter has NO rule codes of its own** — its vocabulary (Malice, minion, squad, captain, Encounter Value, the 9 roles, the 6 organizations, ~16 general keywords, villain actions, end effects, monster traits, creature free strikes) has no SCC target in any book. **Phase 0 mints these** (user decision: "mint a monsters rule-glossary first").
- **Heroes targets exist** (link cross-book, do NOT re-mint) for the shared vocabulary: conditions (`condition/*`), movement (`movement/*`), combat actions & maneuvers (`feature.{trait,ability}.common/*` — Heal/Defend/Hide/Grab/free strike/etc.), characteristics + stability + potency + saving throw (`rule.character/*`, `rule.general/saving-throw`), Stamina/winded/Recoveries (`rule.health/*`), surge (`rule.resource/surge`), damage immunity/weakness (`rule.damage/*`), power-roll/edge/bane/tier (`rule.dice/*`), skills (`skill.<group>/<item>`), and **`rule.combat/signature-ability`** (already linkable — do NOT mint a monster duplicate). The Monsters book leans heavily on these.
- **Statblock parser is already mostly hardened** against `scc:` link-wrapping (done 2026-06-11 for the Summoner book — `docs/statblocks.md`): `sbDiceRe` accepts a link-wrapped characteristic, `linkDisplay` strips link markup from the structured `roll` and stat-grid cell values, `sbTierRe`/`sbBareTierRe` value groups are link-agnostic. **The one remaining risk is `sbPowerRollRe`** (`\*\*(Power Roll[^*]*)\*\*`, the Monsters **labeled** form `**Power Roll + 2:**`), which `docs/linking-guide.md` flags is "spared only because the Monsters book isn't link-swept yet — it will hit the same wall when it is." Phase 1 addresses it.
- **Power-roll forms in this book:** 783 labeled headers — `**Power Roll + 2:**` (281), `+ 3` (184), `+ 4` (144), `+ highest characteristic` (79), `+ 5` (75). The modifier is a **flat number** (or the literal phrase "highest characteristic"), never a single-characteristic link. **Rule: never link the `**Power Roll + N:**` header line** (a structured-block label — same convention as Heroes; `linking-guide.md` footgun). With the header left plain, `sbPowerRollRe` is safe in practice; Phase 1 hardens it defensively anyway as a regression guard against a future editor.
- **Statblock stat-grid is left plain.** Keyword/meta rows (`**1S**<br>Size`, `**Poison 2**<br>Immunity`, `**Climb, swim**<br>Movement`, the M/A/R/I/P row) are structured cells parsed by `cellRe`; leave their **labels** plain. You *may* link a Movement-cell value that is a real movement type (e.g. `Teleport`, `Fly`, `Burrow`) — optional, low value; `cellRe` + `linkDisplay` already tolerate it. **Creature keyword cells** (the first row, e.g. `Angulotl, Humanoid`) link only the *general* keyword to its new `rule.keyword/<id>` code where one exists (e.g. Humanoid), and the group keyword (Angulotl) to `monster.group/angulotls` where sensible — but this is low-value and easy to overlink, so prefer linking keywords **in prose** (the Keywords definitions, group lore) over inside every stat grid.
- **Per-instance judgment, not scripted replacement.** Most new terms are common English words (minion, brute, support, mount, beast, plant, swarm, animal). Each occurrence is evaluated individually with the disambiguation key test ("could you replace it with 'the [X] game term' and have it still make sense?").

---

## File Structure

| File | Responsibility | Phase |
|------|----------------|-------|
| `input/monsters/Draw Steel Monsters.md` | **Modify.** (a) Phase 0: add `@type: rule` annotations to Monster Basics headings to mint the glossary. (b) Phase 2: the link sweep, batch by batch. | 0, 2 |
| `docs/monster-rule-mapping.md` | **New.** The authoritative heading → `rule.<group>/<id>` decision record for every minted glossary code (mirrors Heroes' `docs/rule-term-mapping.md`), incl. terms deliberately left to a Heroes cross-book target. | 0 |
| `docs/monsters-linking-reference.md` | **New.** Canonical list of the Monsters book's own linkable terms (display name, variants, SCC code) — statblocks, groups, terrain, retainers, and the new glossary — + a disambiguation table for common-word terms. | 0 |
| `internal/content/statblock_parse.go` | **Modify.** Harden `sbPowerRollRe` to tolerate a link-wrapped "Power Roll" (defensive regression guard). | 1 |
| `internal/content/statblock_parse_test.go` | **Modify.** Table-driven test for the labeled power-roll header with/without link-wrapping. | 1 |
| `docs/linking-guide.md` | **Modify.** Add a dated Monsters note + a Monsters progress matrix; codify the "never link the `**Power Roll + N:**` header" rule and the monster-glossary groups. | 4 |
| `docs/linking-reference.md` | **Modify.** Add a `## Monsters book` pointer section referencing the new reference file. | 4 |
| `steel-etl/CLAUDE.md` | **Modify.** Note Monsters is link-swept; `sbPowerRollRe` hardened; the new `rule.{monster,role,organization,keyword}` groups. | 4 |
| `../CLAUDE.md` (workspace) | **Modify.** Update the SCC registry paragraph (new code total, Monsters link-swept) + the linking bullet. | 4 |
| `../docs/scc-log.md` | **Modify.** Dated entry: new monster rule-glossary groups + Monsters link sweep. | 4 |
| `../FOLLOWUPS.md` | **Modify.** Mark #5 done (direction 1); note direction 2 (links *into* monsters) remains, if desired, as a new follow-up. | 4 |
| memory + `MEMORY.md` | **Modify.** Record Monsters link sweep + new glossary groups. | 4 |

---

## Phase 0 — Baseline, mint the monster rule-glossary, build references

### Task 0.1: Establish the clean baseline

**Files:** none (read-only verification).

- [ ] **Step 1: Confirm zero links and capture the WARN baseline**

```bash
cd /home/scott/code/steelCompendium/workspace
grep -c "](scc:" "steel-etl/input/monsters/Draw Steel Monsters.md"   # expect 0
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all' > /tmp/gen.log 2>&1
echo "exit=$?"   # MUST be 0
grep -c "WARN" /tmp/gen.log   # ~104 (the beastheart baseline)
grep "WARN: unresolved scc link" /tmp/gen.log | sed -E 's/.*link "([^"]+)".*/\1/' | sort -u > /tmp/scc-warn-baseline-codes.txt
wc -l /tmp/scc-warn-baseline-codes.txt   # expect 12 distinct codes
```
Expected: 0 links in the source; build exit 0; ~12 distinct baseline codes saved (all `mcdm.heroes.v1/skill/{alertness,endurance,handle-animals,hide,intimidate,magic,nature,navigate,read-person,search,sneak,track}`).

- [ ] **Step 2: Record the pre-glossary Monsters code count**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
grep -oE "mcdm\.monsters\.v1/[a-z0-9./-]+" classification.json | sort -u | wc -l   # expect 591
```

### Task 0.2: Annotate the Monster Basics chapter to mint the glossary

**Files:**
- Modify: `input/monsters/Draw Steel Monsters.md` (annotations only — no link edits yet)

The minting mechanism: a heading preceded by `<!-- @type: rule | @group: <group> | @id: <id> -->` is parsed by `RuleParser` (`internal/content/rule.go`) into `mcdm.monsters.v1/rule.<group>/<id>`. `RuleParser` ignores accumulated parent path, so codes stay flat within their group regardless of heading nesting. **One heading = one code** (memory `rule-scc-type`): annotate the heading that *defines* the term; never annotate two headings to the same id.

**⚠️ Before minting `rule.monster/<x>`, check `docs/linking-reference.md` for an existing Heroes target.** If one exists, do NOT mint — the term will link cross-book instead. Pre-resolved exclusions: **Signature Ability** → use Heroes `rule.combat/signature-ability`; **free strike** (generic) → use the Heroes combat-action target; **Stamina/Stability/Recoveries/Victories/saving throw/conditions/movement/characteristics/damage types** → all Heroes targets. (The Monsters "Creature Free Strikes" heading defines a distinct *monster* rule, so `rule.monster/creature-free-strike` IS minted; generic "free strike" prose still links cross-book.)

- [ ] **Step 1: Add the `rule.monster/*` core-concept annotations**

Add each comment line on its own line **immediately above** the named heading (re-`grep -n` each heading first; line numbers below are approximate and will drift):

| Heading (level + text) | Annotation to insert above it |
|------------------------|-------------------------------|
| `### Malice` (the dedicated section, ~line 313 — NOT the brief `#### Malice` mention under Monster Basics ~line 181) | `<!-- @type: rule | @group: monster | @id: malice -->` |
| `#### Encounter Value` (~146) | `<!-- @type: rule | @group: monster | @id: encounter-value -->` |
| `#### Creature Free Strikes` (~154) | `<!-- @type: rule | @group: monster | @id: creature-free-strike -->` |
| `#### Traits` (~177) | `<!-- @type: rule | @group: monster | @id: monster-trait -->` |
| `#### End Effect` (~185) | `<!-- @type: rule | @group: monster | @id: end-effect -->` |
| `#### Villain Actions` (~189) | `<!-- @type: rule | @group: monster | @id: villain-action -->` |
| `#### Keywords` (~70) | `<!-- @type: rule | @group: monster | @id: keyword -->` |
| `#### Organized as Squads` (~359, under `### Using Minions`) | `<!-- @type: rule | @group: monster | @id: squad -->` |
| `#### Attached Squad Captain` (~429) | `<!-- @type: rule | @group: monster | @id: captain -->` |

Leave the brief `#### Malice` overview mention (~181) and `#### Signature Ability` (~173) **unannotated** (the former would duplicate `rule.monster/malice`; the latter links cross-book to Heroes).

- [ ] **Step 2: Add the `rule.organization/*` annotations** (the six `##### ` headings under `#### Creature Organization`, ~211)

| Heading | Annotation |
|---------|-----------|
| `##### Minion` | `<!-- @type: rule | @group: organization | @id: minion -->` |
| `##### Horde` | `<!-- @type: rule | @group: organization | @id: horde -->` |
| `##### Platoon` | `<!-- @type: rule | @group: organization | @id: platoon -->` |
| `##### Elite` | `<!-- @type: rule | @group: organization | @id: elite -->` |
| `##### Leader` | `<!-- @type: rule | @group: organization | @id: leader -->` |
| `##### Solo` | `<!-- @type: rule | @group: organization | @id: solo -->` |

- [ ] **Step 3: Add the `rule.role/*` annotations** (the nine `##### ` headings under `#### Creature Roles`, ~239)

| Heading | Annotation |
|---------|-----------|
| `##### Ambusher` | `<!-- @type: rule | @group: role | @id: ambusher -->` |
| `##### Artillery` | `<!-- @type: rule | @group: role | @id: artillery -->` |
| `##### Brute` | `<!-- @type: rule | @group: role | @id: brute -->` |
| `##### Controller` | `<!-- @type: rule | @group: role | @id: controller -->` |
| `##### Defender` | `<!-- @type: rule | @group: role | @id: defender -->` |
| `##### Harrier` | `<!-- @type: rule | @group: role | @id: harrier -->` |
| `##### Hexer` | `<!-- @type: rule | @group: role | @id: hexer -->` |
| `##### Mount` | `<!-- @type: rule | @group: role | @id: mount -->` |
| `##### Support` | `<!-- @type: rule | @group: role | @id: support -->` |

- [ ] **Step 4: Add the `rule.keyword/*` annotations** (the general-keyword `###### ` headings under `##### General Keywords`, ~88)

Read the full run of `######` headings under `##### General Keywords` (the visible set starts Abyssal, Accursed, Animal, Beast, Construct, Dragon, Elemental, Fey, Giant, Horror, Humanoid, Infernal, Ooze, Plant, Soulless, Swarm — **continue past Swarm to the end of the General Keywords subsection** and capture every one, e.g. Undead, etc.). Annotate each with its slug:

```
<!-- @type: rule | @group: keyword | @id: abyssal -->
###### Abyssal
<!-- @type: rule | @group: keyword | @id: accursed -->
###### Accursed
…one per general keyword, @id = lowercased heading…
```

Do **not** annotate group-specific keyword mentions (Goblin, Gnoll, Human) — those map to their `monster.group/<category>` landing, not a `rule.keyword` code.

- [ ] **Step 5: Build and confirm the glossary codes mint cleanly, the chapter still renders, and nothing collides**

```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all' > /tmp/gen.log 2>&1
echo "exit=$?"   # MUST be 0
# New rule codes present:
grep -oE "mcdm\.monsters\.v1/rule\.(monster|role|organization|keyword)/[a-z-]+" steel-etl/classification.json | sort -u
# No new unresolved warnings (annotations don't add links, but a malformed annotation can error):
grep "WARN: unresolved scc link" /tmp/gen.log | sed -E 's/.*link "([^"]+)".*/\1/' | sort -u > /tmp/warn-now.txt
comm -13 /tmp/scc-warn-baseline-codes.txt /tmp/warn-now.txt   # MUST print nothing
# The Monster Basics chapter page still renders the rule prose inline:
ls data/data-bestiary/en/md-linked/rule/ 2>/dev/null && echo "--- chapter still present ---" && ls data/data-bestiary/en/md-linked/chapter/ 2>/dev/null
```
Expected: exit 0; the `rule.monster/*`, `rule.role/*`, `rule.organization/*`, `rule.keyword/*` codes listed; `comm` prints nothing; a `rule/` tree exists under `data/data-bestiary/en/md-linked/` and the `chapter/` tree still exists. If a rule page failed to generate or the chapter fragmented oddly, STOP and inspect the annotation placement (a stray blank line between the comment and the heading detaches the annotation).

- [ ] **Step 6: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add "input/monsters/Draw Steel Monsters.md"
git commit -m "feat(monsters): mint rule-glossary codes for monster basics vocabulary"
```

### Task 0.3: Author the heading→code mapping record

**Files:**
- Create: `docs/monster-rule-mapping.md`

- [ ] **Step 1: Record every minted code + every deliberate exclusion**

Create `docs/monster-rule-mapping.md` documenting the decisions from Task 0.2, so a future editor knows why each term maps where (mirrors Heroes' `docs/rule-term-mapping.md`). Fill every row from the actual annotations made (no `…` placeholders in the committed file). Skeleton:

```markdown
# Monster Rule-Glossary Term Mapping

Decision record for the Monsters-book rule-glossary (`mcdm.monsters.v1/rule.<group>/<id>`),
minted from the Monster Basics chapter on 2026-06-12. Each row: the term, its SCC code (or
the cross-book Heroes target it reuses), and the defining heading. See `linking-guide.md`
for the link/don't-link rules.

## rule.monster (core concepts)
| Term | Code | Defining heading |
|------|------|------------------|
| Malice | `rule.monster/malice` | `### Malice` |
| Encounter Value (EV) | `rule.monster/encounter-value` | `#### Encounter Value` |
| Creature free strike | `rule.monster/creature-free-strike` | `#### Creature Free Strikes` |
| Monster trait | `rule.monster/monster-trait` | `#### Traits` |
| End effect | `rule.monster/end-effect` | `#### End Effect` |
| Villain action | `rule.monster/villain-action` | `#### Villain Actions` |
| Keyword | `rule.monster/keyword` | `#### Keywords` |
| Squad | `rule.monster/squad` | `#### Organized as Squads` |
| Captain | `rule.monster/captain` | `#### Attached Squad Captain` |

## rule.organization
| Term | Code | Defining heading |
| Minion | `rule.organization/minion` | `##### Minion` |
| … (all six) | | |

## rule.role
| Term | Code | Defining heading |
| Ambusher | `rule.role/ambusher` | `##### Ambusher` |
| … (all nine) | | |

## rule.keyword (general creature keywords)
| Term | Code | Defining heading |
| Abyssal | `rule.keyword/abyssal` | `###### Abyssal` |
| … (all general keywords) | | |

## Reused cross-book Heroes targets (NOT minted here)
| Term | Heroes code |
|------|-------------|
| Signature Ability | `mcdm.heroes.v1/rule.combat/signature-ability` |
| Free strike (generic) | (Heroes combat-action target — see linking-reference.md) |
| Stamina / winded / Recoveries | `mcdm.heroes.v1/rule.health/*` |
| Characteristics / Stability / saving throw | `mcdm.heroes.v1/rule.character/*`, `rule.general/saving-throw` |
| Conditions / movement / damage immunity-weakness / power-roll terms | Heroes `condition/*`, `movement/*`, `rule.damage/*`, `rule.dice/*` |
```

- [ ] **Step 2: Verify every code in the mapping exists in the registry**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
grep -oE "mcdm\.monsters\.v1/[a-z0-9./-]+" classification.json | sort -u > /tmp/monster-codes.txt
grep -oE "rule\.(monster|role|organization|keyword)/[a-z-]+" docs/monster-rule-mapping.md | sed 's#^#mcdm.monsters.v1/#' | sort -u > /tmp/ref-rule-codes.txt
comm -23 /tmp/ref-rule-codes.txt /tmp/monster-codes.txt   # MUST print nothing
wc -l /tmp/monster-codes.txt   # record the new total (591 + #glossary codes)
```
Expected: `comm` prints nothing (every mapped code is real); record the new total for the Phase-2 code-count guard.

- [ ] **Step 3: Commit**

```bash
git add docs/monster-rule-mapping.md
git commit -m "docs(monsters): record rule-glossary term mapping"
```

### Task 0.4: Build the Monsters linking reference table

**Files:**
- Create: `docs/monsters-linking-reference.md`

- [ ] **Step 1: Extract the full Monsters code set with display names**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
grep -oE "mcdm\.monsters\.v1/[a-z0-9./-]+" classification.json | sort -u > /tmp/monster-codes.txt
```

- [ ] **Step 2: Author `docs/monsters-linking-reference.md`**

Create the file grouping codes by type, deriving Display Name from each code's item slug (title-cased, hyphens→spaces) cross-checked against the source headings. Fill every row (no `…` in the committed file). Skeleton:

```markdown
# Monsters Linking Reference Table

Linkable terms for the **Monsters book** (`mcdm.monsters.v1`). For cross-book links to
Heroes terms (conditions, movement, combat actions, characteristics, damage immunity/
weakness, rules-glossary) use `docs/linking-reference.md`. See `linking-guide.md` for the
link/don't-link rules and `docs/monster-rule-mapping.md` for the glossary decision record.

## Rule glossary — monster / role / organization / keyword (<N> terms)
| Display Name | Variants | SCC Code |
|-------------|----------|----------|
| Malice | malice | `mcdm.monsters.v1/rule.monster/malice` |
| Minion | minion, minions | `mcdm.monsters.v1/rule.organization/minion` |
| Brute | brute, brutes | `mcdm.monsters.v1/rule.role/brute` |
| Abyssal | abyssal | `mcdm.monsters.v1/rule.keyword/abyssal` |
<!-- …one row per rule.* code… -->

## Monster groups (<N> terms)
| Display Name | Variants | SCC Code |
|-------------|----------|----------|
| Goblins | goblin, goblins | `mcdm.monsters.v1/monster.group/goblins` |
<!-- …one row per monster.group/* code… -->

## Statblocks (<N> terms)
| Display Name | Group / echelon | SCC Code |
|-------------|-----------------|----------|
| Angulotl Cleaver | Angulotls (minion) | `mcdm.monsters.v1/monster.angulotls.statblock/angulotl-cleaver` |
<!-- …one row per *.statblock/* code… -->

## Malice featureblocks (<N> terms)
| Display Name | Group | SCC Code |
<!-- …one row per monster.<cat>/<id> (non-statblock) code… -->

## Dynamic terrain (<N> terms)
| Display Name | SCC Code |
<!-- …one row per dynamic-terrain.* code… -->

## Retainers (<N> terms)
| Display Name | SCC Code |
<!-- …one row per retainer.statblock/* code… -->

## Chapters (4 terms)
| Display Name | SCC Code |

## Disambiguation — common-word glossary terms (REQUIRED)
Many glossary terms are common English words. **Link only the game-mechanic reference,
never ordinary prose.** Per-instance judgment:

| Term | Link (game mechanic) | Don't link (ordinary) |
|------|----------------------|------------------------|
| Minion | "a minion creature", "this squad of minions", organization | "a minion of the dark lord" (flavor) |
| Brute / Support / Mount / Controller / Defender / Harrier / Hexer | "an Artillery creature", "the Brute role" | "brute force", "lend support", "her mount" (literal) |
| Solo / Elite / Horde / Platoon / Leader | the organization in a stat-block/encounter sense | "a lone elite soldier" (flavor adjective), "a horde of zombies" (flavor noun) |
| Animal / Beast / Dragon / Giant / Plant / Construct / Swarm | the creature **keyword** ("has the Beast keyword", a Beast-keyword creature) | the literal animal/plant/giant in flavor prose |
| Malice | "spend 3 Malice", "Malice features", the resource | — (almost always the mechanic) |
| Captain | "the squad captain", "With Captain" benefit | a narrative military rank in lore |
| Keyword | "the Goblin keyword", "creature keywords" | — |
| Trait | a monster's stat-block trait | "a defining character trait" (flavor) |

**Self-reference:** never link a term inside its own defining section heading/body
(the Malice section doesn't link "Malice" to itself; the Brute role page doesn't link "Brute").
```

- [ ] **Step 3: Verify every code in the reference is real**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
grep -oE "mcdm\.monsters\.v1/[a-z0-9./-]+" docs/monsters-linking-reference.md | sort -u > /tmp/ref-codes.txt
comm -23 /tmp/ref-codes.txt /tmp/monster-codes.txt   # MUST print nothing
```
Expected: no output (every referenced code exists). Any line is a typo — fix it.

- [ ] **Step 4: Commit**

```bash
git add docs/monsters-linking-reference.md
git commit -m "docs(monsters): add linking reference table for monsters book terms"
```

---

## Phase 1 — Harden `sbPowerRollRe` against link-wrapping (TDD)

Goal: make the labeled-power-roll statblock parser tolerate a link-wrapped "Power Roll" so a future editor can't silently empty the power-roll effect by linking the header. The **rule remains "don't link the header"** (Phase 2), so this is a defensive regression guard. Captured values keep raw `[text](scc:…)` verbatim; the structured roll modifier keeps its meaning.

### Task 1.1: Failing test — labeled power-roll header with link-wrapped "Power Roll"

**Files:**
- Test: `internal/content/statblock_parse_test.go`

- [ ] **Step 1: Write the failing test**

`ParseStatblockFeatures(body string) []map[string]any` returns one map per feature; the power-roll data lives under `feature["effects"].([]map[string]any)[0]` with keys `roll`/`tier1`/`tier2`/`tier3` (see the existing `TestParseStatblockFeatureDiceInTitle`). The block must be blockquote-form with the ability table, like the real tests:

```go
func TestStatblock_LabeledPowerRoll_ToleratesLinkedHeader(t *testing.T) {
	// A future editor links "Power Roll" in the labeled header form. The parser
	// must still find the power-roll block and its tiers.
	block := "" +
		"> 🗡 **Hop and Chop (Signature Ability)**\n" +
		">\n" +
		"> | **Melee, Strike, Weapon** | **Main action** |\n" +
		"> |---------------------------|----------------:|\n" +
		"> | **📏 Melee 1** | **🎯 One creature** |\n" +
		">\n" +
		"> **[Power Roll](scc:mcdm.heroes.v1/rule.dice/power-roll) + 2:**\n" +
		">\n" +
		"> - **≤11:** 2 damage\n" +
		"> - **12-16:** 4 damage\n" +
		"> - **17+:** 5 damage\n"

	got := ParseStatblockFeatures(block)
	if len(got) != 1 {
		t.Fatalf("got %d features, want 1", len(got))
	}
	effects, _ := got[0]["effects"].([]map[string]any)
	if len(effects) != 1 {
		t.Fatalf("effects = %v, want 1 (linked Power Roll header must still be recognized)", got[0]["effects"])
	}
	e := effects[0]
	if e["tier1"] == nil || e["tier1"] == "" || e["tier3"] == nil || e["tier3"] == "" {
		t.Errorf("tiers not extracted after linking the header (tier1=%v tier3=%v)", e["tier1"], e["tier3"])
	}
}
```

- [ ] **Step 2: Run it; confirm it fails**

```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestStatblock_LabeledPowerRoll_ToleratesLinkedHeader -v'
```
Expected: FAIL — `sbPowerRollRe = \*\*(Power Roll[^*]*)\*\*` requires the literal `Power Roll` immediately after `**`; with the link it starts `**[Power Roll]…` and the header is not recognized, so roll/tiers are empty.

### Task 1.2: Harden `sbPowerRollRe`

**Files:**
- Modify: `internal/content/statblock_parse.go:92`

- [ ] **Step 1: Accept an optional link-wrap on "Power Roll"** (mirror `ability.go`'s `powerRollHeaderRe`)

Keep the **whole** header inside capture group 1 (the consumer at `statblock_parse.go:231` stores `pr[1]` as the `roll` value, so "Power Roll" must remain in the group); add the link alternation *inside* the group:

```go
// sbPowerRollRe matches the Monsters labeled power-roll header "**Power Roll + N:**".
// "Power Roll" may be link-wrapped ("**[Power Roll](scc:…) + N:**"). The whole header
// stays in group 1; the consumer applies linkDisplay so the stored roll is link-free.
sbPowerRollRe = regexp.MustCompile(`\*\*((?:\[Power Roll\]\([^)]*\)|Power Roll)[^*]*)\*\*`)
```

Then update the consumer at `statblock_parse.go:231` to strip any link markup from the stored roll (a no-op on the unlinked form, so existing behavior is preserved exactly):

```go
roll = strings.TrimSuffix(strings.TrimSpace(linkDisplay(pr[1])), ":")
```

- [ ] **Step 2: Run the test; confirm it passes**

```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestStatblock_LabeledPowerRoll_ToleratesLinkedHeader -v'
```
Expected: PASS.

### Task 1.3: Full regression + commit

**Files:** none (verification).

- [ ] **Step 1: Run the whole content package with race**

```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go test -race ./internal/content/...'
```
Expected: PASS (all existing statblock tests still green — the change is additive/tolerant).

- [ ] **Step 2: Full build + test**

```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go build ./... && go test ./...'
```
Expected: build OK, all tests PASS.

- [ ] **Step 3: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add internal/content/statblock_parse.go internal/content/statblock_parse_test.go
git commit -m "fix(statblock): tolerate scc link-wrapping in labeled power-roll header"
```

---

## Phase 2 — Link sweep, section by section

**Per-pass procedure (apply to EVERY Task 2.x below):**

1. **Re-`grep -n` the section's heading anchors** (line numbers drift after each edit — never trust stale numbers). Read the section's full text.
2. For each candidate term, decide link vs. don't-link **per instance** using:
   - `docs/monsters-linking-reference.md` (the book's own terms — glossary, statblocks, groups, terrain, retainers).
   - `docs/linking-reference.md` (Heroes, all terms) for cross-book references — conditions, movement, combat actions/maneuvers, characteristics (`rule.character/*`), Stamina/winded/Recoveries (`rule.health/*`), surge, saving throw, damage immunity/weakness (`rule.damage/*`), power-roll/edge/bane/tier (`rule.dice/*`), skills (`skill.<group>/<item>`), `rule.combat/signature-ability`.
   - `docs/linking-guide.md` disambiguation rules + the monster disambiguation table in `monsters-linking-reference.md`.
3. **Link format:** `[Display Text](scc:CODE)`. Preserve case/possessive/plural in display text (`[minions](scc:…/minion)`, `[Brute](scc:…/role/brute)`). Bare `scc:` is canonical here (matches the existing corpus; explicit `scc.v1:` restamp is deferred — FOLLOWUPS #4).
4. **Link ALL game-mechanic instances** of a term (the pipeline handles render-time density). Do **not** link: a term in its own section heading or own defining body (self-ref); text inside `<!-- … -->` annotations; ordinary-English uses.
5. **Statblock rules** (Phase 1 made the labeled header safe — but still don't link it):
   - **DO link** trait/ability **effect & tier prose**: conditions ("the target is [slowed]"), movement ("[shift] 1 square", "can [fly]"), forced movement ("[push] 2", "[pull] 1", "[slide]"), combat terms ("[free strike]", "[opportunity attack]", "as a [maneuver]", "[Heal]", "[Defend]"), characteristics in prose, damage immunity/weakness phrasing, and **glossary terms** (Malice in "spend 3 [Malice]", "this [minion]", "[squad]", "with [Captain]").
   - **DO NOT link** the dice/power-roll **header line** (`**Power Roll + N:**` — structured-block label) or the **stat-grid label cells** (Size/Speed/Stamina/Stability/Free Strike/Immunity/Movement/With Captain/Weakness + the M/A/R/I/P row). A Movement-cell *value* that is a real movement type (Fly/Burrow/Teleport) MAY be linked (optional).
   - **DO NOT** over-link the keyword cell — prefer linking keywords in the Keywords definitions + group lore prose, not in every stat grid.
6. Mark uncertain cases `<!-- REVIEW: is this a game reference? -->[term](scc:…)<!-- /REVIEW -->`.
7. Run the incremental validation (below) and commit.

**Incremental validation (run after each pass):**
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all' > /tmp/gen.log 2>&1
echo "exit=$?"   # MUST be 0
grep "WARN: unresolved scc link" /tmp/gen.log | sed -E 's/.*link "([^"]+)".*/\1/' | sort -u > /tmp/warn-now.txt
comm -13 /tmp/scc-warn-baseline-codes.txt /tmp/warn-now.txt   # MUST print nothing (no NEW unresolved code)
grep -oE "mcdm\.monsters\.v1/[a-z0-9./-]+" steel-etl/classification.json | sort -u | wc -l   # must equal the post-Phase-0 total
```
**Statblock footgun spot-check (run after any pass that contains statblocks)** — confirm linked statblocks still extract roll/tiers/stats:
```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
find data/data-bestiary -path "*<group>*statblock*" -name "*.json" | head -1 | xargs -I{} sh -c 'cat {}' | grep -iE "\"roll\"|tier1|\"stamina\"|\"speed\"" | head
```
(Substitute a `<group>` touched by the pass, e.g. `angulotls`.) Expected: non-empty `roll`/`tier1`/`stamina`/`speed` — proving link-wrapping did not silently empty any field.

---

### Task 2.1: Pass A — Monster Basics chapter

**Section:** `# Monster Basics` (~9) through just before `# Monsters` (~1359).

- [ ] **Step 1:** Read the chapter. **Highest glossary density in the book** — this is where every glossary term is defined and cross-referenced. Link: the new `rule.monster/*`, `rule.role/*`, `rule.organization/*`, `rule.keyword/*` codes at their **cross-references** (not inside each term's own defining section — self-ref); cross-book Heroes terms throughout (Victories, Stamina, conditions, movement, combat actions, characteristics, saving throw, surge, damage immunity/weakness, power-roll terms, skills). The encounter-building sub-sections reference roles/organizations heavily — link those.
- [ ] **Step 2:** Apply links per the per-pass procedure. Skip self-references (the Malice section body, each role/organization/keyword's own definition paragraph).
- [ ] **Step 3:** Run incremental validation. Expected: exit 0, no new WARN, code count unchanged.
- [ ] **Step 4: Commit**
```bash
git add "input/monsters/Draw Steel Monsters.md"
git commit -m "link(monsters): monster basics chapter"
```

### Task 2.2: Pass B — Monsters intro + groups Ajax…Bugbears

**Section:** `# Monsters` (~1359) through just before `## Chimera` (~3768). Groups: Ajax the Invincible, Angulotls, Animals, Arixx, Ashen Hoarder, Basilisks, Bredbeddle, Bugbears (+ the `# Monsters` intro prose).

- [ ] **Step 1:** Read the section. Each group = lore prose + malice featureblock + `On <group>` aside + statblocks. Link group lore (conditions/movement/keywords/glossary), featureblock effect prose, and statblock trait/ability effect & tier prose per the statblock rules. Skip the stat-grid labels and power-roll headers.
- [ ] **Step 2:** Apply links.
- [ ] **Step 3:** Run incremental validation + statblock footgun spot-check (e.g. `angulotls`). Expected: 0 new WARN; non-empty roll/tier/stat fields.
- [ ] **Step 4: Commit**
```bash
git add "input/monsters/Draw Steel Monsters.md"
git commit -m "link(monsters): groups ajax through bugbears"
```

### Task 2.3: Pass C — Chimera + Demons

**Section:** `## Chimera` (~3768) through just before `## Devils` (~5501). (Demons is large and echelon-subcategorized.)

- [ ] **Step 1–4:** Apply the per-pass procedure (statblock rules apply). Footgun spot-check on `demons`. Commit:
```bash
git commit -am "link(monsters): chimera and demons"
```

### Task 2.4: Pass D — Devils, Draconians, Dragons

**Section:** `## Devils` (~5501) through just before `## Dwarves` (~7479).

- [ ] **Step 1–4:** Per-pass procedure; footgun spot-check on `dragons`. Commit:
```bash
git commit -am "link(monsters): devils, draconians, dragons"
```

### Task 2.5: Pass E — Dwarves, Elementals, Elves (High/Shadow/Wode)

**Section:** `## Dwarves` (~7479) through just before `## Fossil Cryptic` (~10143).

- [ ] **Step 1–4:** Per-pass procedure; footgun spot-check on `elves-high` (or the actual group dir). Commit:
```bash
git commit -am "link(monsters): dwarves, elementals, elves"
```

### Task 2.6: Pass F — Fossil Cryptic…Hobgoblins

**Section:** `## Fossil Cryptic` (~10143) through just before `## Humans` (~13186). Groups: Fossil Cryptic, Giants, Gnolls, Goblins, Griffons, Hag, Hobgoblins.

- [ ] **Step 1–4:** Per-pass procedure; footgun spot-check on `goblins`. Commit:
```bash
git commit -am "link(monsters): fossil cryptic through hobgoblins"
```

### Task 2.7: Pass G — Humans…Olothec

**Section:** `## Humans` (~13186) through just before `## Orcs` (~16130). Groups: Humans, Kingfissure Worm, Kobolds, Lightbenders, Lizardfolk, Manticore, Medusa, Minotaurs, Ogres, Olothec.

- [ ] **Step 1–4:** Per-pass procedure; footgun spot-check on `humans`. Commit:
```bash
git commit -am "link(monsters): humans through olothec"
```

### Task 2.8: Pass H — Orcs, Radenwights, Rivals

**Section:** `## Orcs` (~16130) through just before `## Shambling Mound` (~18650). (Rivals ~17237 is large and echelon-subcategorized — NPC adversaries referencing many mechanics.)

- [ ] **Step 1–4:** Per-pass procedure; footgun spot-check on `rivals` and `orcs`. Commit:
```bash
git commit -am "link(monsters): orcs, radenwights, rivals"
```

### Task 2.9: Pass I — Shambling Mound, Time Raiders, Trolls, Undead

**Section:** `## Shambling Mound` (~18650) through just before `## Count Rhodar Von Glauer` (~21164). (Undead ~19568 is large/echelon-subcategorized.)

- [ ] **Step 1–4:** Per-pass procedure; footgun spot-check on `undead`. Commit:
```bash
git commit -am "link(monsters): shambling mound, time raiders, trolls, undead"
```

### Task 2.10: Pass J — Named solos: Count Rhodar…War Dogs

**Section:** `## Count Rhodar Von Glauer` (~21164) through just before `## Werewolf` (~24828). Groups: Count Rhodar, Lich, Valok, Voiceless Talkers, Lord Syuul, War Dogs. (War Dogs ~22651 is large.)

- [ ] **Step 1–4:** Per-pass procedure; footgun spot-check on `war-dogs` (or the actual dir). Commit:
```bash
git commit -am "link(monsters): named solos through war dogs"
```

### Task 2.11: Pass K — Werewolf, Wyverns, Xorannox (end of Monsters chapter)

**Section:** `## Werewolf` (~24828) through just before `# Dynamic Terrain` (~25493).

- [ ] **Step 1–4:** Per-pass procedure; footgun spot-check on `wyverns`. Commit:
```bash
git commit -am "link(monsters): werewolf, wyverns, xorannox"
```

### Task 2.12: Pass L — Dynamic Terrain

**Section:** `# Dynamic Terrain` (~25493) through just before `# Retainers` (~27108).

- [ ] **Step 1:** Read the chapter. Dynamic-terrain entries reference conditions, movement, forced movement, damage, and combat terms heavily in their effect prose, plus the new glossary (Malice). Link effect prose; the terrain headings/own bodies are self-ref.
- [ ] **Step 2–3:** Apply links; run incremental validation (terrain has labeled power-roll blocks in places — footgun spot-check a `dynamic-terrain` JSON if any tiers are present).
- [ ] **Step 4: Commit**
```bash
git commit -am "link(monsters): dynamic terrain"
```

### Task 2.13: Pass M — Retainers

**Section:** `# Retainers` (~27108) through end of file (~29062). Retainer statblocks + advancement (H8 folded into the body as bold labels — `demoteOverflowHeadings`).

- [ ] **Step 1:** Read the chapter. Retainers are NPC allies referencing class features, conditions, movement, combat terms, and the glossary. Link statblock trait/ability effect prose and the folded advancement-ability prose. Skip stat-grid labels + power-roll headers.
- [ ] **Step 2–3:** Apply links; run incremental validation + footgun spot-check:
```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
find data/data-bestiary -path "*retainer*" -name "*.json" | head -1 | xargs -I{} sh -c 'cat {}' | grep -iE "\"roll\"|tier1|\"stamina\"" | head
```
Expected: 0 new WARN; non-empty roll/tier/stat fields.
- [ ] **Step 4: Commit**
```bash
git commit -am "link(monsters): retainers"
```

---

## Phase 3 — Validation, REVIEW resolution, regression

### Task 3.1: Resolve all REVIEW markers

**Files:** Modify `input/monsters/Draw Steel Monsters.md`

- [ ] **Step 1: Find every flagged case**
```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
grep -n "<!-- REVIEW:" "input/monsters/Draw Steel Monsters.md"
```
- [ ] **Step 2:** For each, decide keep-link (remove the `<!-- REVIEW: -->…<!-- /REVIEW -->` wrapper, keep the link) or drop-link (remove wrapper + link, restore plain text), applying the disambiguation key test from `linking-guide.md`.
- [ ] **Step 3: Confirm none remain**
```bash
grep -c "<!-- REVIEW:" "input/monsters/Draw Steel Monsters.md"   # expect 0
```
- [ ] **Step 4: Commit**
```bash
git commit -am "link(monsters): resolve REVIEW-flagged disambiguation cases"
```

### Task 3.2: Full clean build + link/footgun/stability verification

**Files:** none (verification).

- [ ] **Step 1: Full build; gate vs. baseline (no new WARN)**
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all' > /tmp/gen.log 2>&1
echo "exit=$?"   # MUST be 0
grep "WARN: unresolved scc link" /tmp/gen.log | sed -E 's/.*link "([^"]+)".*/\1/' | sort -u > /tmp/warn-now.txt
comm -13 /tmp/scc-warn-baseline-codes.txt /tmp/warn-now.txt   # MUST print nothing
```

- [ ] **Step 2: Code count stable; link total reported**
```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
grep -oE "mcdm\.monsters\.v1/[a-z0-9./-]+" classification.json | sort -u | wc -l   # equals post-Phase-0 total
grep -c "](scc:" "input/monsters/Draw Steel Monsters.md"   # the new link total — record for the docs note
```

- [ ] **Step 3: SCC stability check (no frozen codes changed by the glossary mint or linking)**
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate --scc-stable --config pipeline.yaml' 2>&1 | tail -5
```
Expected: stable / no violations. (Note: the Phase-0 glossary codes are *new* additions, not changes to existing codes — additions are allowed; only mutation of an existing frozen code is a violation.)

- [ ] **Step 4: Statblock data-integrity spot-check across several groups**
```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
for g in angulotls goblins demons dragons; do
  echo "== $g =="
  find data/data-bestiary -path "*$g*statblock*" -name "*.json" | head -1 | xargs -I{} sh -c 'cat {}' | grep -iE "\"roll\"|tier1|\"stamina\"|\"speed\"" | head -4
done
```
Expected: each group shows non-empty `roll`/`tier1`/`stamina`/`speed`.

- [ ] **Step 5: Render spot-check — no unresolved `scc:` reaches the site**
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml' 2>&1 | tail -3
grep -rln "scc:" v2/docs/Browse 2>/dev/null | head   # expect EMPTY
```
Expected: site builds; no literal `scc:` strings in the Browse output (all resolved to relative links).

- [ ] **Step 6: Full Go test suite**
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go test -race ./...'
```
Expected: PASS.

---

## Phase 4 — Documentation sync

### Task 4.1: Update linking docs

**Files:**
- Modify: `steel-etl/docs/linking-guide.md`
- Modify: `steel-etl/docs/linking-reference.md`

- [ ] **Step 1: `linking-guide.md`** — add a dated `> **2026-06-12 — Monsters book linked (done).**` block at the top (with the other dated notes): full internal + cross-book Heroes sweep; the new `rule.{monster,role,organization,keyword}` glossary minted from Monster Basics (point to `docs/monster-rule-mapping.md`); `sbPowerRollRe` hardened so the labeled power-roll form is link-safe; the final link total (Task 3.2 Step 2); validated with `--all`. **Codify the rule:** never link the `**Power Roll + N:**` statblock header (structured-block label). Add a Monsters progress matrix (one row per pass A–M, marked done) or a short section noting the 13 passes complete.
- [ ] **Step 2: `linking-reference.md`** — add a short `## Monsters book (mcdm.monsters.v1)` section pointing to `docs/monsters-linking-reference.md` for the book's own terms (incl. the new glossary), and noting Heroes terms here apply cross-book.
- [ ] **Step 3: Commit**
```bash
cd /home/scott/code/steelCompendium/workspace/steel-etl
git add docs/linking-guide.md docs/linking-reference.md
git commit -m "docs(linking): record monsters book link sweep + rule-glossary (done)"
```

### Task 4.2: Update CLAUDE.md files + scc-log

**Files:**
- Modify: `steel-etl/CLAUDE.md`
- Modify: `../CLAUDE.md` (workspace)
- Modify: `../docs/scc-log.md`

- [ ] **Step 1: `steel-etl/CLAUDE.md`** — in the Statblocks section, note `sbPowerRollRe` is now hardened (the "spared because Monsters isn't link-swept" caveat is resolved) and Monsters is fully link-swept. In the grouped-types note, add the new `rule.{monster,role,organization,keyword}` groups (minted from the Monsters book).
- [ ] **Step 2: workspace `../CLAUDE.md`** — SCC section: bump the registry total (591 → new Monsters total incl. glossary; and the overall ~2,956 → new total), note Monsters is now fully link-swept, and update the "Linking" bullet (monsters was "not link-swept (FOLLOWUPS #5)" → now done).
- [ ] **Step 3: `../docs/scc-log.md`** — append a dated `## 2026-06-12` entry: new monster rule-glossary groups (`rule.monster/role/organization/keyword`), code count delta, Monsters book link sweep done; link to this plan.
- [ ] **Step 4: Commit**
```bash
cd /home/scott/code/steelCompendium/workspace
git add steel-etl/CLAUDE.md CLAUDE.md docs/scc-log.md
git commit -m "docs: monsters link sweep + monster rule-glossary; update SCC state"
```

### Task 4.3: FOLLOWUPS + memory

**Files:**
- Modify: `../FOLLOWUPS.md`
- Memory: `/home/scott/.claude/projects/-home-scott-code-steelCompendium-workspace/memory/`

- [ ] **Step 1: `FOLLOWUPS.md` #5** — mark direction (1) **done** (Monsters source link-swept; glossary minted; parser fully hardened). If keeping direction (2) (links *into* monster pages from other books) as future work, restate it as a new follow-up item; otherwise note it as the remaining open half. Grep the live docs for `FOLLOWUPS #5` references and fix any that change meaning (per the workspace CLAUDE.md renumbering rule).
- [ ] **Step 2: Memory** — add/update a memory recording: Monsters book fully link-swept 2026-06-12; new `rule.{monster,role,organization,keyword}` glossary groups; `sbPowerRollRe` hardened; link with the `--all` validation gate. Add the one-line pointer to `MEMORY.md`. Link `[[comprehensive-linking-density]]` and `[[rule-scc-type]]`.
- [ ] **Step 3: Commit**
```bash
cd /home/scott/code/steelCompendium/workspace
git add FOLLOWUPS.md
git commit -m "docs(followups): monsters link sweep done (direction 1)"
```

---

## Self-Review notes

- **Spec coverage:** glossary minting (Phase 0, user's chosen approach — Task 0.2 annotates roles/organizations/keywords/core concepts, each with its own heading so one-heading-one-code holds); cross-book + internal link sweep of the whole 29k-line source (Tasks 2.1–2.13 cover Monster Basics → Retainers in 13 batches aligned to H2 boundaries); statblock safety (Phase 1 hardens the one remaining `sbPowerRollRe`; the others were hardened for Summoner; per-pass footgun spot-checks); validation (Phase 3, no-new-WARN gate vs. the 104-warning beastheart baseline + code-count guard + render spot-check); docs (Phase 4 across linking docs, both CLAUDE.md, scc-log, FOLLOWUPS, memory). Direction (2) explicitly out of scope per the user. ✓
- **Type/name consistency:** `linkDisplay`, `sbPowerRollRe`, `sbDiceRe`, `sbTierRe`, `cellRe`, `ParseStatblockFeatures` named as in `statblock_parse.go`; new SCC groups `rule.{monster,role,organization,keyword}` used consistently across Phase 0/4; book prefix `mcdm.monsters.v1`; output dir `data/data-bestiary`; paths use the real `/home/scott/code/steelCompendium/workspace`. ✓
- **Placeholder scan:** the reference/mapping skeletons explicitly say "fill every row, no `…` in the committed file"; the per-pass tasks delegate per-instance judgment to the documented procedure + reference tables (inherent to a manual link sweep — same shape as the completed Summoner plan), not vague TODOs. ✓
- **Risk:** (1) glossary annotations fragmenting the Monster Basics chapter render — verified up front (Task 0.2 Step 5) before any link work. (2) Minting a `rule.monster/<x>` that duplicates an existing Heroes target — guarded by the explicit "check linking-reference.md first" rule + pre-resolved exclusion list. (3) Scale (29k lines, 591 statblocks) — mitigated by 13 committed, independently-validated batches.
```
