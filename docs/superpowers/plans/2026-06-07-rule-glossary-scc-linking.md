# Rule/Glossary SCC Linking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every Draw Steel *rules* term (glossary entries + mechanics referenced throughout the heroes doc — flanking, cover, winded, edge, bane, surge, size, characteristics, etc.) clickable, by minting a new grouped `rule.<group>/<term>` SCC type anchored to the existing rules sections, then linking all instances.

**Architecture:** A new lightweight `RuleParser` (mirrors `GodParser`) emits `rule.<group>/<id>` codes from `<!-- @type: rule | @group: <g> | @id: <slug> -->` annotations placed on the **existing** headed rules sections (The Basics, Tests, Combat, Downtime, Rewards/Titles, Introduction). No glossary restructuring: links point at the full rules section (terms lacking their own heading map to the nearest headed parent, e.g. `[pushed](scc:.../rule.movement/forced-movement)`). Terms already covered by a typed code (conditions, movement, skills, classes, …) keep linking to those. Then a per-instance linking pass covers the glossary (every entry) and the whole document.

**Tech Stack:** Go (steel-etl pipeline parsers + SCC classifier), the annotated `input/heroes/Draw Steel Heroes.md`, the Python link-audit tooling in `steel-etl/scripts/`, MkDocs/`v2/site.yaml` for Browse publishing.

**Decisions locked (2026-06-07, with the user):**
- Namespace: **`rule.<group>/<term>`** (grouped hierarchy), e.g. `rule.combat/flanking`, `rule.character/might`, `rule.damage/untyped-damage`, `rule.dice/power-roll`.
- Link target: **the full rules section** (annotate the real heading; nearest-parent fallback or a small new heading where none exists).
- Scope: **everything this round** — the type + annotations + reference table + glossary + a full per-instance sweep across all chapters.

**Verification bar (every phase):** `gen` **0 WARN** (run via devbox — see below), `go test ./...` green, no malformed links (`grep` patterns in Phase 7), and `rule/*` pages publish + the Read-tab chapters still render.

**Devbox invocation (all Go/pipeline/python commands):**
```bash
devbox run -- bash -c 'cd steel-etl && <command>'
# e.g. devbox run -- bash -c 'cd steel-etl && go test ./internal/content/... ./internal/scc/...'
# gen 0-WARN check:
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml' 2>&1 | grep -iE "WARN|error" || echo "clean"
```

---

## Group Taxonomy (the `<group>` segment)

A fixed, small, predictable set. Every `rule` annotation MUST carry one of these `@group` values. (Movement terms are **not** in this list — they reuse the existing `movement/*` type; see "Reuse existing codes" below.)

| `@group` | Covers (examples) |
|----------|-------------------|
| `dice` | power-roll, natural-roll, natural-19-20, edge, bane, double-edge, double-bane, tier-outcome, tier-1, tier-2, tier-3, bonuses-and-penalties, automatic-tier-outcome |
| `character` | characteristic, might, agility, reason, intuition, presence, size, speed, stability, potency |
| `health` | stamina, temporary-stamina, recoveries, recovery-value, winded, dying, suffocating, falling |
| `resource` | heroic-resource, heroic-ability, hero-token, surge, victories, experience, renown, wealth |
| `combat` | combat-round, turn, main-action, maneuver, move-action, free-maneuver, triggered-action, free-triggered-action, no-action, opportunity-attack, flanking, cover, concealment, line-of-effect, surprised, side, objective, critical-hit, strike, opposed-power-roll, distance, line, cube, wall |
| `damage` | damage, damage-type, untyped-damage, rolled-damage, damage-immunity, damage-weakness |
| `test` | test, group-test, montage-test, reactive-test, test-difficulty, crafting-skills, exploration-skills, interpersonal-skills, intrigue-skills, lore-skills |
| `downtime` | downtime-project, crafting-project, research-project, project-goal, project-points, project-roll, project-source, project-event, respite, respite-activity, guide, item-prerequisite |
| `negotiation` | interest, patience, motivation, pitfall |
| `treasure` | leveled-treasure, trinket, consumable, enhancement, implement |
| `world` | orden, timescape, manifold, vasloria, god, saint, capital |
| `general` | hero, npc, creature, object, unattended-object, mundane, supernatural, reward, consequence, save-ends, saving-throw, echelon, ground, ceiling, level, subclass, follower, retainer, sage |

**Reuse existing codes (NO new `rule` code) — link these glossary terms to their existing type:**
- Conditions → `condition/*` (bleeding, dazed, frightened, grabbed, prone, restrained, slowed, taunted, weakened)
- Movement (incl. **pushed/pulled/slide/vertical** → `movement/forced-movement`; shift→`movement/shifting`; walk/burrow/climb-or-swim/jump/crawl/fly/hover/teleport/difficult-terrain/damaging-terrain/high-ground → their `movement/*`)
- Skills (named skills) → `skill/*`; the shared combat actions/maneuvers (Charge/Hide/Grab/Free Strike/…) → `feature.trait.common.*`
- Classes/ancestries/careers/kits/perks/complications/titles/cultures/treasure-categories/specific negotiation motivations/specific gods/chapters → their existing types.

**Slug rules:** `@id` = `Slugify(term)` (lowercase, hyphenated). Drop trailing punctuation. Where a heading's name differs from the desired term (e.g. heading "Dying and Death" but term "dying"), set `@id` explicitly. Each full `rule.<group>/<id>` must be **globally unique** (the classifier just concatenates; duplicate ids within a group collide silently — Phase 0 guarantees uniqueness).

---

## Phase 0 — Term inventory & code mapping

### Task 0: Build the canonical term→code mapping file

**Files:**
- Create: `steel-etl/docs/rule-term-mapping.md`

The mapping drives every later phase. It is data, produced by a defined procedure.

- [ ] **Step 1: Extract every glossary entry.** The Introduction glossary is the bolded-definition list spanning roughly lines 95–588 of `input/heroes/Draw Steel Heroes.md` (entries look like `**Term:** definition.`). List them:

Run:
```bash
devbox run -- bash -c 'cd steel-etl && grep -nE "^\*\*[A-Z][^*]*:\*\*" "input/heroes/Draw Steel Heroes.md" | sed -n "1,200p"'
```
Expected: ~150 `**Term:**` lines (the glossary).

- [ ] **Step 2: Find each term's anchor heading.** For each glossary term, locate its defining headed section:

Run (repeat per term, or scan chapter headings):
```bash
devbox run -- bash -c 'cd steel-etl && grep -nE "^#{1,6} " "input/heroes/Draw Steel Heroes.md"' > /tmp/headings.txt
```
Match each term to the most specific heading whose section defines it (e.g. `winded`→`#### Winded` L21975; `flanking`→`### Flanking` L21915; `pushed`→`#### Forced Movement` via the existing `movement/forced-movement`). Confirmed anchors already located:

| Term | Anchor heading (line) |
|------|----------------------|
| flanking | `### Flanking` (21915) |
| cover | `### Cover` (21923) |
| concealment | `### Concealment` (21927) |
| winded | `#### Winded` (21975) |
| dying | `#### Dying and Death` (21981) — `@id: dying` |
| suffocating | `### Suffocating` (22032) |
| stamina | `### Stamina` (21965) |
| recoveries | `#### Recoveries and Recovery Value` (21971) / Basics `#### Recoveries` (866) — pick Combat 21971; `@id: recoveries` |
| surge | `#### Surges` (4505) — `@id: surge` |
| size | `#### Size and Space` (21323) — `@id: size`; also `space` `@id: space` if separate, else share |
| characteristics + might/agility/reason/intuition/presence | `### Characteristics` (637) + `#### Might/Agility/Reason/Intuition/Presence` (52/56/60/64/68 within Basics) |
| edge / bane | `##### Edge` / `##### Bane` (within Basics Edges and Banes ~723) |
| hero-token | `### Hero Tokens` (769) |
| renown | `## Renown` (26857) or `#### Renown` (3611) — pick the Rewards/Titles `## Renown` 26857 |
| forced-movement (pushed/pulled/slide/vertical) | reuse `movement/forced-movement` (no new code) |
| damage / damage-type / damage-immunity / damage-weakness | `### Damage`(21935)/`#### Damage Types`(21939)/`##### Damage Immunity`(21945)/`##### Damage Weakness`(21955) |

- [ ] **Step 3: Write the mapping table.** Columns: `Term | Variants | Decision | Code | Anchor heading line`. `Decision` ∈ {`new-rule`, `reuse:<existing-code>`, `skip` (pure flavor/setting with no rule)}. Example rows:

```markdown
| Term | Variants | Decision | Code | Anchor (line) |
|------|----------|----------|------|---------------|
| Flanking | flanking | new-rule | mcdm.heroes.v1/rule.combat/flanking | ### Flanking (21915) |
| Cover | cover | new-rule | mcdm.heroes.v1/rule.combat/cover | ### Cover (21923) |
| Winded | winded | new-rule | mcdm.heroes.v1/rule.health/winded | #### Winded (21975) |
| Edge | edge, edges | new-rule | mcdm.heroes.v1/rule.dice/edge | ##### Edge (~727) |
| Bane | bane, banes | new-rule | mcdm.heroes.v1/rule.dice/bane | ##### Bane (~733) |
| Surge | surge, surges | new-rule | mcdm.heroes.v1/rule.resource/surge | #### Surges (4505) |
| Might | might | new-rule | mcdm.heroes.v1/rule.character/might | #### Might (~641) |
| Pushed | pushed, push | reuse | mcdm.heroes.v1/movement/forced-movement | #### Forced Movement (21620) |
| Prone | prone | reuse | mcdm.heroes.v1/condition/prone | (existing) |
| Vasloria | vasloria | new-rule | mcdm.heroes.v1/rule.world/vasloria | #### Vasloria (~924) |
```

- [ ] **Step 4: Verify completeness & uniqueness.** Every glossary `**Term:**` from Step 1 appears as a row. No two `new-rule` rows share a `Code`. No `Code` collides with an existing registry code:

Run:
```bash
devbox run -- bash -c 'cd steel-etl && jq -r ".codes[]" classification.json | grep "^mcdm.heroes.v1/rule" || echo "(no rule codes yet — expected before Phase 3)"'
```
Expected before Phase 3: none. Manually confirm each planned `Code` is unique within the mapping (sort the Code column; no dupes).

- [ ] **Step 5: Commit the mapping.**
```bash
cd steel-etl && git add docs/rule-term-mapping.md && git commit -m "docs: term->rule-SCC mapping for glossary/rules linking"
```

---

## Phase 1 — `rule` SCC type (parser + classifier)

### Task 1: RuleParser

**Files:**
- Create: `steel-etl/internal/content/rule.go`
- Modify: `steel-etl/internal/content/registry.go` (register near `GodParser`, ~line 35)
- Test: `steel-etl/internal/content/rule_test.go`

- [ ] **Step 1: Write the failing test.**

```go
package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestRuleParserGroupedTypePath(t *testing.T) {
	p := &RuleParser{}
	sec := &parser.Section{
		Heading:    "Flanking",
		Annotation: map[string]string{"type": "rule", "group": "combat", "id": "flanking"},
	}
	got, err := p.Parse(context.NewContextStack(), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if want := []string{"rule", "combat"}; len(got.TypePath) != 2 || got.TypePath[0] != want[0] || got.TypePath[1] != want[1] {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
	if got.ItemID != "flanking" {
		t.Errorf("ItemID = %q, want flanking", got.ItemID)
	}
	if got.Frontmatter["type"] != "rule" {
		t.Errorf("type = %v, want rule", got.Frontmatter["type"])
	}
	if got.Frontmatter["name"] != "Flanking" {
		t.Errorf("name = %v, want Flanking", got.Frontmatter["name"])
	}
}

func TestRuleParserDerivesIDFromHeading(t *testing.T) {
	p := &RuleParser{}
	sec := &parser.Section{
		Heading:    "Dying and Death",
		Annotation: map[string]string{"type": "rule", "group": "health"},
	}
	got, _ := p.Parse(context.NewContextStack(), sec)
	if got.ItemID != "dying-and-death" {
		t.Errorf("ItemID = %q, want dying-and-death (slug of heading when @id absent)", got.ItemID)
	}
}

func TestRuleParserFlatWhenNoGroup(t *testing.T) {
	p := &RuleParser{}
	sec := &parser.Section{Heading: "Reward", Annotation: map[string]string{"type": "rule", "id": "reward"}}
	got, _ := p.Parse(context.NewContextStack(), sec)
	if len(got.TypePath) != 1 || got.TypePath[0] != "rule" {
		t.Errorf("TypePath = %v, want [rule]", got.TypePath)
	}
}
```

- [ ] **Step 2: Run it; verify it fails.**
Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestRuleParser'`
Expected: FAIL — `undefined: RuleParser`.

- [ ] **Step 3: Implement `rule.go`.**

```go
package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// RuleParser handles @type: rule sections — rules-glossary terms that get a
// grouped, human-readable SCC code (rule.<group>/<id>) so prose can link to
// the rule that defines them. Mirrors GodParser; the optional @group annotation
// adds the second TypePath segment (rule.<group>); with no @group it is flat
// (rule/<id>). RuleParser never accumulates a parent path, so codes stay flat
// within their group even when the annotated heading is nested in a chapter.
type RuleParser struct{}

func (p *RuleParser) Type() string { return "rule" }

func (p *RuleParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	typePath := []string{"rule"}
	if group, ok := section.Annotation["group"]; ok && group != "" {
		typePath = []string{"rule", group}
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": name,
			"type": "rule",
		},
		Body:     section.FullBodySource(),
		TypePath: typePath,
		ItemID:   id,
	}, nil
}
```

- [ ] **Step 4: Register the parser.** In `internal/content/registry.go`, after `r.Register(&GodParser{})`:
```go
	r.Register(&RuleParser{})
```

- [ ] **Step 5: Run tests; verify pass.**
Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestRuleParser'`
Expected: PASS (3 tests).

- [ ] **Step 6: Classifier sanity test (grouped code string).**

Add to `steel-etl/internal/scc/classifier_test.go`:
```go
func TestClassifyGroupedRule(t *testing.T) {
	got := Classify("mcdm.heroes.v1", []string{"rule", "combat"}, "flanking")
	if want := "mcdm.heroes.v1/rule.combat/flanking"; got != want {
		t.Errorf("Classify = %q, want %q", got, want)
	}
}
```
Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/scc/ -run TestClassifyGroupedRule'`
Expected: PASS.

- [ ] **Step 7: Full unit suite + commit.**
```bash
devbox run -- bash -c 'cd steel-etl && go test ./...'   # expect: all green
cd steel-etl && git add internal/content/rule.go internal/content/rule_test.go internal/content/registry.go internal/scc/classifier_test.go && git commit -m "feat(scc): add grouped 'rule' type (rule.<group>/<id>) for rules-glossary linking"
```

---

## Phase 2 — Publish `rule/*` pages in Browse

### Task 2: Add `rule/` to the Browse section + verify reachability

**Files:**
- Modify: `v2/site.yaml` (Browse `include:` list, ~line 60)

- [ ] **Step 1: Add the include.** In `v2/site.yaml`, in the `- name: Browse` block's `include:` list, add (alongside `condition/`, `movement/`):
```yaml
      - rule/
```
(Dotted type components map to directories, so `rule.combat/flanking` → `rule/combat/flanking.md`; the `rule/` prefix matches every group. `matchesSection` is prefix-based.)

- [ ] **Step 2: Defer card index.** No `cards.go` index is required for first publish — the default index renders. (Optional later enhancement: a `rule` card type in `internal/site/cards.go`; out of scope for this plan.)

- [ ] **Step 3: Verify after Phase 3 annotations exist** (cross-phase): see Phase 3 Task 3 Step 4 (build site, confirm `v2/docs/Browse/rule/...` pages exist and a sample resolves). Commit the site.yaml change with the first annotation batch.

---

## Phase 3 — Annotate the rules sections (mint the codes)

For each chapter, add `<!-- @type: rule | @group: <g> | @id: <slug> -->` on the line **immediately before** the target heading (the annotation pre-pass binds an annotation to the next heading). Use the Phase-0 mapping. Annotate only `new-rule` rows; leave `reuse`/`skip` rows alone.

**Pattern (example — Flanking):**
```markdown
<!-- @type: rule | @group: combat | @id: flanking -->
### Flanking
```

### Task 3a: Annotate The Basics chapter rules (lines ~590–1055)

**Files:** Modify `steel-etl/input/heroes/Draw Steel Heroes.md`

- [ ] **Step 1:** Annotate (per mapping): `### Characteristics`→`character/characteristic`; `#### Might/Agility/Reason/Intuition/Presence`→`character/<name>`; `##### Edge`→`dice/edge`; `##### Bane`→`dice/bane`; `### Power Rolls`→`dice/power-roll`; `##### Natural Roll`→`dice/natural-roll`; `### Hero Tokens`→`resource/hero-token`; `#### Recoveries`→(reuse Combat anchor — skip here); `#### Victories`→`resource/victories`; `#### Experience`→`resource/experience`; `#### Heroic Resources`→`resource/heroic-resource`; `#### Respite`→`downtime/respite`; `### Echelons of Play`→`general/echelon`; `#### Unattended Objects`→`general/unattended-object`; `### Supernatural or Mundane`→split into `general/supernatural` (+ `general/mundane` if separate heading exists, else map `mundane` to this anchor); `#### Vasloria`→`world/vasloria`; world/setting headings per mapping.

- [ ] **Step 2: gen 0-WARN check.**
```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml' 2>&1 | grep -iE "WARN|error" || echo clean
```
Expected: `clean` (new codes mint; no unresolved links yet because we haven't linked anything).

- [ ] **Step 3: Confirm the codes exist.**
```bash
devbox run -- bash -c 'cd steel-etl && jq -r ".codes[]" classification.json | grep "^mcdm.heroes.v1/rule" | sort'
```
Expected: the Basics rule codes (`rule.character/might`, `rule.dice/edge`, …).

- [ ] **Step 4: Commit.**
```bash
cd steel-etl && git add "input/heroes/Draw Steel Heroes.md" && git commit -m "annotate: rule codes for The Basics (characteristics, dice, resources, world)"
```

### Task 3b: Annotate Tests + Combat chapters (the bulk)

**Files:** Modify `steel-etl/input/heroes/Draw Steel Heroes.md`

- [ ] **Step 1: Tests chapter** (~20337–20580): `### When to Make a Test`/`### How to Make a Test`→`test/test`; `#### Test Difficulty`→`test/test-difficulty`; the skill-group headings in the Skills chapter (`##### Intrigue Skills`, `##### Lore Skills`, and the crafting/exploration/interpersonal group headings)→`test/<group>-skills`.

- [ ] **Step 2: Combat chapter** (~21311–22100): `#### Size and Space`→`combat/size`; `#### Sides`→`combat/side`; `### Combat Round`→`combat/combat-round`; `#### Determine Surprise`→`combat/surprised`; `### Taking a Turn`→`combat/turn`; `#### Triggered Actions and Free Triggered Actions`→`combat/triggered-action`; `#### Free Maneuvers`→`combat/free-maneuver`; `#### No-Action Activities`→`combat/no-action`; `#### Falling`→`health/falling`; `### Flanking`→`combat/flanking`; `### Cover`→`combat/cover`; `### Concealment`→`combat/concealment`; `### Damage`→`damage/damage`; `#### Damage Types`→`damage/damage-type`; `##### Damage Immunity`→`damage/damage-immunity`; `##### Damage Weakness`→`damage/damage-weakness`; `### Stamina`→`health/stamina`; `#### Recoveries and Recovery Value`→`health/recoveries`; `#### Winded`→`health/winded`; `#### Dying and Death`→`health/dying`; `### Suffocating`→`health/suffocating`; line-of-effect/opportunity-attack/strike/critical-hit/distance/area-shapes→`combat/*` (add small headings only if no section exists — see Task 3e).

- [ ] **Step 3: gen 0-WARN check** (same command as 3a Step 2). Expected: `clean`.

- [ ] **Step 4: Build site; verify pages publish + Read chapter intact.**
```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all'
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml'
ls v2/docs/Browse/rule/combat/flanking* 2>/dev/null && echo "rule page published"
grep -q "Flanking" "v2/docs/Read/heroes/combat.md" && echo "Read chapter still renders Flanking inline"
```
Expected: both echoes print.

- [ ] **Step 5: Commit** (include the Phase-2 `v2/site.yaml` change here).
```bash
cd steel-etl && git add "input/heroes/Draw Steel Heroes.md" && git commit -m "annotate: rule codes for Tests + Combat (combat/health/damage/test groups)"
cd /home/scott/code/steelCompendium/workspace && git add v2/site.yaml && git commit -m "site: publish Browse/rule/* (rule-glossary pages)"
```

### Task 3c: Annotate Downtime, Negotiation, Rewards/Titles, Treasures

**Files:** Modify `steel-etl/input/heroes/Draw Steel Heroes.md`

- [ ] **Step 1:** Per mapping: downtime project headings→`downtime/*` (downtime-project, crafting-project, research-project, project-goal/points/roll/source/event, respite-activity, guide, item-prerequisite); negotiation `interest/patience/motivation/pitfall` concept headings→`negotiation/*`; `## Renown`→`resource/renown`; treasure concept headings (leveled-treasure, trinket, consumable, enhancement, implement)→`treasure/*` **only if** they have headings — else map to nearest parent (e.g. `treasure/leveled-benefits`).

- [ ] **Step 2: gen 0-WARN check.** Expected: `clean`.
- [ ] **Step 3: Commit.** `git commit -m "annotate: rule codes for downtime/negotiation/rewards/treasure concepts"`

### Task 3d: Annotate Introduction glossary-only terms

Some glossary terms have **no** chapter section (e.g. `manifold`, `orden`, `save-ends`, `consequence`). For these, annotate the **glossary entry's nearest headed parent** is not possible (the glossary is a flat list). Instead: per the user's winded→dying rule, map them to the closest real section if one exists; if truly glossary-only, add a small dedicated heading.

- [ ] **Step 1:** For each glossary-only term in the mapping with no anchor, choose: (a) map to an existing related rule code (e.g. `save-ends`→`general/saving-throw`), or (b) add a minimal headed subsection in the most relevant chapter and annotate it. Record the choice in `docs/rule-term-mapping.md`.
- [ ] **Step 2: gen 0-WARN check.** Expected: `clean`.
- [ ] **Step 3: Commit.** `git commit -m "annotate: rule codes for glossary-only terms (nearest-parent/new headings)"`

### Task 3e: Add headings for unheaded combat sub-rules (only if needed)

A few combat terms (e.g. `pushed`/`pulled`/`slide`) are sub-rules of `#### Forced Movement` without their own headings → **reuse `movement/forced-movement`** (no new heading). `distance`, `line`, `cube`, `wall` (area shapes) and `opportunity-attack` may lack headings.

- [ ] **Step 1:** For each genuinely-unheaded term still wanted as its own code, add a small heading at its definition point and annotate it; otherwise map to the nearest parent in the mapping. Prefer reuse over new headings to minimize source churn.
- [ ] **Step 2: gen 0-WARN check.** Expected: `clean`.
- [ ] **Step 3: Commit.** `git commit -m "annotate: rule codes for area-shape / misc combat terms"`

---

## Phase 4 — Reference table + linking-guide

### Task 4: Document the `rule` type

**Files:**
- Modify: `steel-etl/docs/linking-reference.md`
- Modify: `steel-etl/docs/linking-guide.md`

- [ ] **Step 1:** In `linking-reference.md`, add a `## Rules (rule.<group>/<term>)` section: one sub-table per group, columns `Display Name | Variants | SCC Code`, generated from `docs/rule-term-mapping.md` (`new-rule` rows). Update the `**Total linkable terms:**` count.

- [ ] **Step 2:** Add a strong **Disambiguation** note: many rule terms are common English words (`edge`, `bane`, `cover`, `size`, `distance`, `surge`, `strike`). Link only when referring to the mechanic — e.g. **link** "gains an edge on the roll", "has cover", "a size 1 creature", "the Winded value"; **don't link** "the edge of the cliff", "cover the distance", "a strike of luck". Same per-instance rule as conditions/skills.

- [ ] **Step 3:** In `linking-guide.md`, add a dated note describing the new `rule` type + the grouped convention + the disambiguation policy.

- [ ] **Step 4: Commit.** `git commit -m "docs: add 'rule' type to linking-reference + linking-guide"`

---

## Phase 5 — Link the glossary (every entry)

The Introduction glossary is the bounded, highest-value target: every `**Term:**` entry links its headword to its code (existing or new `rule.*`).

### Task 5: Link all glossary headwords

**Files:** Modify `steel-etl/input/heroes/Draw Steel Heroes.md`

- [ ] **Step 1:** For each glossary entry, wrap the bold headword in a link to its mapped code. Example transforms:
```markdown
**Flanking:** When two or more allied creatures…
→ **[Flanking](scc:mcdm.heroes.v1/rule.combat/flanking):** When two or more allied creatures…

**Winded:** A state a creature enters…
→ **[Winded](scc:mcdm.heroes.v1/rule.health/winded):** A state a creature enters…
```
Skip entries whose `Decision` is `skip`. Entries already linked (conditions/movement done in prior passes) — link the headword to the existing code if not already.

- [ ] **Step 2: gen 0-WARN check.** Expected: `clean` (every link now resolves to a minted/existing code). If any WARN: the link points at a code Phase 3 didn't mint — fix the mapping/annotation.

- [ ] **Step 3: No malformed links.**
```bash
grep -nE '\]\(scc:[^)]*\)\]\(scc:|\[\[|\]\(\)' "steel-etl/input/heroes/Draw Steel Heroes.md" | head
```
Expected: no output.

- [ ] **Step 4: Commit.** `git commit -m "link: every Introduction glossary headword -> its SCC code (rule + existing types)"`

---

## Phase 6 — Full-document per-instance sweep

Link in-prose occurrences across all chapters. This is per-instance judgment (NOT a blanket script), mirroring the conditions/skills passes. Work **one term (or small term-set) at a time**, using `scripts/link_apply.py` for safe single-rule application (it skips headings/existing-links/comments and is dry-run by default) and reading each occurrence for mundane-vs-mechanic.

### Task 6 (repeat per term-batch): Sweep one rule term

**Files:** Modify `steel-etl/input/heroes/Draw Steel Heroes.md`

- [ ] **Step 1: Report occurrences.**
```bash
devbox run -- bash -c 'cd steel-etl && python3 scripts/link_audit_category.py "rule.<group>/<term>"'
```
(For section-aware own-section exclusion, use `scripts/link_audit_sectioned.py "<Term>"`.)

- [ ] **Step 2: Decide per instance.** Apply the disambiguation rule from Phase 4 Step 2. For high-frequency common words, only link clear mechanic uses. **Do not** link a term inside its own definition section heading.

- [ ] **Step 3: Apply (dry-run first).**
```bash
devbox run -- bash -c 'cd steel-etl && python3 scripts/link_apply.py "(<regex capturing the term as group 1>)" "rule.<group>/<term>" <excl-start>-<excl-end>'   # dry run
# review output, then re-run with --apply
```
Use line-range exclusions for the term's own definition section.

- [ ] **Step 4: gen 0-WARN + malformed check** (commands as Phase 5 Steps 2–3). Expected: `clean`, no malformed.

- [ ] **Step 5: Commit per chapter or per term-batch.**
```bash
cd steel-etl && git add "input/heroes/Draw Steel Heroes.md" && git commit -m "link: <term> rule cross-references (<chapter/scope>)"
```

**Suggested batch order** (highest yield / least ambiguous first): winded, dying, suffocating, flanking, cover, concealment, surge, recoveries, stamina, hero-token, heroic-resource → then characteristics (might/agility/reason/intuition/presence) → then the common-word tail (edge, bane, size, distance, strike, damage) with careful per-instance review.

---

## Phase 7 — Verify, document, commit, push

### Task 7: Final verification + doc/memory updates

- [ ] **Step 1: Full gen + tests + site.**
```bash
devbox run -- bash -c 'cd steel-etl && go test ./... && go run ./cmd/steel-etl gen --config pipeline.yaml --all' 2>&1 | grep -iE "WARN|error|FAIL" || echo "clean+green"
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml' 2>&1 | tail -3
```
Expected: `clean+green`; site builds.

- [ ] **Step 2: Malformed-link grep** (whole doc).
```bash
grep -nE '\]\(scc:[^)]*\)\]\(scc:|\[\[|\]\(\)|\(scc:[^)]*scc:' "steel-etl/input/heroes/Draw Steel Heroes.md" | head
```
Expected: no output.

- [ ] **Step 3: Spot-check rendered pages.** Open `v2/docs/Browse/rule/combat/flanking/…` and a Read chapter; confirm the rule page renders the full section and inbound links resolve.

- [ ] **Step 4: Update docs/memory.**
  - `FOLLOWUPS.md` — add/close the relevant item; note any deferred tail.
  - workspace `CLAUDE.md` SCC section — bump link/term counts; mention the new `rule.<group>/<term>` type.
  - `steel-etl/CLAUDE.md` — note `rule` in the parser list (registry now has 25 parsers).
  - memory `link-audit-tooling.md` — record the `rule` type + mapping file.

- [ ] **Step 5: Commit + push both repos, bump submodule pointer** (per the established workflow: commit steel-etl → push; bump workspace `steel-etl` pointer + docs → rebase onto origin/main → push). Leave `just deploy-v2` for the user.

---

## Self-Review notes

- **Spec coverage:** new SCC codes (Phase 1), grouped+human-readable+predictable (taxonomy + `rule.<group>/<id>`), find terms (Phase 0 inventory + glossary extraction), link the named examples (flanking/cover/winded/recoveries/suffocating/pushed/pulled/slide/observed/surge/edge/bane/size/distance/characteristics/renown/skill-groups/surprised — each appears in the taxonomy/mapping), "every glossary entry should have a link" (Phase 5), nearest-parent fallback for unheaded terms (Phase 3d/3e, e.g. winded→dying pattern, pushed→forced-movement). Covered.
- **`observed`:** not a glossary headword on its own; maps to the Hide/Concealment rules — link `observed`/`hidden` to `rule.combat/concealment` (record in mapping).
- **Reuse vs new:** conditions/movement/skills/classes/etc. reuse existing codes (no duplication); pushed/pulled/slide reuse `movement/forced-movement`.
- **Risk — chapter render:** annotating mid-chapter headings with `@type: rule` must not break the Read-tab book-faithful render or create odd Browse nesting → verified in Phase 3b Step 4 and Phase 7 Step 3.
- **Risk — uniqueness:** duplicate `rule.<group>/<id>` would silently collide → Phase 0 Step 4 enforces global uniqueness before annotating.
- **Type consistency:** parser `RuleParser`, `Type()=="rule"`, annotation keys `type/group/id`, code shape `rule.<group>/<id>` used identically across Phases 1, 3, 4, 5, 6.
