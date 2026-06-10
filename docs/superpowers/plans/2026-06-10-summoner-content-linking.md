# Summoner Content Linking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `scc:` cross-reference links throughout the Summoner book source (`input/summoner/Draw Steel Summoner.md`) — both *internal* links (to the book's own 221 SCC codes) and *cross-book* links to the full Heroes reference — and harden the shared statblock parser so link-wrapping cannot break statblock data extraction.

**Architecture:** The Summoner source is fully annotated (221 codes already mint) but has **zero** links today. Linking is a manual, per-instance editing pass on one markdown file, governed by the existing disambiguation rules in `docs/linking-guide.md`. Two reference tables drive it: the existing `docs/linking-reference.md` (Heroes, 582 terms) for cross-book links, and a new `docs/summoner-linking-reference.md` (Summoner's own terms) built in Phase 0. Before any statblock is touched, Phase 1 hardens the Monsters/Summoner statblock parser regexes (`statblock_parse.go`) to tolerate `[term](scc:…)` wrapping — CLAUDE.md warns this parser "will hit the same wall when it is link-swept." Links are added section-by-section, validated with `gen` (0 WARN) plus JSON/card spot-checks, then docs are synced.

**Tech Stack:** Go (steel-etl pipeline + statblock parser, `go test`), annotated Markdown, devbox toolchain. All Go/just commands MUST be prefixed with `devbox run --` (Go is not on PATH).

---

## Background: what was learned during planning

- **Source:** `input/summoner/Draw Steel Summoner.md`, 4,435 lines, **0** existing `scc:` links.
- **Own codes (221):** `class/summoner` (1); `feature.summoner.level-N/*` (90); `feature.ability.summoner.level-N/*` (26); statblocks — `minion.<portfolio>.statblock/*`, `fixture.<portfolio>.statblock/*`, `champion.<portfolio>.statblock/*`, `retainer.summoner.statblock/*`, `rival.summoner.<echelon>.statblock/*`, `monster.statblock/*` (80 total); `treasure.*` (9); `title/*` (6); `chapter/*` (5). Portfolios = demon / elemental / fey / undead.
- **Cross-book targets (Heroes):** the source leans heavily on Heroes terms — characteristics (Reason, Might, Agility, Intuition, Presence), Stamina, Potency, Recoveries, Victories, named skills (Magic, Strategy, Eavesdrop, Monsters) and skill groups (intrigue, lore), conditions (slowed, prone, grabbed, winded, frightened…), movement (fly, hover, teleport, shift, forced movement / push / pull / slide), combat actions/maneuvers (Heal, Defend, free strike, opportunity attack, main action, maneuver, saving throw, hide), heroic resource / surges, and rules-glossary terms (power roll, edge, bane, tier, potency). User decision: **link the full Heroes reference**, including the dense rules-glossary sweep.
- **Statblock footgun (Phase 1):** the parser keys off literal text. Affected regexes in `internal/content/statblock_parse.go`:
  - `sbDiceRe = ^(.*?)\s+(\d+d\d+\s*\+\s*\S.*?)$` — splits the dice-in-title form `🏹 **Mind Twist 2d10 + R (Signature Ability)**`. Linking the name or the `R` breaks it.
  - `sbTierRe = ^-\s*\*\*(≤?\d+(?:-\d+)?\+?):\*\*\s*(.*)$` — bold tier labels.
  - `sbBareTierRe = ^\d` — bare digit-led tier lines, e.g. `4 damage; P < WEAK twisted (save ends)`.
  - `sbPowerRollRe = \*\*(Power Roll[^*]*)\*\*` — labeled form (Summoner mostly uses the dice-in-title form, but harden anyway).
  - `cellRe = \*\*(.*?)\*\*\s*<br\s*/?>\s*([A-Za-z][A-Za-z ]*)` — stat-grid `**VALUE**<br>Label` cells.
  - The matching ability parser (`internal/content/ability.go`) already hardened `powerRollHeaderRe` to tolerate `[Power Roll](…)`; its `effectRe`/`triggerRe`/`spendRe`/`tierRe` anchor on the *label* before the colon (Effect/Trigger/Spend), which is never linked, so ability-body effect VALUES are safe to link.
- **Keywords are NOT linkable:** Abyssal, Demon, Undead, Fey, Elemental creature keywords have no SCC code. Leave them plain. Circle/Portfolio names (Circle of Blight/Graves/Spring/Storms, "your portfolio") link to their feature code (`feature.summoner.level-1/summoner-circle` or `…/portfolio`) only when referenced as the game feature, not in flavor.

---

## File Structure

| File | Responsibility | Phase |
|------|----------------|-------|
| `docs/summoner-linking-reference.md` | **New.** Canonical list of the Summoner book's own linkable terms (display name, variants, SCC code) + disambiguation notes for common-word ability/feature names. | 0 |
| `internal/content/statblock_parse.go` | **Modify.** Harden `sbDiceRe`, `sbTierRe`, `sbPowerRollRe`, `cellRe` to tolerate `[term](scc:…)` and capture link-wrapped values verbatim. | 1 |
| `internal/content/statblock_parse_test.go` | **Modify.** Table-driven tests for each hardened regex with link-wrapped inputs. | 1 |
| `input/summoner/Draw Steel Summoner.md` | **Modify.** The actual link sweep, section by section. | 2 |
| `docs/linking-guide.md` | **Modify.** Add a Summoner progress section + the dated note. | 4 |
| `docs/linking-reference.md` | **Modify.** Add a short "Summoner book" pointer section referencing the new file. | 4 |
| `steel-etl/CLAUDE.md` | **Modify.** Update the SCC paragraph: Summoner is link-swept; statblock parser hardened. | 4 |
| `../FOLLOWUPS.md` / `CLAUDE.md` (workspace) | **Modify.** Note Summoner statblocks now swept (Monsters still pending #10); update SCC link totals. | 4 |

---

## Phase 0 — Baseline, references, cross-book spike

### Task 0.1: Establish a clean baseline

**Files:** none (read-only verification).

- [ ] **Step 1: Confirm zero links and a clean full build**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
grep -c "scc:" "input/summoner/Draw Steel Summoner.md"   # expect 0
devbox run -- go run ./cmd/steel-etl gen --config pipeline.yaml --all 2>&1 | grep -c WARN
```
Expected: first command prints `0`; second prints `0` (clean baseline — no pre-existing warnings).

- [ ] **Step 2: Record the summoner code count for later sanity checks**

Run:
```bash
grep -oE "mcdm\.summoner\.v1/[a-z0-9./-]+" classification.json | sort -u | wc -l
```
Expected: `221`. (Linking must NOT change this number — links never mint codes. If it changes, a link typo created a phantom target; investigate before proceeding.)

### Task 0.2: Build the Summoner linking reference

**Files:**
- Create: `steel-etl/docs/summoner-linking-reference.md`

- [ ] **Step 1: Extract the Summoner's own codes with display names**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
grep -oE "mcdm\.summoner\.v1/[a-z0-9./-]+" classification.json | sort -u > /tmp/summoner-codes.txt
wc -l /tmp/summoner-codes.txt   # expect 221
```

- [ ] **Step 2: Author `docs/summoner-linking-reference.md`**

Create the file with this structure. Populate the tables from `/tmp/summoner-codes.txt`, deriving Display Name from each code's item slug (title-cased, hyphens→spaces) cross-checked against the source headings. Group by type. Use this exact header and section skeleton (fill every row — no `…` placeholders in the committed file):

```markdown
# Summoner Linking Reference Table

Linkable terms for the **Summoner book** (`mcdm.summoner.v1`). For cross-book
links to Heroes terms (characteristics, conditions, skills, movement, combat
actions, rules-glossary) use `docs/linking-reference.md`. See `linking-guide.md`
for rules. Total Summoner terms: <N>.

## Class (1 term)

| Display Name | Variants | SCC Code |
|-------------|----------|----------|
| Summoner | summoners, summoner's | `mcdm.summoner.v1/class/summoner` |

## Features — by level (90 terms)

| Display Name | Variants | SCC Code |
|-------------|----------|----------|
| Minions | minion, minions | `mcdm.summoner.v1/feature.summoner.level-1/minions` |
| Essence | essence | `mcdm.summoner.v1/feature.summoner.level-1/essence` |
| Portfolio | portfolio, portfolios | `mcdm.summoner.v1/feature.summoner.level-1/portfolio` |
| Summoner Circle | circle, circles | `mcdm.summoner.v1/feature.summoner.level-1/summoner-circle` |
| Formation | formation, formations | `mcdm.summoner.v1/feature.summoner.level-1/formation` |
| Quick Command | quick command | `mcdm.summoner.v1/feature.summoner.level-1/quick-command` |
<!-- …one row per feature.summoner.level-N/* code… -->

## Abilities (26 terms)

| Display Name | Variants | SCC Code |
|-------------|----------|----------|
| Summoner Strike | summoner strike | `mcdm.summoner.v1/feature.ability.summoner.level-1/summoner-strike` |
| Strike for Me | strike for me | `mcdm.summoner.v1/feature.ability.summoner.level-1/strike-for-me` |
| Call Forth | call forth | `mcdm.summoner.v1/feature.ability.summoner.level-1/call-forth` |
<!-- …one row per feature.ability.summoner.level-N/* code… -->

## Statblocks — Minions / Fixtures / Champions / Rivals / Retainers (80 terms)

| Display Name | Portfolio | SCC Code |
|-------------|-----------|----------|
| Ensnarer | Demon (signature minion) | `mcdm.summoner.v1/minion.demon.statblock/ensnarer` |
<!-- …one row per *.statblock/* code… -->

## Treasures (9 terms)

| Display Name | Variants | SCC Code |
|-------------|----------|----------|
<!-- …one row per treasure.* code… -->

## Titles (6 terms)

| Display Name | Variants | SCC Code |
|-------------|----------|----------|
<!-- …one row per title/* code… -->

## Chapters (5 terms)

| Display Name | Variants | SCC Code |
|-------------|----------|----------|
<!-- …one row per chapter/* code… -->

## Disambiguation — common-word ability/feature names (REQUIRED)

Many Summoner feature/ability names are common English words. **Link only the
game-mechanic reference, never ordinary prose.** Per-instance judgment required:

| Term | Link (game mechanic) | Don't link (ordinary) |
|------|----------------------|------------------------|
| Essence | "spend 1 essence", "your essence pool", the Essence feature | "the essence of creation" (flavor lore) |
| Minions / Minion | "summon minions", "your minions act" | "a minion of the dark lord" (flavor) |
| Shield | the Shield ability | "raise your shield" (mundane) |
| Halt | the Halt ability | "halt the advance" (verb) |
| Rise | the Rise circle feature | "rise to the occasion" (verb) |
| Formation | the Formation feature / a named Formation | "in formation" (generic) |
| Portfolio | the Portfolio feature | — (almost always the feature) |
| Shield / Rise / Halt / Not Yet / Focus Fire | the named ability | the literal verb/phrase |

**Self-reference:** never link a term inside its own defining section heading or
body (e.g. the Essence feature page does not link the word "essence" to itself).
```

- [ ] **Step 3: Verify every code in the file exists in the registry**

Run (extracts each `scc:`-style code from the new doc's code column and checks it against the registry):
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
grep -oE "mcdm\.summoner\.v1/[a-z0-9./-]+" docs/summoner-linking-reference.md | sort -u > /tmp/ref-codes.txt
comm -23 /tmp/ref-codes.txt /tmp/summoner-codes.txt
```
Expected: **no output** (every code in the reference is a real registry code). Any line printed is a typo in the reference — fix it.

- [ ] **Step 4: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add docs/summoner-linking-reference.md
git commit -m "docs(summoner): add linking reference table for summoner book terms"
```

### Task 0.3: Cross-book resolution spike

**Files:**
- Modify (temporarily): `input/summoner/Draw Steel Summoner.md`

This confirms a Heroes-book link resolves correctly from within the Summoner source before committing to ~hundreds of them.

- [ ] **Step 1: Add one cross-book link to a Summoner prose line**

In the Basics section, line ~348, wrap the first "Reason" reference:
```markdown
**Starting Characteristics:** You start with a [Reason](scc:mcdm.heroes.v1/rule.character/reason) of 2, and you can choose one of the following arrays for your other characteristics scores:
```

- [ ] **Step 2: Build and confirm it resolves with no warning**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go run ./cmd/steel-etl gen --config pipeline.yaml --all 2>&1 | grep -iE "WARN|reason" | head
```
Expected: **no WARN** about an unresolved `rule.character/reason`. (If a WARN appears, the cross-book registry is not shared in the same build — STOP and inspect `selectBookConfigs` / registry merge; cross-book linking depends on this working.)

- [ ] **Step 3: Confirm the link resolved to a real relative path in the linked output**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
grep -rn "reason" data/data-summoner/en/md-linked 2>/dev/null | grep -i "rule.character\|reason.md\|](\.\./" | head
```
Expected: the Summoner basics page shows a resolved relative link to the Heroes `reason` page (a `](../…reason…)` path), **not** a literal `scc:` string.

- [ ] **Step 4: Confirm a Summoner-only `gen` also resolves the cross-book link**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go run ./cmd/steel-etl gen --config pipeline.yaml --book summoner 2>&1 | grep -c WARN
```
Expected: `0`. (A single-book `gen` merges into the existing registry, so Heroes codes remain available. If this prints non-zero, the cross-book link only resolves under `--all` — record that in the linking-guide note in Phase 4 so future editors always validate with `--all`.)

- [ ] **Step 5: Keep the spike link, commit**

The Reason link is a genuine, correct link — keep it.
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): cross-book resolution spike — link first Reason ref"
```

---

## Phase 1 — Harden the statblock parser for link-wrapping (TDD)

Goal: make `statblock_parse.go` extract the same data whether or not a value is wrapped in `[text](scc:…)`. Captured values keep the raw `[text](scc:…)` verbatim (matching the effect/distance convention; cards render it via `richInline`). **Do this before linking any statblock.**

### Task 1.1: Failing test — dice-in-title with link-wrapped characteristic

**Files:**
- Test: `internal/content/statblock_parse_test.go`

- [ ] **Step 1: Add a helper to strip links for matching, write the failing test**

Add this test (adjust the parse entrypoint name to match the file's existing tests — use the same statblock-parsing function the current tests call, e.g. `ParseStatblockFeatures` or `parseStatblockFeature`):

```go
func TestStatblock_DiceTitle_ToleratesLinkedCharacteristic(t *testing.T) {
	// Power-roll dice-in-title form with the characteristic wrapped in an scc link.
	block := "🏹 **Mind Twist 2d10 + [R](scc:mcdm.heroes.v1/rule.character/reason) (Signature Ability)**\n" +
		"\n" +
		"4 damage; P < WEAK twisted (save ends)\n" +
		"6 damage; P < AVERAGE twisted (save ends)\n" +
		"8 damage; P < STRONG twisted (save ends)\n"

	feats := ParseStatblockFeatures(block) // use the real entrypoint
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	if f.Name != "Mind Twist" {
		t.Errorf("name = %q, want %q (dice + linked characteristic must be stripped from the title)", f.Name, "Mind Twist")
	}
	if f.Roll == "" {
		t.Errorf("roll not extracted from linked title; want the 2d10 + R expression preserved")
	}
}
```

- [ ] **Step 2: Run it; confirm it fails**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go test ./internal/content/ -run TestStatblock_DiceTitle_ToleratesLinkedCharacteristic -v
```
Expected: FAIL — `sbDiceRe`'s `\d+d\d+\s*\+\s*\S` trailing capture swallows the `[R](scc:…)` into the roll/name boundary incorrectly, or the title isn't recognized.

### Task 1.2: Harden `sbDiceRe` + title cleaning

**Files:**
- Modify: `internal/content/statblock_parse.go:86`

- [ ] **Step 1: Make the dice tail tolerate a link-wrapped characteristic**

The dice expression's right operand may now be `[R](scc:…)` instead of bare `R`. Update `sbDiceRe` so the operand after `+` accepts either form, and ensure the name (everything before the dice) is captured cleanly:

```go
// sbDiceRe splits a "Name Nd10 + <characteristic>" title into clean name + dice.
// The characteristic may be link-wrapped: "+ [R](scc:…)".
sbDiceRe = regexp.MustCompile(`^(.*?)\s+(\d+d\d+\s*\+\s*(?:\[[^\]]+\]\([^)]*\)|\S).*?)$`)
```

If the parser separately slugifies/cleans the captured roll or name, ensure any `[text](scc:…)` in the roll is normalized to its display `text` for the stored `roll` (strip the link markup from the dice expression only — the roll is structured data, not prose). Add a small helper if one does not already exist:

```go
// linkDisplay returns the display text of a markdown link, or s unchanged.
var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)

func linkDisplay(s string) string {
	return mdLinkRe.ReplaceAllString(s, "$1")
}
```
Apply `linkDisplay` to the captured dice roll before storing it.

- [ ] **Step 2: Run the test; confirm it passes**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go test ./internal/content/ -run TestStatblock_DiceTitle_ToleratesLinkedCharacteristic -v
```
Expected: PASS.

### Task 1.3: Failing test — tier lines with link-wrapped values

**Files:**
- Test: `internal/content/statblock_parse_test.go`

- [ ] **Step 1: Write the failing test (both labeled and bare tier forms)**

```go
func TestStatblock_TierLines_ToleratesLinks(t *testing.T) {
	// Bare digit-led tier line with a linked condition in the value.
	bare := "4 damage; P < WEAK [slowed](scc:mcdm.heroes.v1/condition/slowed) (save ends)"
	// Labeled tier line with a linked condition.
	labeled := "- **≤11:** 2 damage; the target is [prone](scc:mcdm.heroes.v1/condition/prone)"

	if !sbBareTierRe.MatchString(bare) {
		t.Errorf("bare tier line no longer recognized after linking the value")
	}
	m := sbTierRe.FindStringSubmatch(labeled)
	if m == nil {
		t.Fatalf("labeled tier line not matched: %q", labeled)
	}
	if want := "2 damage; the target is [prone](scc:mcdm.heroes.v1/condition/prone)"; m[2] != want {
		t.Errorf("tier value = %q, want %q (link must be preserved verbatim)", m[2], want)
	}
}
```

- [ ] **Step 2: Run it**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go test ./internal/content/ -run TestStatblock_TierLines_ToleratesLinks -v
```
Expected: likely PASS for `sbBareTierRe` (anchored at line start, value untouched) and PASS for `sbTierRe` (value capture `(.*)` is greedy and link-agnostic). **If both pass, no code change is needed for tiers** — keep the test as a regression guard and skip Task 1.4. If either fails, proceed to Task 1.4.

### Task 1.4: Harden tier parsing (only if Task 1.3 failed)

**Files:**
- Modify: `internal/content/statblock_parse.go:80,87`

- [ ] **Step 1: Adjust the failing regex**

`sbTierRe`'s value group is `(.*)` (already link-safe). `sbBareTierRe` only checks `^\d` (already link-safe). If the *consumer* of these matches re-parses the value with another literal regex (e.g. to split damage vs. effect), make that tolerate `[x](scc:…)` the same way (accept `(?:\[[^\]]+\]\([^)]*\)|…)`). Apply the minimal change that makes Task 1.3's test pass.

- [ ] **Step 2: Run the test; confirm PASS**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go test ./internal/content/ -run TestStatblock_TierLines_ToleratesLinks -v
```
Expected: PASS.

### Task 1.5: Failing test — stat-grid cell with link-wrapped value

**Files:**
- Test: `internal/content/statblock_parse_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestStatblock_Cell_ToleratesLinkedValue(t *testing.T) {
	// Movement cell with the value linked to the Teleport movement rule.
	cell := "**[Teleport](scc:mcdm.heroes.v1/movement/teleport)**<br>Movement"
	m := cellRe.FindStringSubmatch(cell)
	if m == nil {
		t.Fatalf("linked stat-grid cell not matched: %q", cell)
	}
	if got := linkDisplay(m[1]); got != "Teleport" {
		t.Errorf("cell value = %q, want display %q", got, "Teleport")
	}
	if m[2] != "Movement" {
		t.Errorf("cell label = %q, want %q", m[2], "Movement")
	}
}
```

- [ ] **Step 2: Run it; confirm it fails**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go test ./internal/content/ -run TestStatblock_Cell_ToleratesLinkedValue -v
```
Expected: FAIL — `cellRe`'s value group `\*\*(.*?)\*\*` is non-greedy and a `]` is fine, but the inner `*`-free assumption holds (links contain no `*`), so it may actually match; if it matches, assert the display extraction. If the label group `([A-Za-z][A-Za-z ]*)` mis-binds, fix in Task 1.6.

### Task 1.6: Harden `cellRe` (only if Task 1.5 failed)

**Files:**
- Modify: `internal/content/statblock_parse.go:26`

- [ ] **Step 1: Adjust if needed**

`cellRe`'s value is `\*\*(.*?)\*\*` — a link `[Teleport](scc:…)` contains no `*`, so the existing regex captures the whole `[Teleport](scc:…)` as the value. The consumer must `linkDisplay()` the captured value before using it as a movement type / damage type. Apply `linkDisplay` at the cell-value consumption site. Only change `cellRe` itself if Task 1.5 proves the label group mis-binds.

- [ ] **Step 2: Run the test; confirm PASS**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go test ./internal/content/ -run TestStatblock_Cell_ToleratesLinkedValue -v
```
Expected: PASS.

### Task 1.7: Full regression + commit

**Files:** none (verification).

- [ ] **Step 1: Run the whole package test suite with race**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go test -race ./internal/content/...
```
Expected: PASS (all existing statblock tests still green — the changes are additive/tolerant).

- [ ] **Step 2: Run the full build to confirm nothing else broke**

Run:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go build ./... && devbox run -- go test ./...
```
Expected: build OK, all tests PASS.

- [ ] **Step 3: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add internal/content/statblock_parse.go internal/content/statblock_parse_test.go
git commit -m "fix(statblock): tolerate scc link-wrapping in dice title, tiers, and stat cells"
```

---

## Phase 2 — Link sweep, section by section

**Per-pass procedure (apply to every task below):**

1. Read the section's full text (use the heading anchors given; re-`grep -n` the anchor each time since prior edits shift line numbers — never trust stale line numbers).
2. For each candidate term, decide link vs. don't-link **per instance** using:
   - `docs/summoner-linking-reference.md` (internal terms) — for the book's own abilities/features/statblocks/treasures/titles.
   - `docs/linking-reference.md` (Heroes, all 582 terms) — for cross-book terms, **including** the rules-glossary (`rule.character/*` for Reason/Might/Agility/Intuition/Presence; `rule.dice/*` for power roll/edge/bane/tier; `rule.health/*` for winded/recovery; etc.).
   - `docs/linking-guide.md` disambiguation rules (conditions, skills, movement, negotiation, culture, rules-glossary — link only the game-mechanic use).
3. Link format: `[Display Text](scc:CODE)`. Preserve case/possessive/plural in display text (e.g. `[minions](scc:…/minions)`, `[Summoner's](scc:…/class/summoner)`).
4. **Link ALL game-mechanic instances** of a term (pipeline handles density). Do **not** link: a term in its own section heading or own defining body (self-ref), text inside `<!-- … -->` annotations, creature keywords (Abyssal/Demon/Undead/Fey/Elemental — no code), or ordinary-English uses.
5. **Statblock-specific rules:** Phase 1 made statblocks safe. Inside statblocks: **link the trait/ability effect prose** (e.g. "the target is [slowed]", "inflict [pull] 1", "can't be [hidden]", "uses a [maneuver]"). **Do NOT link the dice-in-title line** (the name is self-ref and `R` is part of the dice expression) and **do NOT link inside the stat-grid header keywords** (Abyssal/Demon etc.). You *may* link a stat-grid Movement value that is a real movement type (e.g. Teleport) — optional, low value.
6. Mark uncertain cases with `<!-- REVIEW: is this a game reference? -->[term](scc:…)<!-- /REVIEW -->`.
7. After the pass, run the incremental validation (below) and commit.

**Incremental validation (run after each pass):**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go run ./cmd/steel-etl gen --config pipeline.yaml --all 2>&1 | grep WARN
grep -oE "mcdm\.summoner\.v1/[a-z0-9./-]+" classification.json | sort -u | wc -l   # must stay 221
```
Expected: no WARN (every `scc:` resolved); code count unchanged at 221. A WARN naming an `mcdm.summoner.v1/…` or `mcdm.heroes.v1/…` code means a typo'd target — fix before committing.

### Task 2.1: Pass A — "The Summoner" lore chapter

**Files:** Modify `input/summoner/Draw Steel Summoner.md`

**Section:** `# The Summoner` (≈ line 8) through just before `# The Summoner Class` (≈ line 333). Includes the `## On Summoning` lore.

- [ ] **Step 1:** Read the section. This is mostly setting/lore prose — expect few links: occasional cross-book references (essence as the elementalist's resource, characteristics, conditions) and forward references to summoner features. Link conservatively (lore leans flavor).
- [ ] **Step 2:** Apply links per the per-pass procedure.
- [ ] **Step 3:** Run incremental validation (above). Expected: 0 WARN, 221 codes.
- [ ] **Step 4: Commit**
```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): the summoner lore chapter"
```

### Task 2.2: Pass B — Class basics + 1st-level features (non-statblock)

**Section:** `## Summoner` (≈ 337) through just before `### Portfolio` (≈ 774). Covers Basics (characteristics, Stamina, Potency, Recoveries, Skills, the advancement table), Draw Steel Master Classes, and the 1st-level features (Summoner Circle, Minions, Essence, Summoner Strike, Strike For Me, Call Forth, Minion Bridge, the circle features, Formation, Quick Command + their abilities).

- [ ] **Step 1:** Read the section. High link density here:
  - **Cross-book:** Reason/Might/Agility/Intuition/Presence (`rule.character/*`), Stamina/winded/Recoveries (`rule.health/*`), Potency (`rule.character/potency`), Victories, the named skills Magic & Strategy (`skill.lore/*`) + intrigue/lore **skill groups** (`skill.group/*`) + Quick-Build skills (Eavesdrop `skill.intrigue/eavesdrop`, Monsters `skill.lore/monsters`), conditions (slowed, prone, grabbed), movement (fly, hover, shift, forced movement / pull), combat (free strike, opportunity attack, main action, maneuver, Heal, Defend, saving throw), surges, power roll / edge / bane / tier (`rule.dice/*`).
  - **Internal:** cross-references between features ("see Portfolio", "see Minion Machinations", named abilities like Call Forth, Focus Fire, Halt, Not Yet, Shield) → link to their `feature.summoner.*` / `feature.ability.summoner.*` codes.
  - **Disambiguation:** "essence", "minions", "shield", "halt", "rise", "formation" appear constantly as both feature names and ordinary words — apply the `summoner-linking-reference.md` disambiguation table per instance. Skip self-references inside each feature's own body.
- [ ] **Step 2:** Apply links. The ability blocks (`#### Summoner Strike`, etc.) — link **effect/trigger VALUES** (safe per Phase-1 note on `ability.go`), not the `**Effect:**`/`**Trigger:**` labels.
- [ ] **Step 3:** Run incremental validation. Expected: 0 WARN, 221 codes.
- [ ] **Step 4: Commit**
```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): class basics and 1st-level features"
```

### Task 2.3: Pass C — Portfolio minion statblocks

**Section:** `### Portfolio` (≈ 774) through just before `### Summoner Abilities` (≈ 1356). Contains the demon/elemental/fey/undead signature + 3-Essence minion statblocks (25 statblocks).

- [ ] **Step 1:** Read each statblock. Apply the **statblock-specific rules** (procedure step 5): link the ⭐️ trait bodies and 🏹 ability **effect/tier prose** — conditions (slowed, prone, grabbed, frightened, weakened, restrained, bleeding, taunted), movement (shift, fly, hover, teleport, forced movement / push / pull / slide), combat terms (free strike, opportunity attack, maneuver, hide/hidden, main action), characteristics in prose. **Do not** link the dice-in-title line or the keyword header row.
- [ ] **Step 2:** Apply links across all 25 statblocks.
- [ ] **Step 3:** Run incremental validation. Then **footgun spot-check** — confirm a linked statblock still extracts power-roll/tiers/stats:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
# Find a linked minion's JSON and confirm roll/tiers/stat fields are populated:
find data/data-summoner -path "*minion.demon*" -name "*.json" | head -1 | xargs -I{} sh -c 'echo {}; cat {}' | grep -iE "\"roll\"|tier1|\"stamina\"|\"speed\"" | head
```
Expected: 0 WARN; the JSON shows non-empty `roll`/`tier*`/stat fields (Phase-1 hardening verified on real linked content).
- [ ] **Step 4: Commit**
```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): portfolio minion statblocks"
```

### Task 2.4: Pass D — Summoner Abilities + 2nd–5th level features (incl. fixtures & mid-tier minions)

**Section:** `### Summoner Abilities` (≈ 1356) through just before `### 6th-Level Features` (≈ 2490). Covers the 5-/7-Essence abilities, 2nd-level Perk, the four Portfolio **Fixtures** (statblocks), 5-Essence minion statblocks, Ward circle features, 3rd–5th level features.

- [ ] **Step 1:** Read the section. Mixed prose-abilities + statblocks. Apply both the prose rules (Pass B style) and statblock rules (Pass C style). Internal cross-references are dense here (abilities reference Essence costs, Minions, Formations, Call Forth, Portfolio).
- [ ] **Step 2:** Apply links.
- [ ] **Step 3:** Run incremental validation. Expected: 0 WARN, 221 codes.
- [ ] **Step 4: Commit**
```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): summoner abilities and 2nd-5th level features"
```

### Task 2.5: Pass E — 6th–10th level features (incl. high-tier minions & champions)

**Section:** `### 6th-Level Features` (≈ 2490) through just before `# Rewards` (≈ 3136). Covers 6th–10th level features and abilities, 7-Essence minion statblocks, Portfolio **Champions** (statblocks), and the 9-/11-Essence abilities.

- [ ] **Step 1:** Read the section. Same mixed prose + statblock treatment.
- [ ] **Step 2:** Apply links.
- [ ] **Step 3:** Run incremental validation. Expected: 0 WARN, 221 codes.
- [ ] **Step 4: Commit**
```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): 6th-10th level features and champion statblocks"
```

### Task 2.6: Pass F — Rewards (trinkets, leveled treasures, titles)

**Section:** `# Rewards` (≈ 3136) through just before `# Other Summoners` (≈ 3539). Trinkets, leveled implement treasures, and new titles.

- [ ] **Step 1:** Read the section. Link: internal treasure/title self-context only where cross-referenced (not the item's own heading); cross-book terms (echelon, conditions, characteristics, abilities granted, essence, minions). Treasures/titles often grant effects referencing combat/conditions — link those effect terms.
- [ ] **Step 2:** Apply links.
- [ ] **Step 3:** Run incremental validation. Expected: 0 WARN, 221 codes.
- [ ] **Step 4: Commit**
```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): rewards — trinkets, treasures, titles"
```

### Task 2.7: Pass G — Other Summoners (Retainer + Rival statblocks)

**Section:** `# Other Summoners` (≈ 3539) through just before `# Summoner Advice` (≈ 4189). The Retainer Summoner (4 statblocks) and Rival Summoner (18 echelon-versioned statblocks).

- [ ] **Step 1:** Read each statblock + surrounding prose. Apply statblock rules (Pass C). These are NPC summoners, so they reference summoner class features (Minions, Essence, Portfolio, Call Forth) — link those internal references in their prose, plus conditions/movement/combat in effect text.
- [ ] **Step 2:** Apply links across all 22 statblocks.
- [ ] **Step 3:** Run incremental validation + footgun spot-check on a rival statblock JSON:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
find data/data-summoner -path "*rival.summoner*" -name "*.json" | head -1 | xargs -I{} sh -c 'cat {}' | grep -iE "\"roll\"|tier1|\"stamina\"" | head
```
Expected: 0 WARN; non-empty roll/tier/stat fields.
- [ ] **Step 4: Commit**
```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): other summoners — retainer and rival statblocks"
```

### Task 2.8: Pass H — Summoner Advice

**Section:** `# Summoner Advice` (≈ 4189) through end of file (4435). For Players / For Directors guidance.

- [ ] **Step 1:** Read the section. Advice prose references many features by name (Minions, Essence, Portfolio, Standby Minions, converting minions into monsters) and cross-book concepts. Link game-mechanic references; this section is fairly reference-dense despite being prose.
- [ ] **Step 2:** Apply links.
- [ ] **Step 3:** Run incremental validation. Expected: 0 WARN, 221 codes.
- [ ] **Step 4: Commit**
```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): summoner advice (players & directors)"
```

---

## Phase 3 — Validation, REVIEW resolution, regression

### Task 3.1: Resolve all REVIEW markers

**Files:** Modify `input/summoner/Draw Steel Summoner.md`

- [ ] **Step 1: Find every flagged case**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
grep -n "<!-- REVIEW:" "input/summoner/Draw Steel Summoner.md"
```
- [ ] **Step 2:** For each, decide keep-link (remove the `<!-- REVIEW: -->…<!-- /REVIEW -->` wrapper, keep the link) or drop-link (remove wrapper + the link, restore plain text). Apply the disambiguation key test from `linking-guide.md`.
- [ ] **Step 3: Confirm none remain**
```bash
grep -c "<!-- REVIEW:" "input/summoner/Draw Steel Summoner.md"   # expect 0
```
- [ ] **Step 4: Commit**
```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "link(summoner): resolve REVIEW-flagged disambiguation cases"
```

### Task 3.2: Full clean build + link/footgun verification

**Files:** none (verification).

- [ ] **Step 1: Full build, zero warnings**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go run ./cmd/steel-etl gen --config pipeline.yaml --all 2>&1 | grep -c WARN
```
Expected: `0`.

- [ ] **Step 2: Code count unchanged; link count reported**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
grep -oE "mcdm\.summoner\.v1/[a-z0-9./-]+" classification.json | sort -u | wc -l   # expect 221
grep -oc "](scc:" "input/summoner/Draw Steel Summoner.md"   # the new total — record it for the docs note
```
Expected: 221 codes; a non-zero link total (record the number for Phase 4).

- [ ] **Step 3: SCC stability check (no frozen codes changed)**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go run ./cmd/steel-etl validate --scc-stable --config pipeline.yaml 2>&1 | tail -5
```
Expected: stable / no violations (linking never changes codes).

- [ ] **Step 4: Statblock data-integrity spot-check across all portfolios**

Confirm linked statblocks across each portfolio still produce populated roll/tier/stat data (the Phase-1 guarantee, verified on the final linked content):
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
for p in demon elemental fey undead; do
  echo "== $p =="
  find data/data-summoner -path "*minion.$p*" -name "*.json" | head -1 | xargs -I{} sh -c 'cat {}' | grep -iE "\"roll\"|tier1|\"stamina\"|\"speed\"" | head -4
done
```
Expected: each portfolio shows non-empty `roll`/`tier1`/`stamina`/`speed` — proving link-wrapping did not silently empty any field.

- [ ] **Step 5: Render spot-check — a linked ability card + a linked statblock page**

Build the site and eyeball one ability page and one statblock page for resolved links + intact power-roll panel:
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go run ./cmd/steel-etl site --config ../v2/site.yaml 2>&1 | tail -3
grep -rln "scc:" ../v2/docs/Browse 2>/dev/null | head   # expect EMPTY: no unresolved scc: should reach the site
```
Expected: site builds; **no** literal `scc:` strings left in the Browse output (all resolved to relative links).

- [ ] **Step 6: Full Go test suite**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
devbox run -- go test -race ./...
```
Expected: PASS.

---

## Phase 4 — Documentation sync

### Task 4.1: Update linking docs

**Files:**
- Modify: `steel-etl/docs/linking-guide.md`
- Modify: `steel-etl/docs/linking-reference.md`

- [ ] **Step 1: Add a dated Summoner note + progress section to `linking-guide.md`**

At the top (with the other dated notes), add a `> **2026-06-10 — Summoner book linked (done).**` block summarizing: full internal + full Heroes cross-book sweep; statblock parser hardened (Phase 1) so statblocks are now safely link-swept; the final link total (from Task 3.2 Step 2); cross-book links validated with `--all`. Add a "Summoner" row/section to the progress matrix (or a short separate matrix) marking each pass A–H done.

- [ ] **Step 2: Add a pointer section to `linking-reference.md`**

Add a short `## Summoner book (mcdm.summoner.v1)` section pointing to `docs/summoner-linking-reference.md` for the book's own terms, and noting Heroes terms in this file apply cross-book.

- [ ] **Step 3: Commit**
```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add docs/linking-guide.md docs/linking-reference.md
git commit -m "docs(linking): record summoner book link sweep (done)"
```

### Task 4.2: Update CLAUDE.md files

**Files:**
- Modify: `steel-etl/CLAUDE.md`
- Modify: `../CLAUDE.md` (workspace)

- [ ] **Step 1: steel-etl/CLAUDE.md** — in the Monsters/statblock section, update the footgun note: the statblock parser (`sbDiceRe`/`sbTierRe`/`cellRe`) is now hardened to tolerate `scc:` link-wrapping, and the **Summoner** statblocks are link-swept (Monsters still pending). Adjust the "spared only because not link-swept yet" sentence accordingly.

- [ ] **Step 2: workspace CLAUDE.md** — in the SCC registry paragraph, note the Summoner book is now fully link-swept (internal + cross-book to Heroes), with the new link total; reference this plan.

- [ ] **Step 3: Commit**
```bash
cd /home/vexa/code/steel_compendium/workspace
git add steel-etl/CLAUDE.md CLAUDE.md
git commit -m "docs: note summoner link sweep + statblock parser hardening"
```

### Task 4.3: FOLLOWUPS + memory

**Files:**
- Modify: `../FOLLOWUPS.md`
- Memory: `~/.claude/projects/-home-vexa-code-steel-compendium-workspace/memory/`

- [ ] **Step 1: FOLLOWUPS.md** — update #10 (bestiary in-prose SCC sweep): the Summoner statblocks are now swept, so the remaining gap is the **Monsters** book statblocks/pages (and the bestiary Browse pages). Note that the statblock parser is now hardened, removing the blocker for sweeping the Monsters book.

- [ ] **Step 2: Memory** — update `project_pdf_conversion_pipeline.md` or the summoner-related memory (and `MEMORY.md` index) to record: Summoner book fully link-swept 2026-06-10; statblock parser hardened for link-wrapping; cross-book links to Heroes validated under `gen --all`.

- [ ] **Step 3: Commit**
```bash
cd /home/vexa/code/steel_compendium/workspace
git add FOLLOWUPS.md
git commit -m "docs(followups): summoner statblocks swept; monsters sweep unblocked"
```

---

## Self-Review notes

- **Spec coverage:** internal links (Tasks 2.1–2.8 cover every section; reference built in 0.2), cross-book full-Heroes links (per-pass procedure step 2 + Pass B/C call out the rules-glossary explicitly), statblock safety (Phase 1 + per-pass footgun spot-checks), validation (Phase 3), docs (Phase 4). ✓
- **Type/name consistency:** `linkDisplay` helper defined once (Task 1.2) and reused (1.6); `sbDiceRe`/`sbTierRe`/`sbBareTierRe`/`cellRe`/`sbPowerRollRe` named exactly as in `statblock_parse.go`; SCC code shapes match the registry dump. ✓
- **Conditional tasks:** Tasks 1.4 and 1.6 are explicitly gated on their preceding test failing — if `sbTierRe`/`cellRe` already tolerate links (likely, since their value groups are link-agnostic), keep the tests as regression guards and skip the code change. This is intentional, not a placeholder.
- **Risk:** the only hard dependency is cross-book registry sharing (Task 0.3) — validated up front before the bulk sweep.
