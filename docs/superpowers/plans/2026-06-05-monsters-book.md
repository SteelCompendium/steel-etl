# Monsters Book Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Process `Draw Steel Monsters.md` through `steel-etl` into all output formats and onto the v2 website, completing Phase 5 (Monsters + Multi-Book) of the architecture redesign.

**Architecture:** Add four new content parsers (`monster`, `statblock`, `featureblock`, `dynamic-terrain`) plus a statblock SDK transform and schema. Monsters classify into a nested SCC hierarchy mirroring the existing `treasure/<tier>/<category>/<item>` pattern: statblocks at `monster.<category>.statblock/<item>` (path `monster/<category>/statblock/<item>.md`) and per-group Malice featureblocks at `monster.<category>/<malice>` (sibling of the `statblock/` folder). The annotated source is hand-labeled, validated one group at a time against legacy `data-bestiary-md`, then expanded book-wide. The site builder gains a Bestiary tab.

**Tech Stack:** Go (goldmark, cobra), table-driven tests, `devbox run --` for all Go commands, MkDocs Material for the site.

**Spec:** `plans/architecture-redesign/phases.md` (Phase 5), `plans/architecture-redesign/scc-taxonomy.md`, `steel-etl/ANNOTATION-GUIDE.md`.

---

## Background & Key Facts (read before starting)

**Run all Go commands through devbox** from the workspace root, e.g.
`devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestStatblock -v'`.
The justfile recipes (`just test`, `just build`) only work inside `devbox shell`.

**Source document:** `data-gen/input/monsters/Draw Steel Monsters.md` (~26k lines, 1.5 MB). It is copied into `steel-etl/input/monsters/` in Part F — until then, read it from the `data-gen` path. Heading structure is highly regular:

| Heading level | Meaning | Count (approx) |
|---------------|---------|-------|
| `#` H1 | Top chapters: `# Monster Basics`, `# Monsters`, `# Dynamic Terrain`, `# Retainers` | 4 |
| `##` H2 | Monster groups (`## Goblins`, `## Dragons`, …) under `# Monsters` | ~55 |
| `###` H3 | Dynamic-terrain categories (`### Environmental Hazards`, `### Fieldworks`, …) | ~5 |
| `#######` H7 | Individual statblocks (`####### Goblin Cursespitter`) — also the retainer creatures | ~437 |
| `#########` H9 | Malice featureblocks (`######### Goblin Malice (Malice Features)`) and dynamic-terrain objects (`######### Angry Beehive (Level 2 Hazard Hexer)`) | ~130 |

**SCC mechanics (already implemented — do not change):** `scc.Classify(source, typePath, itemID)` joins `typePath` with **dots** into one type component, then joins `source / typeComponent / itemID` with **slashes** (`internal/scc/classifier.go`). `sccToRootPath` splits the SCC on `/`, then splits each segment after the source on `.` to make directory segments (`internal/scc/resolver.go`). Therefore:

- `TypePath ["monster","goblins","statblock"]`, item `goblin-warrior` → SCC `mcdm.monsters.v1/monster.goblins.statblock/goblin-warrior` → file `monster/goblins/statblock/goblin-warrior.md`.
- `TypePath ["monster","goblins"]`, item `goblin-malice` → SCC `mcdm.monsters.v1/monster.goblins/goblin-malice` → file `monster/goblins/goblin-malice.md` (sibling of the `statblock/` folder ✓).
- `TypePath ["monster","goblins"]`, item `goblins` → SCC `mcdm.monsters.v1/monster.goblins/goblins` → file `monster/goblins/goblins.md` (the group lore page, **inside** the `monster/goblins/` directory alongside the Malice featureblock).

**Context provision (already implemented):** the pipeline pushes **every** annotated section's full annotation map into the context stack (`internal/pipeline/pipeline.go`: `contextStack.Push(...)`). So an `@category: goblins` annotation on a `## Goblins` group is visible to all descendant statblocks via `ctx.Lookup(level, "category")`. Container parsers that return no `TypePath`/`ItemID` produce no output file but still seed context (see `FeatureGroupParser`, `TreasureGroupParser`).

**Target structured format:** the SDK statblock schema is `data-sdk-npm/src/schema/statblock.schema.json` (required: name, type, level, role, organization, keywords, ev, stamina, speed, size, stability, free_strike, 5 ability scores; optional: immunities, weaknesses, movement, with_captain, features[], metadata). Legacy reference output: `data/data-bestiary-json/Monsters/<Group>/Statblocks/<name>.json` and `data/data-bestiary-md/Monsters/Monsters/<Group>/...`.

**Statblock body anatomy** (raw markdown, e.g. `Goblin Cursespitter`):
1. A 4-row markdown stat grid. Header row cells: `keywords | - | Level N | <Org> <Role> | EV N`. Then three label/value rows where each cell is `**VALUE**<br>Label` (labels: Size, Speed, Stamina, Stability, Free Strike, Immunity, Movement, With Captain, Weakness, Might, Agility, Reason, Intuition, Presence).
2. One or more feature blockquotes: `> EMOJI **Name (parenthetical)**` followed by optional keyword/usage and distance/target tables, then either a power-roll tier list (`- **≤11:** …`, `- **12-16:** …`, `- **17+:** …`) or plain trait text.

**Role/organization vocabulary** (for splitting the role cell `Horde Hexer` → organization `Horde`, role `Hexer`):
- Organizations: `Minion, Horde, Platoon, Elite, Solo, Leader, Retainer`
- Roles: `Ambusher, Artillery, Brute, Controller, Defender, Harrier, Hexer, Support, Mount`
- Some cells are organization-only (`Leader`, `Solo`) → role empty. Order varies (`Harrier Retainer`), so match each word against both sets rather than assuming position.

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/content/monster.go` | Create | `MonsterParser` (group lore page + category context), `StatblockParser`, `FeatureblockParser` |
| `internal/content/monster_test.go` | Create | Table-driven tests for the three parsers using real fixtures |
| `internal/content/statblock_parse.go` | Create | Pure helpers: `parseStatGrid`, `splitRoleCell`, `parseStatblockFeatures` |
| `internal/content/statblock_parse_test.go` | Create | Unit tests for the grid/role/feature helpers |
| `internal/content/dynamic_terrain.go` | Create | `DynamicTerrainParser` (EV/Stamina/Size + activate/effect blockquotes) |
| `internal/content/dynamic_terrain_test.go` | Create | Tests for the terrain parser |
| `internal/content/registry.go` | Modify | Register the four new parsers |
| `internal/output/sdk_transform.go` | Modify | Add `case "statblock"` → `transformStatblock` |
| `internal/output/statblock_transform.go` | Create | `transformStatblock` builds SDK statblock JSON with embedded `features[]` |
| `internal/output/statblock_transform_test.go` | Create | Conformance test against legacy `Goblin Cursespitter.json` |
| `internal/output/schema_validation_test.go` | Modify | Add statblock schema validation cases |
| `schemas/statblock.schema.json` | Create | Copy of the SDK statblock schema for in-repo validation |
| `testdata/monsters/*.md`, `*.json` | Create | Fixtures extracted from the real source + legacy output |
| `input/monsters/Draw Steel Monsters.md` | Create | Annotated copy of the source document (Parts F–H) |
| `pipeline.yaml` | Modify | Point the monsters book at the in-repo input |
| `ANNOTATION-GUIDE.md` | Modify | Document the monster annotation types (promote from "future") |
| `../v2/site.yaml` | Modify | Add the Bestiary section + Browse includes |
| `internal/site/cards.go` | Modify | Add monster/statblock/terrain icon + card support (if needed) |
| `plans/architecture-redesign/scc-taxonomy.md` | Modify | Update monster taxonomy to the nested shape |
| `../ARCHITECTURE.md`, `../CLAUDE.md`, `CLAUDE.md` | Modify | Registry counts, monster pipeline notes |

---

## Part A — Statblock stat-grid + role parsing (pure helpers)

Start with the pure, side-effect-free helpers. They are the foundation and the easiest to TDD.

### Task A1: Parse the stat grid into a label→value map

**Files:**
- Create: `internal/content/statblock_parse.go`
- Test: `internal/content/statblock_parse_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/content/statblock_parse_test.go`:

```go
package content

import (
	"reflect"
	"testing"
)

const cursespitterGrid = "" +
	"| Goblin, Humanoid  |           -           |      Level 1      |      Horde Hexer      |         EV 3         |\n" +
	"|:-----------------:|:---------------------:|:-----------------:|:---------------------:|:--------------------:|\n" +
	"|  **1S**<br>Size   |    **5**<br>Speed     | **10**<br>Stamina |  **0**<br>Stability   | **1**<br>Free Strike |\n" +
	"| **-**<br>Immunity | **Climb**<br>Movement |         -         | **-**<br>With Captain |  **-**<br>Weakness   |\n" +
	"|  **-2**<br>Might  |   **+1**<br>Agility   |  **0**<br>Reason  |  **+2**<br>Intuition  |  **0**<br>Presence   |\n"

func TestParseStatGrid(t *testing.T) {
	got := parseStatGrid(cursespitterGrid)

	wantHeader := statHeader{
		keywords:     []string{"Goblin", "Humanoid"},
		level:        1,
		organization: "Horde",
		role:         "Hexer",
		ev:           "3",
	}
	if !reflect.DeepEqual(got.header, wantHeader) {
		t.Errorf("header: got %+v, want %+v", got.header, wantHeader)
	}

	wantLabels := map[string]string{
		"Size": "1S", "Speed": "5", "Stamina": "10", "Stability": "0", "Free Strike": "1",
		"Immunity": "-", "Movement": "Climb", "With Captain": "-", "Weakness": "-",
		"Might": "-2", "Agility": "+1", "Reason": "0", "Intuition": "+2", "Presence": "0",
	}
	if !reflect.DeepEqual(got.labels, wantLabels) {
		t.Errorf("labels: got %+v, want %+v", got.labels, wantLabels)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestParseStatGrid -v'`
Expected: compile error — `parseStatGrid` / `statHeader` undefined.

- [ ] **Step 3: Implement the grid parser**

Create `internal/content/statblock_parse.go`:

```go
package content

import (
	"regexp"
	"strconv"
	"strings"
)

// statHeader holds the values from a statblock grid's header row.
type statHeader struct {
	keywords     []string
	level        int
	organization string
	role         string
	ev           string
}

// statGrid is the fully parsed stat grid: header row + label→value map.
type statGrid struct {
	header statHeader
	labels map[string]string
}

var (
	// "**VALUE**<br>Label" or "**VALUE**<br/>Label" — value may be empty bold.
	cellRe  = regexp.MustCompile(`\*\*(.*?)\*\*\s*<br\s*/?>\s*([A-Za-z][A-Za-z ]*)`)
	levelRe = regexp.MustCompile(`Level\s+(\d+)`)
	evRe    = regexp.MustCompile(`EV\s+([0-9A-Za-z+ /x-]+)`)
)

var knownOrganizations = map[string]bool{
	"Minion": true, "Horde": true, "Platoon": true,
	"Elite": true, "Solo": true, "Leader": true, "Retainer": true,
}

var knownRoles = map[string]bool{
	"Ambusher": true, "Artillery": true, "Brute": true, "Controller": true,
	"Defender": true, "Harrier": true, "Hexer": true, "Support": true, "Mount": true,
}

// splitRoleCell separates an "Org Role" cell (e.g. "Horde Hexer") into
// organization and role by matching each word against the known vocabularies.
// Organization-only cells ("Leader", "Solo") return an empty role.
func splitRoleCell(cell string) (organization, role string) {
	for _, w := range strings.Fields(cell) {
		switch {
		case knownOrganizations[w]:
			organization = w
		case knownRoles[w]:
			role = w
		}
	}
	return organization, role
}

// gridRows returns the non-separator table rows split into trimmed cells.
func gridRows(grid string) [][]string {
	var rows [][]string
	for _, line := range strings.Split(grid, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if strings.Contains(line, "---") { // separator row
			continue
		}
		raw := strings.Split(strings.Trim(line, "|"), "|")
		cells := make([]string, len(raw))
		for i, c := range raw {
			cells[i] = strings.TrimSpace(c)
		}
		rows = append(rows, cells)
	}
	return rows
}

// parseStatGrid parses a statblock's 4-row markdown grid.
func parseStatGrid(grid string) statGrid {
	out := statGrid{labels: map[string]string{}}
	rows := gridRows(grid)
	if len(rows) == 0 {
		return out
	}

	// Header row.
	header := rows[0]
	if len(header) > 0 {
		out.header.keywords = splitCommaList(header[0])
	}
	joined := strings.Join(header, " | ")
	if m := levelRe.FindStringSubmatch(joined); m != nil {
		out.header.level, _ = strconv.Atoi(m[1])
	}
	if m := evRe.FindStringSubmatch(joined); m != nil {
		out.header.ev = strings.TrimSpace(m[1])
	}
	// Role cell is the one (besides the EV cell) containing an org/role word.
	for _, cell := range header {
		if strings.Contains(cell, "EV ") {
			continue
		}
		if org, role := splitRoleCell(cell); org != "" || role != "" {
			out.header.organization = org
			out.header.role = role
			break
		}
	}

	// Label/value rows (rows[1:]).
	for _, row := range rows[1:] {
		for _, cell := range row {
			if m := cellRe.FindStringSubmatch(cell); m != nil {
				value := strings.TrimSpace(m[1])
				if value == "" {
					value = "-"
				}
				out.labels[strings.TrimSpace(m[2])] = value
			}
		}
	}
	return out
}
```

> `splitCommaList` already exists in `internal/content/helpers.go` (used by the treasure parser). Do not redefine it.

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestParseStatGrid -v'`
Expected: PASS.

- [ ] **Step 5: Add a role-split edge-case test**

Append to `internal/content/statblock_parse_test.go`:

```go
func TestSplitRoleCell(t *testing.T) {
	tests := []struct{ in, org, role string }{
		{"Horde Hexer", "Horde", "Hexer"},
		{"Elite Brute", "Elite", "Brute"},
		{"Leader", "Leader", ""},
		{"Solo", "Solo", ""},
		{"Harrier Retainer", "Retainer", "Harrier"},
		{"Minion Artillery", "Minion", "Artillery"},
	}
	for _, tt := range tests {
		org, role := splitRoleCell(tt.in)
		if org != tt.org || role != tt.role {
			t.Errorf("%q: got (%q,%q), want (%q,%q)", tt.in, org, role, tt.org, tt.role)
		}
	}
}
```

- [ ] **Step 6: Run and verify**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run "TestParseStatGrid|TestSplitRoleCell" -v'`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
cd steel-etl && git add internal/content/statblock_parse.go internal/content/statblock_parse_test.go && git commit -m "feat: statblock stat-grid + role-cell parsing helpers"
```

---

### Task A2: Parse statblock feature blockquotes into structured features

**Files:**
- Modify: `internal/content/statblock_parse.go`
- Test: `internal/content/statblock_parse_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/content/statblock_parse_test.go`:

```go
const cursespitterFeatures = "" +
	"> 🏹 **Eye of Surlach (Signature Ability)**\n" +
	">\n" +
	"> | **Magic, Ranged, Strike** |     **Main action** |\n" +
	"> |---------------------------|--------------------:|\n" +
	"> | **📏 Ranged 15**          | **🎯 One creature** |\n" +
	">\n" +
	"> **Power Roll + 2:**\n" +
	">\n" +
	"> - **≤11:** 3 corruption damage; I < 0 weakened (save ends)\n" +
	"> - **12-16:** 4 corruption damage; I < 1 weakened (save ends)\n" +
	"> - **17+:** 5 corruption damage; I < 2 weakened (save ends)\n" +
	"\n" +
	"> ⭐️ **Crafty**\n" +
	">\n" +
	"> The cursespitter doesn't provoke opportunity attacks by moving.\n"

func TestParseStatblockFeatures(t *testing.T) {
	got := parseStatblockFeatures(cursespitterFeatures)
	if len(got) != 2 {
		t.Fatalf("got %d features, want 2", len(got))
	}

	ability := got[0]
	if ability["name"] != "Eye of Surlach" {
		t.Errorf("name: got %v", ability["name"])
	}
	if ability["ability_type"] != "Signature Ability" {
		t.Errorf("ability_type: got %v", ability["ability_type"])
	}
	if ability["icon"] != "🏹" {
		t.Errorf("icon: got %v", ability["icon"])
	}
	if ability["usage"] != "Main action" {
		t.Errorf("usage: got %v", ability["usage"])
	}
	if ability["distance"] != "Ranged 15" {
		t.Errorf("distance: got %v", ability["distance"])
	}
	if ability["target"] != "One creature" {
		t.Errorf("target: got %v", ability["target"])
	}
	kw, _ := ability["keywords"].([]string)
	if len(kw) != 3 || kw[0] != "Magic" {
		t.Errorf("keywords: got %v", ability["keywords"])
	}
	effects, _ := ability["effects"].([]map[string]any)
	if len(effects) != 1 || effects[0]["tier1"] != "3 corruption damage; I < 0 weakened (save ends)" {
		t.Errorf("effects: got %v", ability["effects"])
	}

	trait := got[1]
	if trait["name"] != "Crafty" || trait["feature_type"] != "trait" {
		t.Errorf("trait: got %+v", trait)
	}
	teff, _ := trait["effects"].([]map[string]any)
	if len(teff) != 1 || teff[0]["effect"] != "The cursespitter doesn't provoke opportunity attacks by moving." {
		t.Errorf("trait effect: got %v", trait["effects"])
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestParseStatblockFeatures -v'`
Expected: compile error — `parseStatblockFeatures` undefined.

- [ ] **Step 3: Implement the feature-blockquote parser**

Append to `internal/content/statblock_parse.go`:

```go
var (
	featTitleRe = regexp.MustCompile(`^([^\sA-Za-z*][^*]*?)\s*\*\*(.+?)\*\*\s*$`)
	parenRe     = regexp.MustCompile(`^(.*?)\s*\(([^)]+)\)\s*$`)
	tierRe      = regexp.MustCompile(`^-\s*\*\*(≤?\d+(?:-\d+)?\+?):\*\*\s*(.*)$`)
	powerRollRe = regexp.MustCompile(`\*\*(Power Roll[^*]*)\*\*`)
	iconCellRe  = regexp.MustCompile(`(?:📏|🎯|🔅)?\s*\*\*(?:📏|🎯|🔅)?\s*([^*]+?)\s*\*\*`)
)

// splitBlockquoteBlocks breaks a body into individual blockquote blocks. Each
// returned block has its leading "> " markers stripped from every line.
func splitBlockquoteBlocks(body string) []string {
	var blocks []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			blocks = append(blocks, strings.Join(cur, "\n"))
			cur = nil
		}
	}
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, ">"):
			// Detect a new block: a "> EMOJI **Title**" line after we already
			// have content collected.
			stripped := strings.TrimSpace(strings.TrimPrefix(t, ">"))
			if len(cur) > 0 && featTitleRe.MatchString(stripped) && !strings.HasPrefix(strings.TrimSpace(cur[len(cur)-1]), "") {
			}
			cur = append(cur, strings.TrimPrefix(strings.TrimPrefix(t, ">"), " "))
		case t == "":
			// Blank line between separate top-level blockquotes ends a block
			// only when the next non-blank line starts a new title; simplest
			// robust rule: a fully blank line (not "> ") closes the block.
			flush()
		default:
			flush()
		}
	}
	flush()
	// Re-split: a single ">"-run may contain multiple titles if not separated by
	// a blank line. Split on title boundaries.
	var out []string
	for _, b := range blocks {
		out = append(out, splitOnTitles(b)...)
	}
	return out
}

// splitOnTitles splits a block whenever a new "EMOJI **Title**" line appears.
func splitOnTitles(block string) []string {
	lines := strings.Split(block, "\n")
	var blocks []string
	var cur []string
	for _, line := range lines {
		if featTitleRe.MatchString(strings.TrimSpace(line)) && len(cur) > 0 {
			blocks = append(blocks, strings.Join(cur, "\n"))
			cur = nil
		}
		cur = append(cur, line)
	}
	if len(cur) > 0 {
		blocks = append(blocks, strings.Join(cur, "\n"))
	}
	return blocks
}

// parseStatblockFeatures parses the feature blockquotes of a statblock body
// into SDK-feature maps (matching feature.schema.json shape).
func parseStatblockFeatures(body string) []map[string]any {
	var features []map[string]any
	for _, block := range splitBlockquoteBlocks(body) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		f := parseOneFeature(block)
		if f != nil {
			features = append(features, f)
		}
	}
	return features
}

func parseOneFeature(block string) map[string]any {
	lines := strings.Split(block, "\n")
	title := strings.TrimSpace(lines[0])
	m := featTitleRe.FindStringSubmatch(title)
	if m == nil {
		return nil
	}
	icon := strings.TrimSpace(m[1])
	name := strings.TrimSpace(m[2])

	f := map[string]any{
		"type":         "feature",
		"feature_type": "ability",
		"name":         name,
	}
	if icon != "" {
		f["icon"] = icon
	}

	// Parenthetical: "(Signature Ability)" → ability_type; "(N Malice)" → cost.
	if pm := parenRe.FindStringSubmatch(name); pm != nil {
		f["name"] = strings.TrimSpace(pm[1])
		paren := strings.TrimSpace(pm[2])
		if strings.EqualFold(paren, "Signature Ability") {
			f["ability_type"] = paren
		} else {
			f["cost"] = paren
		}
	}

	rest := lines[1:]
	rows := tableRows(rest)
	switch {
	case len(rows) >= 2:
		// First table = keywords | usage; second = distance | target.
		kw, usage := rows[0][0], rows[0][1]
		f["keywords"] = splitCommaList(stripBold(kw))
		f["usage"] = stripBold(usage)
		dist, target := rows[1][0], rows[1][1]
		f["distance"] = cleanIconCell(dist)
		f["target"] = cleanIconCell(target)
	case len(rows) == 1:
		f["keywords"] = splitCommaList(stripBold(rows[0][0]))
		f["usage"] = stripBold(rows[0][1])
	}

	// Effects: power-roll tiers or plain trait text.
	tiers := map[string]string{}
	var prose []string
	var roll string
	for _, line := range rest {
		t := strings.TrimSpace(line)
		if pr := powerRollRe.FindStringSubmatch(t); pr != nil {
			roll = strings.TrimSuffix(strings.TrimSpace(pr[1]), ":")
			continue
		}
		if tm := tierRe.FindStringSubmatch(t); tm != nil {
			switch {
			case strings.HasPrefix(tm[1], "≤"):
				tiers["tier1"] = strings.TrimSpace(tm[2])
			case strings.Contains(tm[1], "-"):
				tiers["tier2"] = strings.TrimSpace(tm[2])
			case strings.HasSuffix(tm[1], "+"):
				tiers["tier3"] = strings.TrimSpace(tm[2])
			}
			continue
		}
		if t == "" || strings.HasPrefix(t, "|") {
			continue
		}
		prose = append(prose, t)
	}

	if len(tiers) > 0 {
		eff := map[string]any{"roll": roll}
		for k, v := range tiers {
			eff[k] = v
		}
		f["effects"] = []map[string]any{eff}
	} else if len(prose) > 0 {
		// No power roll and no keyword/usage table → a trait.
		if _, hasUsage := f["usage"]; !hasUsage {
			f["feature_type"] = "trait"
		}
		text := strings.Join(prose, "\n")
		f["effects"] = []map[string]any{{"effect": text}}
	}

	return f
}

// tableRows extracts non-separator markdown table rows (2 cells each) from lines.
func tableRows(lines []string) [][2]string {
	var rows [][2]string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "|") || strings.Contains(t, "---") {
			continue
		}
		cells := strings.Split(strings.Trim(t, "|"), "|")
		if len(cells) >= 2 {
			rows = append(rows, [2]string{strings.TrimSpace(cells[0]), strings.TrimSpace(cells[1])})
		}
	}
	return rows
}

func stripBold(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "**", ""))
}

// cleanIconCell removes bold markers and a leading 📏/🎯 icon from a cell.
func cleanIconCell(s string) string {
	s = stripBold(s)
	for _, icon := range []string{"📏", "🎯", "🔅"} {
		s = strings.TrimSpace(strings.TrimPrefix(s, icon))
	}
	return strings.TrimSpace(s)
}
```

> The `splitBlockquoteBlocks` body above contains one defensive no-op `if` for readability of the boundary rule; the real boundary work is done by `splitOnTitles`. Keep both — `splitBlockquoteBlocks` handles blank-line-separated blocks (the common case) and `splitOnTitles` handles back-to-back titles inside one `>` run.

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestParseStatblockFeatures -v'`
Expected: PASS. If the trait/ability split or tier parsing fails, fix the helper (not the test) until it matches the legacy JSON shape.

- [ ] **Step 5: Add a cost-ability test (Dizzying Hex)**

Append to `internal/content/statblock_parse_test.go`:

```go
func TestParseStatblockFeatureCost(t *testing.T) {
	block := "" +
		"> 🏹 **Dizzying Hex (1 Malice)**\n" +
		">\n" +
		"> | **Magic, Ranged, Strike** |        **Maneuver** |\n" +
		"> |---------------------------|--------------------:|\n" +
		"> | **📏 Ranged 10**          | **🎯 One creature** |\n" +
		">\n" +
		"> **Power Roll + 2:**\n" +
		">\n" +
		"> - **≤11:** I < 0 prone\n" +
		"> - **12-16:** I < 1 prone and can't stand (EoT)\n" +
		"> - **17+:** Prone; I < 2 can't stand (save ends)\n"
	got := parseStatblockFeatures(block)
	if len(got) != 1 {
		t.Fatalf("got %d, want 1", len(got))
	}
	if got[0]["cost"] != "1 Malice" {
		t.Errorf("cost: got %v", got[0]["cost"])
	}
	if got[0]["usage"] != "Maneuver" {
		t.Errorf("usage: got %v", got[0]["usage"])
	}
}
```

- [ ] **Step 6: Run and verify**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestParseStatblockFeature -v'`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
cd steel-etl && git add internal/content/statblock_parse.go internal/content/statblock_parse_test.go && git commit -m "feat: parse statblock feature blockquotes into structured features"
```

---

## Part B — The Monster, Statblock, and Featureblock parsers

### Task B1: StatblockParser

**Files:**
- Create: `internal/content/monster.go`
- Test: `internal/content/monster_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/content/monster_test.go`:

```go
package content

import (
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// newSection builds a Section with the given heading, level, annotation, and body.
func newSection(heading string, level int, ann map[string]string, body string) *parser.Section {
	return &parser.Section{
		Heading:      heading,
		HeadingLevel: level,
		Annotation:   ann,
		BodySource:   body,
	}
}

func TestStatblockParser(t *testing.T) {
	body := cursespitterGrid + "\n" + cursespitterFeatures
	sec := newSection("Goblin Cursespitter", 7, map[string]string{"type": "statblock"}, body)

	ctx := context.NewContextStack(nil)
	ctx.Push(2, map[string]string{"category": "goblins"})

	p := &StatblockParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got.ItemID != "goblin-cursespitter" {
		t.Errorf("ItemID: got %q", got.ItemID)
	}
	wantPath := []string{"monster", "goblins", "statblock"}
	if strings.Join(got.TypePath, "/") != strings.Join(wantPath, "/") {
		t.Errorf("TypePath: got %v, want %v", got.TypePath, wantPath)
	}
	if got.Frontmatter["type"] != "statblock" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
	if got.Frontmatter["level"] != 1 {
		t.Errorf("level: got %v", got.Frontmatter["level"])
	}
	if got.Frontmatter["role"] != "Hexer" || got.Frontmatter["organization"] != "Horde" {
		t.Errorf("role/org: got %v / %v", got.Frontmatter["role"], got.Frontmatter["organization"])
	}
	if got.Frontmatter["ev"] != "3" {
		t.Errorf("ev: got %v", got.Frontmatter["ev"])
	}
	if got.Frontmatter["might"] != -2 || got.Frontmatter["intuition"] != 2 {
		t.Errorf("scores: got might=%v int=%v", got.Frontmatter["might"], got.Frontmatter["intuition"])
	}
	if got.Frontmatter["movement"] != "Climb" {
		t.Errorf("movement: got %v", got.Frontmatter["movement"])
	}
}
```

> Check the real field name for a section's raw body before running: open `internal/parser/section.go`. The struct field backing `FullBodySource()` may be named `BodySource` or similar. If `FullBodySource()` derives from children rather than a plain field, set the body via whatever field/constructor the existing parser tests use (grep `internal/content/*_test.go` for how they build a `parser.Section` with a body). Adjust `newSection` accordingly — this is the only place the helper touches parser internals.

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestStatblockParser -v'`
Expected: compile error — `StatblockParser` undefined.

- [ ] **Step 3: Implement StatblockParser**

Create `internal/content/monster.go`:

```go
package content

import (
	"strconv"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// statblockDomain returns the SCC domain root ("monster" by default) and the
// category slug from the surrounding context, set by an enclosing MonsterParser
// group or monster-group container.
func statblockDomain(ctx *context.ContextStack, level int) (domain, category string) {
	domain = "monster"
	if d, ok := ctx.Lookup(level, "domain"); ok && d != "" {
		domain = d
	}
	category, _ = ctx.Lookup(level, "category")
	return domain, category
}

// compactPath drops empty segments from a type path.
func compactPath(parts ...string) []string {
	var out []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// intField parses a numeric stat value ("+2", "-2", "0") into an int.
func intField(s string) (int, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "+", ""))
	if s == "" || s == "-" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// StatblockParser handles @type: statblock sections — individual creature stat
// blocks. Classifies as {domain}.{category}.statblock/{id}.
type StatblockParser struct{}

func (p *StatblockParser) Type() string { return "statblock" }

func (p *StatblockParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)
	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	body := section.FullBodySource()
	grid := parseStatGrid(body)

	fm := map[string]any{
		"name": name,
		"type": "statblock",
	}
	if grid.header.level > 0 {
		fm["level"] = grid.header.level
	}
	if grid.header.role != "" {
		fm["role"] = grid.header.role
	}
	if grid.header.organization != "" {
		fm["organization"] = grid.header.organization
	}
	if len(grid.header.keywords) > 0 {
		fm["keywords"] = grid.header.keywords
	}
	if grid.header.ev != "" {
		fm["ev"] = grid.header.ev
	}

	// String labels.
	for label, key := range map[string]string{
		"Stamina": "stamina", "Size": "size", "Movement": "movement",
	} {
		if v, ok := grid.labels[label]; ok && v != "-" {
			fm[key] = v
		}
	}
	// Integer labels.
	for label, key := range map[string]string{
		"Speed": "speed", "Stability": "stability", "Free Strike": "free_strike",
		"Might": "might", "Agility": "agility", "Reason": "reason",
		"Intuition": "intuition", "Presence": "presence",
	} {
		if n, ok := intField(grid.labels[label]); ok {
			fm[key] = n
		}
	}
	// Immunity / Weakness become arrays (split on comma).
	if v, ok := grid.labels["Immunity"]; ok && v != "-" {
		fm["immunities"] = splitCommaList(v)
	}
	if v, ok := grid.labels["Weakness"]; ok && v != "-" {
		fm["weaknesses"] = splitCommaList(v)
	}
	if v, ok := grid.labels["With Captain"]; ok && v != "-" {
		fm["with_captain"] = v
	}

	domain, category := statblockDomain(ctx, section.HeadingLevel)
	typePath := compactPath(domain, category, "statblock")

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestStatblockParser -v'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd steel-etl && git add internal/content/monster.go internal/content/monster_test.go && git commit -m "feat: StatblockParser classifies creatures under monster.<category>.statblock"
```

---

### Task B2: MonsterParser (group lore page + category context)

**Files:**
- Modify: `internal/content/monster.go`
- Test: `internal/content/monster_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/content/monster_test.go`:

```go
func TestMonsterParser(t *testing.T) {
	sec := newSection("Goblins", 2, map[string]string{
		"type": "monster", "category": "goblins",
	}, "Goblins are small and crafty...")

	p := &MonsterParser{}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ItemID != "goblins" {
		t.Errorf("ItemID: got %q", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "monster/goblins" {
		t.Errorf("TypePath: got %v, want [monster goblins]", got.TypePath)
	}
	if got.Frontmatter["type"] != "monster" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestMonsterParser -v'`
Expected: compile error — `MonsterParser` undefined.

- [ ] **Step 3: Implement MonsterParser**

Append to `internal/content/monster.go`:

```go
// MonsterParser handles @type: monster sections — a monster group (e.g.
// "Goblins"). It produces a lore landing page at monster/{category} AND seeds
// the `category` (and optional `domain`) context the pipeline pushes for its
// descendant statblocks and featureblocks.
type MonsterParser struct{}

func (p *MonsterParser) Type() string { return "monster" }

func (p *MonsterParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)

	category := ""
	if section.Annotation != nil {
		category = section.Annotation["category"]
	}
	if category == "" {
		category = Slugify(name)
	}

	fm := map[string]any{
		"name":     name,
		"type":     "monster",
		"category": category,
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    []string{"monster", category},
		ItemID:      category,
	}, nil
}
```

> The pipeline already pushes `section.Annotation` (which contains `category` and any `domain`) into the context stack for every annotated section, so `StatblockParser` / `FeatureblockParser` see it via `ctx.Lookup`. `MonsterParser` does not need to push anything itself.

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestMonsterParser -v'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd steel-etl && git add internal/content/monster.go internal/content/monster_test.go && git commit -m "feat: MonsterParser produces group lore page + category context"
```

---

### Task B3: FeatureblockParser (Malice and similar)

**Files:**
- Modify: `internal/content/monster.go`
- Test: `internal/content/monster_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/content/monster_test.go`:

```go
func TestFeatureblockParser(t *testing.T) {
	body := "" +
		"At the start of any goblin's turn, you can spend Malice...\n\n" +
		"> ⭐️ **Goblin Mode (3 Malice)**\n>\n> Each goblin gains +2 speed.\n"
	sec := newSection("Goblin Malice (Malice Features)", 9,
		map[string]string{"type": "featureblock"}, body)

	ctx := context.NewContextStack(nil)
	ctx.Push(2, map[string]string{"category": "goblins"})

	p := &FeatureblockParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ItemID != "goblin-malice" {
		t.Errorf("ItemID: got %q (want goblin-malice)", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "monster/goblins" {
		t.Errorf("TypePath: got %v, want [monster goblins]", got.TypePath)
	}
	if got.Frontmatter["type"] != "featureblock" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
}
```

> Note the expected `ItemID` is `goblin-malice`, not `goblin-malice-malice-features`. The parser strips a trailing parenthetical from the heading before slugifying. `CleanHeading` may already drop `(Malice Features)`; verify and, if not, strip it explicitly as shown below.

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestFeatureblockParser -v'`
Expected: compile error — `FeatureblockParser` undefined.

- [ ] **Step 3: Implement FeatureblockParser**

Append to `internal/content/monster.go` (add `"regexp"` to the import block):

```go
var trailingParenRe = regexp.MustCompile(`\s*\([^)]*\)\s*$`)

// FeatureblockParser handles @type: featureblock sections — a named block of
// malice/tactical features attached to a monster group (e.g. "Goblin Malice").
// Classifies as {domain}.{category}/{id}, a sibling of the statblock/ folder.
type FeatureblockParser struct{}

func (p *FeatureblockParser) Type() string { return "featureblock" }

func (p *FeatureblockParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)
	name = strings.TrimSpace(trailingParenRe.ReplaceAllString(name, ""))

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	fm := map[string]any{
		"name": name,
		"type": "featureblock",
	}

	domain, category := statblockDomain(ctx, section.HeadingLevel)
	typePath := compactPath(domain, category)

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}
```

> Add `import "regexp"` to `monster.go` if it is not already imported (it is not, in the Task B1 version). Keep imports grouped.

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestFeatureblockParser -v'`
Expected: PASS. If `ItemID` comes out as `goblin-malice-malice-features`, `CleanHeading` did not strip the parenthetical — the explicit `trailingParenRe` strip above handles it; re-run.

- [ ] **Step 5: Run the full content package test suite**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -v 2>&1 | tail -20'`
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
cd steel-etl && git add internal/content/monster.go internal/content/monster_test.go && git commit -m "feat: FeatureblockParser classifies Malice as monster.<category>/<id>"
```

---

## Part C — Dynamic-terrain parser

### Task C1: DynamicTerrainParser

**Files:**
- Create: `internal/content/dynamic_terrain.go`
- Test: `internal/content/dynamic_terrain_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/content/dynamic_terrain_test.go`:

```go
package content

import (
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
)

func TestDynamicTerrainParser(t *testing.T) {
	body := "" +
		"This beehive is full of angry bees.\n\n" +
		"- **EV:** 2\n- **Stamina:** 3\n- **Size:** 1S\n\n" +
		"> 🌀 **Deactivate**\n>\n> The beehive can't be deactivated.\n"
	sec := newSection("Angry Beehive (Level 2 Hazard Hexer)", 9,
		map[string]string{"type": "dynamic-terrain"}, body)

	ctx := context.NewContextStack(nil)
	ctx.Push(1, map[string]string{"domain": "dynamic-terrain", "category": "environmental-hazards"})

	p := &DynamicTerrainParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ItemID != "angry-beehive" {
		t.Errorf("ItemID: got %q", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "dynamic-terrain/environmental-hazards" {
		t.Errorf("TypePath: got %v", got.TypePath)
	}
	if got.Frontmatter["ev"] != "2" || got.Frontmatter["stamina"] != "3" || got.Frontmatter["size"] != "1S" {
		t.Errorf("stats: got ev=%v stamina=%v size=%v",
			got.Frontmatter["ev"], got.Frontmatter["stamina"], got.Frontmatter["size"])
	}
	if got.Frontmatter["level"] != "2" {
		t.Errorf("level: got %v", got.Frontmatter["level"])
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestDynamicTerrainParser -v'`
Expected: compile error — `DynamicTerrainParser` undefined.

- [ ] **Step 3: Implement DynamicTerrainParser**

Create `internal/content/dynamic_terrain.go`:

```go
package content

import (
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

var (
	// "- **EV:** 2" style list fields.
	terrainFieldRe = regexp.MustCompile(`(?m)^-\s*\*\*([A-Za-z ]+):\*\*\s*(.+)$`)
	// "(Level 2 Hazard Hexer)" trailing classifier in the heading.
	terrainLevelRe = regexp.MustCompile(`Level\s+(\d+)`)
)

// DynamicTerrainParser handles @type: dynamic-terrain sections — terrain
// objects (hazards, fieldworks, mechanisms, fixtures). Classifies as
// {domain}.{category}/{id} where domain defaults to "dynamic-terrain".
type DynamicTerrainParser struct{}

func (p *DynamicTerrainParser) Type() string { return "dynamic-terrain" }

func (p *DynamicTerrainParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)
	name = strings.TrimSpace(trailingParenRe.ReplaceAllString(name, ""))

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	body := section.FullBodySource()
	fm := map[string]any{
		"name": name,
		"type": "dynamic-terrain",
	}

	if m := terrainLevelRe.FindStringSubmatch(section.Heading); m != nil {
		fm["level"] = m[1]
	}
	for _, m := range terrainFieldRe.FindAllStringSubmatch(body, -1) {
		key := strings.ToLower(strings.TrimSpace(m[1]))
		key = strings.ReplaceAll(key, " ", "_")
		fm[key] = strings.TrimSpace(m[2])
	}

	domain := "dynamic-terrain"
	if d, ok := ctx.Lookup(section.HeadingLevel, "domain"); ok && d != "" {
		domain = d
	}
	category, _ := ctx.Lookup(section.HeadingLevel, "category")

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    compactPath(domain, category),
		ItemID:      id,
	}, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestDynamicTerrainParser -v'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd steel-etl && git add internal/content/dynamic_terrain.go internal/content/dynamic_terrain_test.go && git commit -m "feat: DynamicTerrainParser for hazards/fieldworks/mechanisms"
```

---

## Part D — Register parsers

### Task D1: Register the four new parsers

**Files:**
- Modify: `internal/content/registry.go`
- Test: `internal/content/content_test.go` (or wherever registry tests live)

- [ ] **Step 1: Write the failing test**

Append a test to `internal/content/content_test.go`:

```go
func TestMonsterParsersRegistered(t *testing.T) {
	r := NewRegistry()
	for _, typeName := range []string{"monster", "statblock", "featureblock", "dynamic-terrain"} {
		if !r.Has(typeName) {
			t.Errorf("parser %q not registered", typeName)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestMonsterParsersRegistered -v'`
Expected: FAIL — parsers not registered.

- [ ] **Step 3: Register the parsers**

In `internal/content/registry.go`, after the Phase 3 parsers block, add:

```go
	// Phase 5 parsers (Monsters book)
	r.Register(&MonsterParser{})
	r.Register(&StatblockParser{})
	r.Register(&FeatureblockParser{})
	r.Register(&DynamicTerrainParser{})
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestMonsterParsersRegistered -v'`
Expected: PASS.

- [ ] **Step 5: Build + full content suite**

Run: `devbox run -- bash -c 'cd steel-etl && go build ./... && go test ./internal/content/ 2>&1 | tail -5'`
Expected: builds, all pass.

- [ ] **Step 6: Commit**

```bash
cd steel-etl && git add internal/content/registry.go internal/content/content_test.go && git commit -m "feat: register monster/statblock/featureblock/dynamic-terrain parsers"
```

---

## Part E — Statblock SDK transform + schema

### Task E1: transformStatblock + schema

**Files:**
- Create: `internal/output/statblock_transform.go`
- Modify: `internal/output/sdk_transform.go`
- Create: `schemas/statblock.schema.json`
- Test: `internal/output/statblock_transform_test.go`

- [ ] **Step 1: Copy the SDK statblock schema into the repo**

```bash
cp data-sdk-npm/src/schema/statblock.schema.json steel-etl/schemas/statblock.schema.json
```

> The `$ref: "feature.schema.json-3.0.0#"` inside it must resolve the same way the existing kit/treasure schemas resolve their `$ref`s. Check how `internal/output/schema_validation_test.go` loads sibling schemas (it already validates `feature.schema.json`); reuse that loader so the `features[]` `$ref` resolves.

- [ ] **Step 2: Write the failing conformance test**

Create `internal/output/statblock_transform_test.go`:

```go
package output

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

func TestTransformStatblock(t *testing.T) {
	body := "" + // grid + features, abbreviated; full fixture added in Step 4
		"| Goblin, Humanoid | - | Level 1 | Horde Hexer | EV 3 |\n" +
		"|:--:|:--:|:--:|:--:|:--:|\n" +
		"| **1S**<br>Size | **5**<br>Speed | **10**<br>Stamina | **0**<br>Stability | **1**<br>Free Strike |\n" +
		"| **-**<br>Immunity | **Climb**<br>Movement | - | **-**<br>With Captain | **-**<br>Weakness |\n" +
		"| **-2**<br>Might | **+1**<br>Agility | **0**<br>Reason | **+2**<br>Intuition | **0**<br>Presence |\n\n" +
		"> ⭐️ **Crafty**\n>\n> Doesn't provoke opportunity attacks by moving.\n"

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Goblin Cursespitter", "type": "statblock", "level": 1,
			"role": "Hexer", "organization": "Horde", "ev": "3",
			"keywords": []string{"Goblin", "Humanoid"}, "stamina": "10",
			"speed": 5, "size": "1S", "stability": 0, "free_strike": 1,
			"might": -2, "agility": 1, "reason": 0, "intuition": 2, "presence": 0,
			"movement": "Climb",
		},
		Body: body,
	}

	out := TransformToSDKFormat("mcdm.monsters.v1/monster.goblins.statblock/goblin-cursespitter", parsed)

	if out["type"] != "statblock" || out["name"] != "Goblin Cursespitter" {
		t.Fatalf("base fields wrong: %+v", out)
	}
	if out["role"] != "Hexer" || out["organization"] != "Horde" {
		t.Errorf("role/org: %v / %v", out["role"], out["organization"])
	}
	feats, ok := out["features"].([]map[string]any)
	if !ok || len(feats) != 1 {
		t.Fatalf("features: got %v", out["features"])
	}
	if feats[0]["name"] != "Crafty" || feats[0]["feature_type"] != "trait" {
		t.Errorf("feature: %+v", feats[0])
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/output/ -run TestTransformStatblock -v'`
Expected: FAIL — `transformPassthrough` returns no `features`/`role` handling (statblock falls through to passthrough).

- [ ] **Step 4: Implement transformStatblock**

Create `internal/output/statblock_transform.go`:

```go
package output

import (
	"github.com/SteelCompendium/steel-etl/internal/content"
)

// statblockScalarKeys are frontmatter fields copied straight into SDK output.
var statblockScalarKeys = []string{
	"name", "type", "level", "role", "organization", "keywords", "ev",
	"stamina", "speed", "movement", "size", "stability", "free_strike",
	"might", "agility", "reason", "intuition", "presence",
	"immunities", "weaknesses", "with_captain",
}

// transformStatblock builds an SDK statblock object: scalar stats from the
// parsed frontmatter plus a features[] array parsed from the body blockquotes.
func transformStatblock(sccCode string, parsed *content.ParsedContent) map[string]any {
	out := map[string]any{}
	for _, key := range statblockScalarKeys {
		if v, ok := parsed.Frontmatter[key]; ok {
			out[key] = v
		}
	}
	out["type"] = "statblock"

	// Schema requires these even when absent in the source; default them.
	defaults := map[string]any{
		"role": "", "organization": "", "keywords": []string{},
		"ev": "", "stamina": "", "level": 0,
		"speed": 0, "size": "", "stability": 0, "free_strike": 0,
		"might": 0, "agility": 0, "reason": 0, "intuition": 0, "presence": 0,
	}
	for k, dv := range defaults {
		if _, ok := out[k]; !ok {
			out[k] = dv
		}
	}

	if feats := content.ParseStatblockFeatures(parsed.Body); len(feats) > 0 {
		out["features"] = feats
	}

	out["metadata"] = map[string]any{"scc": sccCode, "source": sccSource(sccCode)}
	return out
}
```

> `sccSource` may already exist in the output package (used by `buildAbilityMetadata`). If not, inline `strings.SplitN(sccCode, "/", 2)[0]`. Check before adding a duplicate.

- [ ] **Step 5: Export `parseStatblockFeatures` for the output package**

In `internal/content/statblock_parse.go`, rename the function to exported `ParseStatblockFeatures` (and update the content-package tests + any internal callers). Add a thin internal alias if you prefer to keep the lowercase name in tests:

```go
// ParseStatblockFeatures parses the feature blockquotes of a statblock body
// into SDK-feature maps (matching feature.schema.json shape).
func ParseStatblockFeatures(body string) []map[string]any {
	// ...existing body, formerly parseStatblockFeatures...
}
```

Update `internal/content/statblock_parse_test.go` to call `ParseStatblockFeatures`.

- [ ] **Step 6: Wire the dispatch**

In `internal/output/sdk_transform.go`, add to the `switch contentType` in `TransformToSDKFormat`:

```go
	case "statblock":
		return transformStatblock(sccCode, parsed)
```

- [ ] **Step 7: Run to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/output/ -run TestTransformStatblock -v'`
Expected: PASS.

- [ ] **Step 8: Add schema validation for statblock output**

In `internal/output/schema_validation_test.go`, follow the existing pattern (a table of `{type, fixture, schema}`) to add a `statblock` case validating the `TestTransformStatblock` output against `schemas/statblock.schema.json`. Mirror exactly how the `kit`/`treasure` cases are wired (loader, `$ref` resolution).

- [ ] **Step 9: Run to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/output/ -run "Statblock|Schema" -v 2>&1 | tail -20'`
Expected: PASS. If the `features[]` `$ref` fails to resolve, fix the schema loader registration (register `feature.schema.json` before validating statblock), not the data.

- [ ] **Step 10: Commit**

```bash
cd steel-etl && git add internal/output/statblock_transform.go internal/output/sdk_transform.go internal/output/statblock_transform_test.go internal/output/schema_validation_test.go internal/content/statblock_parse.go internal/content/statblock_parse_test.go schemas/statblock.schema.json && git commit -m "feat: statblock SDK transform + schema validation"
```

---

## Part F — Copy input into steel-etl + wire the pipeline

### Task F1: Move the source into the repo and point the pipeline at it

**Files:**
- Create: `input/monsters/Draw Steel Monsters.md`
- Modify: `pipeline.yaml`

- [ ] **Step 1: Copy the source document into steel-etl**

```bash
mkdir -p steel-etl/input/monsters
cp "data-gen/input/monsters/Draw Steel Monsters.md" "steel-etl/input/monsters/Draw Steel Monsters.md"
```

> This is the new canonical, hand-annotated source (parallel to `input/heroes/` and `input/beastheart/`). The `data-gen` copy becomes legacy reference only.

- [ ] **Step 2: Update pipeline.yaml to point at the in-repo input**

In `steel-etl/pipeline.yaml`, change the monsters book entry from:

```yaml
  - book: mcdm.monsters.v1
    input: ../data-gen/input/monsters/Draw Steel Monsters.md
    output:
      base_dir: ../data/data-bestiary
```

to:

```yaml
  - book: mcdm.monsters.v1
    input: ./input/monsters/Draw Steel Monsters.md
    output:
      base_dir: ../data/data-bestiary
```

- [ ] **Step 3: Add the document frontmatter to the source**

At the very top of `steel-etl/input/monsters/Draw Steel Monsters.md`, ensure this frontmatter block exists (matching the beastheart pattern):

```markdown
---
book: mcdm.monsters.v1
source: MCDM
title: Draw Steel Monsters
---
```

- [ ] **Step 4: Smoke-test the book parses (no annotations yet → all skipped)**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --book mcdm.monsters.v1 2>&1 | tail -20'`
Expected: completes without crashing. Most sections report "skipped" (no annotations yet). No panics.

- [ ] **Step 5: Commit (without the large generated output)**

```bash
cd steel-etl && git add pipeline.yaml "input/monsters/Draw Steel Monsters.md" && git commit -m "chore: bring Monsters source into steel-etl/input, point pipeline at it"
```

> `data/` is gitignored build output — nothing under `data/data-bestiary/` is committed here.

---

## Part G — Annotate one group end-to-end (Goblins) and validate

This is the validation loop. Annotate **only the Goblins group first**, run the pipeline, and diff against legacy output. Fix parser bugs surfaced here before annotating the rest. The user chose hand annotation; do it carefully, section by section.

### Annotation conventions (apply throughout Parts G–H)

Place each annotation on its own line **immediately before** the heading it describes:

| Source heading | Annotation |
|----------------|-----------|
| `# Monsters` | `<!-- @type: chapter | @id: monsters -->` |
| `## Goblins` (a monster group) | `<!-- @type: monster | @category: goblins -->` |
| `####### Goblin Cursespitter` (statblock) | `<!-- @type: statblock -->` |
| `######### Goblin Malice (Malice Features)` | `<!-- @type: featureblock -->` |
| `# Dynamic Terrain` | `<!-- @type: chapter | @id: dynamic-terrain -->` |
| `### Environmental Hazards` (terrain category) | `<!-- @type: monster-group | @domain: dynamic-terrain | @category: environmental-hazards -->` |
| `######### Angry Beehive (Level 2 Hazard Hexer)` | `<!-- @type: dynamic-terrain -->` |
| `# Retainers` | `<!-- @type: chapter | @id: retainers -->` |
| Retainer creature `####### Goblin Guide` | `<!-- @type: statblock -->` under a `monster-group` with `@domain: retainer` |

`@category` slugs use lowercase hyphenated forms (`elves-high`, `count-rhodar-von-glauer`). For groups whose heading slugifies poorly, set `@category` explicitly.

> **`monster-group` container:** the terrain categories and the retainer statblock group need a non-code-producing container that only seeds `domain`/`category` context. Reuse the existing `feature-group` pattern by adding a tiny `MonsterGroupParser` — OR, simpler, set `@domain`/`@category` directly on the enclosing `@type: chapter`/`### ` heading and let the leaf parsers read them from context. **Decision for this plan:** add a `MonsterGroupParser` (Task G0) so terrain/retainer grouping is explicit and produces no stray pages.

### Task G0: Add the non-code MonsterGroupParser container

**Files:**
- Modify: `internal/content/monster.go`, `internal/content/registry.go`
- Test: `internal/content/monster_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/content/monster_test.go`:

```go
func TestMonsterGroupParser(t *testing.T) {
	sec := newSection("Environmental Hazards", 3, map[string]string{
		"type": "monster-group", "domain": "dynamic-terrain", "category": "environmental-hazards",
	}, "intro prose")
	p := &MonsterGroupParser{}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Container: no output file.
	if got.TypePath != nil || got.ItemID != "" {
		t.Errorf("expected no classification, got TypePath=%v ItemID=%q", got.TypePath, got.ItemID)
	}
	if got.Frontmatter["type"] != "monster-group" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestMonsterGroupParser -v'`
Expected: compile error — `MonsterGroupParser` undefined.

- [ ] **Step 3: Implement MonsterGroupParser**

Append to `internal/content/monster.go`:

```go
// MonsterGroupParser handles @type: monster-group — a non-code-producing
// container (like feature-group/treasure-group) that seeds `domain` and
// `category` context for descendant statblocks/terrain. Produces no file.
type MonsterGroupParser struct{}

func (p *MonsterGroupParser) Type() string { return "monster-group" }

func (p *MonsterGroupParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	fm := map[string]any{
		"name": CleanHeading(section.Heading),
		"type": "monster-group",
	}
	if section.Annotation != nil {
		for _, k := range []string{"domain", "category"} {
			if v, ok := section.Annotation[k]; ok {
				fm[k] = v
			}
		}
	}
	return &ParsedContent{Frontmatter: fm, Body: section.FullBodySource()}, nil
}
```

- [ ] **Step 4: Register it**

In `internal/content/registry.go`, add to the Phase 5 block:

```go
	r.Register(&MonsterGroupParser{})
```

- [ ] **Step 5: Run to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run "TestMonsterGroupParser|TestMonsterParsersRegistered" -v'`
Expected: PASS (also update `TestMonsterParsersRegistered` to include `"monster-group"`).

- [ ] **Step 6: Commit**

```bash
cd steel-etl && git add internal/content/monster.go internal/content/registry.go internal/content/monster_test.go && git commit -m "feat: MonsterGroupParser container for terrain/retainer grouping"
```

### Task G1: Annotate the Goblins group

**Files:**
- Modify: `input/monsters/Draw Steel Monsters.md` (the `## Goblins` section only, ~lines 11360–11875 in the original)

- [ ] **Step 1: Annotate the group, its statblocks, and its Malice block**

In the Goblins section:
1. Add `<!-- @type: chapter | @id: monsters -->` before `# Monsters` (line ~1348) if not already present.
2. Add `<!-- @type: monster | @category: goblins -->` before `## Goblins`.
3. Add `<!-- @type: featureblock -->` before `######### Goblin Malice (Malice Features)`.
4. Add `<!-- @type: statblock -->` before each H7 statblock: Goblin Runner, Goblin Sniper, Goblin Spinecleaver, Skitterling, Goblin Assassin, Goblin Cursespitter, Goblin Stinker, Goblin Underboss, Goblin Warrior, Goblin Monarch, War Spider, Worg.

Leave the intervening prose H4 sections (`#### Mobile and Sneaky`, etc.) unannotated — they become part of the group lore page body via `MonsterParser` / are skipped.

- [ ] **Step 2: Run the pipeline for just the monsters book**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --book mcdm.monsters.v1 2>&1 | tail -30'`
Expected: ~13 sections parsed/classified for Goblins (1 monster + 12 statblocks + 1 featureblock), no errors, no duplicate-SCC warnings.

- [ ] **Step 3: Inspect the generated SCC codes and paths**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl classify "input/monsters/Draw Steel Monsters.md" --config pipeline.yaml 2>&1 | grep -i goblin'`
Expected codes include:
- `mcdm.monsters.v1/monster.goblins/goblins` (group lore page)
- `mcdm.monsters.v1/monster.goblins.statblock/goblin-cursespitter`
- `mcdm.monsters.v1/monster.goblins/goblin-malice`

- [ ] **Step 4: Verify the file hierarchy matches the chosen shape**

Run: `find data/data-bestiary/en/md -path '*goblins*' | sort`
Expected:
```
data/data-bestiary/en/md/monster/goblins/goblins.md          (group lore page)
data/data-bestiary/en/md/monster/goblins/goblin-malice.md    (featureblock sibling)
data/data-bestiary/en/md/monster/goblins/statblock/goblin-cursespitter.md
data/data-bestiary/en/md/monster/goblins/statblock/...
```

- [ ] **Step 5: Diff the structured JSON against legacy output**

Run:
```bash
devbox run -- bash -c '
  diff <(jq -S . "data/data-bestiary/en/json/monster/goblins/statblock/goblin-cursespitter.json") \
       <(jq -S "del(.metadata, .roles, .organization) | .role //= \"Hexer\"" \
            "data/data-bestiary-json/Monsters/Goblins/Statblocks/Goblin Cursespitter.json") || true
'
```
Expected: differences are only the intended schema changes (legacy `roles:["Horde Hexer"]` → new `role:"Hexer"` + `organization:"Horde"`, plus the new `metadata.scc`). The `features[]` arrays should match field-for-field (name, icon, keywords, usage, distance, target, effects/tiers). **If `features[]` differ, fix the parser helpers (Part A) and re-run** — this is the key correctness gate.

- [ ] **Step 6: Spot-check the markdown + DSE outputs render**

Run: `sed -n '1,40p' data/data-bestiary/en/md/monster/goblins/statblock/goblin-cursespitter.md`
Expected: YAML frontmatter with the structured stats, followed by the grid + ability blockquotes (book-faithful body).

- [ ] **Step 7: Commit the Goblins annotations**

```bash
cd steel-etl && git add "input/monsters/Draw Steel Monsters.md" && git commit -m "content: annotate Goblins monster group (validation slice)"
```

---

## Part H — Annotate the rest of the book

Repeat the Task G1 annotation pattern across the whole document, in batches, committing per batch and re-running the pipeline after each to catch edge cases early. Each batch is small and independently verifiable.

### Task H1: Annotate all remaining monster groups (~54 groups)

**Files:** `input/monsters/Draw Steel Monsters.md`

- [ ] **Step 1: Annotate named solo/villain groups first (single-statblock groups)**

These `##` groups contain one statblock and sometimes extra featureblocks (Ajax has `Ajaxs Malice` + `Tactical Stance`): Ajax the Invincible, Arixx, Ashen Hoarder, Bredbeddle, Chimera, Fossil Cryptic, Hag, Kingfissure Worm, Manticore, Medusa, Olothec, Shambling Mound, Count Rhodar Von Glauer, Lich, Valok, Lord Syuul, Werewolf, Xorannox the Tyract.

For each: `<!-- @type: monster | @category: <slug> -->` on the `##` heading, `<!-- @type: statblock -->` on each H7, `<!-- @type: featureblock -->` on each H9 (e.g. Ajax's `Tactical Stance` and `Ajaxs Malice` both become featureblocks — siblings of the `statblock/` folder, exactly the layout you requested).

- [ ] **Step 2: Run + classify after this batch**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --book mcdm.monsters.v1 2>&1 | tail -15'`
Expected: no duplicate-SCC errors. Confirm Ajax produces `monster/ajax-the-invincible/tactical-stance` and `monster/ajax-the-invincible/ajaxs-malice` as siblings of `monster/ajax-the-invincible/statblock/`.

- [ ] **Step 3: Commit**

```bash
cd steel-etl && git add "input/monsters/Draw Steel Monsters.md" && git commit -m "content: annotate named solo/villain monster groups"
```

- [ ] **Step 4: Annotate the multi-statblock groups (the bulk)**

Annotate the remaining `##` groups (Angulotls, Animals, Basilisks, Bugbears, Demons, Devils, Draconians, Dragons, Dwarves, Elementals, Elves High/Shadow/Wode, Giants, Gnolls, Griffons, Hobgoblins, Humans, Kobolds, Lightbenders, Lizardfolk, Minotaurs, Ogres, Orcs, Radenwights, Rivals, Time Raiders, Trolls, Undead, Voiceless Talkers, War Dogs, Wyverns, …). Work top to bottom. Each: `@type: monster` on the group, `@type: statblock` on every H7, `@type: featureblock` on every H9.

> **Rivals** (`## Rivals`, line ~16874) is large and may have an extra `###` sub-grouping (e.g. by echelon). If sub-headings group statblocks, leave them unannotated (prose) — the statblocks still inherit `category: rivals` from the `## Rivals` context. Only add a nested `monster-group` if you want a deeper path.

- [ ] **Step 5: Run after every ~10 groups and fix issues**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --book mcdm.monsters.v1 2>&1 | tail -15'`
Expected: parsed-section count climbs; no errors. Watch for: statblocks whose grid has extra/missing rows, multi-line keyword cells, or unusual role cells — fix the Part A helpers if a real statblock doesn't parse, then re-run.

- [ ] **Step 6: Commit in batches**

```bash
cd steel-etl && git add "input/monsters/Draw Steel Monsters.md" && git commit -m "content: annotate monster groups <range>"
```

### Task H2: Annotate Dynamic Terrain

**Files:** `input/monsters/Draw Steel Monsters.md`

- [ ] **Step 1: Annotate the chapter, categories, and terrain objects**

1. `<!-- @type: chapter | @id: dynamic-terrain -->` before `# Dynamic Terrain`.
2. Before each `###` category (`Environmental Hazards`, `Fieldworks`, `Mechanisms`, `Power Fixtures`, `Siege Engines`): `<!-- @type: monster-group | @domain: dynamic-terrain | @category: <slug> -->`.
3. Before each `#########` terrain object: `<!-- @type: dynamic-terrain -->`.

Leave the `### Terrain Object Stat Blocks` rules sub-section and its `#### EV/Stamina/...` explainer headings unannotated (they are rules prose, not objects).

- [ ] **Step 2: Run + verify paths**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --book mcdm.monsters.v1 2>&1 | tail -10' && find data/data-bestiary/en/md/dynamic-terrain -type f | head`
Expected: files like `dynamic-terrain/environmental-hazards/angry-beehive.md`.

- [ ] **Step 3: Commit**

```bash
cd steel-etl && git add "input/monsters/Draw Steel Monsters.md" && git commit -m "content: annotate Dynamic Terrain"
```

### Task H3: Annotate Retainers

**Files:** `input/monsters/Draw Steel Monsters.md`

- [ ] **Step 1: Annotate the chapter and retainer statblocks**

1. `<!-- @type: chapter | @id: retainers -->` before `# Retainers`.
2. Add a container before the first retainer statblock grouping: `<!-- @type: monster-group | @domain: retainer | @category: -->` placed on the heading that introduces the retainer creatures (the `#####`/`####` heading just above the first H7 retainer, around line ~27600). With empty category, statblocks classify as `retainer/statblock/<id>`.
3. Before each retainer creature H7 (Goblin Guide, Human Warrior, Undead Servitor, …): `<!-- @type: statblock -->`.

Leave the retainer **rules** (advancement, role advancement abilities at H8) unannotated — they are rules prose, mirroring the legacy output which only extracted retainer statblocks.

> If `@category: -->` (empty value) is awkward for the annotation scanner, use `@domain: retainer` only and confirm `statblockDomain` yields `category=""` → `compactPath("retainer","","statblock")` → `["retainer","statblock"]`. Verify with the classify command in Step 2.

- [ ] **Step 2: Run + verify**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --book mcdm.monsters.v1 2>&1 | tail -10' && find data/data-bestiary/en/md/retainer -type f | head`
Expected: `retainer/statblock/goblin-guide.md`, etc.

- [ ] **Step 3: Annotate Monster Basics (the intro chapter)**

The `# Monster Basics` chapter (keywords, malice rules, etc.) is reference prose. Annotate `# Monster Basics` as `<!-- @type: chapter | @id: monster-basics -->` so it renders as a Read-tab chapter; leave its sub-sections unannotated.

- [ ] **Step 4: Full book run + validate annotation coverage**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate "input/monsters/Draw Steel Monsters.md" --config pipeline.yaml 2>&1 | tail -20'`
Expected: high annotation coverage on the intended sections; **no "unknown @type"** errors. Review the unannotated list — it should be only prose/rules sub-headings.

- [ ] **Step 5: Full deploy-style multi-book gen (sanity)**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all 2>&1 | tail -25'`
Expected: heroes + beastheart + monsters all generate; no duplicate-SCC errors across books.

- [ ] **Step 6: Commit**

```bash
cd steel-etl && git add "input/monsters/Draw Steel Monsters.md" && git commit -m "content: annotate Retainers + Monster Basics; full book annotated"
```

---

## Part I — Website: the Bestiary tab

### Task I1: Add the Bestiary section to site.yaml

**Files:**
- Modify: `../v2/site.yaml`

- [ ] **Step 1: Add a Bestiary section and Read inclusion**

In `v2/site.yaml`, replace the commented-out Bestiary block with a real section. Add it after the `Browse` section and before `Read`:

```yaml
  # Bestiary: monsters, terrain, and retainers (Monsters book)
  - name: Bestiary
    include:
      - monster/
      - dynamic-terrain/
      - retainer/
    sort: natural
```

And add `monster.basics`/monster chapters to the `Read` grouping by ensuring the monsters book chapters (already `chapter/…` with `group_by_book: true`) flow into `Read/bestiary/` via the existing `books:` entry (`key: mcdm.monsters.v1`, `folder: bestiary`). No change needed there — it is already configured.

- [ ] **Step 2: Build the site locally**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml 2>&1 | tail -20'`
Expected: completes; `v2/docs/Bestiary/` is created with `monster/`, `dynamic-terrain/`, `retainer/` subtrees.

- [ ] **Step 3: Verify the Browse hierarchy matches your preference**

Run: `find v2/docs/Bestiary/monster/goblins -type f | sort`
Expected: `goblins.md` (group page) plus `goblin-malice.md` as a **sibling** of the `statblock/` folder, and statblocks under `statblock/`. Confirm there is no separate `Features/` folder.

- [ ] **Step 4: Check the generated Bestiary index pages**

Run: `sed -n '1,30p' v2/docs/Bestiary/monster/index.md 2>/dev/null; echo '---'; sed -n '1,30p' v2/docs/Bestiary/index.md 2>/dev/null`
Expected: navigable index listing monster categories. If the index cards look wrong or the dragon/skull crest is missing, see Task I2.

- [ ] **Step 5: Commit**

```bash
cd ../v2 && git add site.yaml && git commit -m "feat: add Bestiary tab (monsters, terrain, retainers)"
```

### Task I2: Bestiary index cards + icons (only if Task I1 Step 4 looks unstyled)

**Files:**
- Modify: `steel-etl/internal/site/cards.go` (and `cards_test.go`)

- [ ] **Step 1: Decide if work is needed**

If the Bestiary type-index pages already render as `.sc-card` grids (like Browse), skip this task. If they render as plain bullet lists, add `monster`/`statblock`/`dynamic-terrain`/`retainer` to the card-producing type set in `cards.go`, mirroring how `kit`/`ancestry`/`class` indexes get cards. Use the existing `skull` glyph (there is no dragon glyph in the MDI free set — see the `iconPaths` note in `site.yaml`).

- [ ] **Step 2: Add a focused test + implementation**

Follow the existing `cards_test.go` table pattern: add a case asserting a monster-type index produces a `.sc-card` block with the `skull` crest. Implement minimally to pass.

- [ ] **Step 3: Run tests**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -v 2>&1 | tail -20'`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
cd steel-etl && git add internal/site/cards.go internal/site/cards_test.go && git commit -m "feat: bestiary index cards"
```

---

## Part J — Docs, registry freeze, and final verification

### Task J1: Update documentation

**Files:**
- Modify: `ANNOTATION-GUIDE.md`, `plans/architecture-redesign/scc-taxonomy.md`, `CLAUDE.md`, `../ARCHITECTURE.md`, `../CLAUDE.md`

- [ ] **Step 1: Promote the Monsters annotation types from "future" to current**

In `steel-etl/ANNOTATION-GUIDE.md`, replace the "Monsters (future)" table with the real types and add a worked example (Goblins group → statblock → malice), documenting the `@category`/`@domain` keys, the `monster-group` container, and the resulting SCC shapes.

- [ ] **Step 2: Update the SCC taxonomy doc to the nested shape**

In `plans/architecture-redesign/scc-taxonomy.md`, update the Monsters Book Types table to the implemented design:
- `monster` group lore → `mcdm.monsters.v1/monster.<category>/<category>` (path `monster/<category>/<category>.md`)
- statblock → `mcdm.monsters.v1/monster.<category>.statblock/<id>`
- featureblock (Malice) → `mcdm.monsters.v1/monster.<category>/<id>`
- dynamic-terrain → `mcdm.monsters.v1/dynamic-terrain.<category>/<id>`
- retainer statblock → `mcdm.monsters.v1/retainer.statblock/<id>`

- [ ] **Step 3: Update CLAUDE.md / ARCHITECTURE.md**

- `steel-etl/CLAUDE.md`: bump the parser count (registry now has the 5 new parsers) and add `input/monsters/` to the source-of-truth list.
- `../CLAUDE.md`: update the SCC registry count (run `go run ./cmd/steel-etl classify --config pipeline.yaml ...` to get the new total) and note the monster hierarchy alongside the treasure-hierarchy note.
- `../ARCHITECTURE.md`: note the Bestiary output target and that monsters flow to `data/data-bestiary/`.

- [ ] **Step 4: Commit**

```bash
cd steel-etl && git add ANNOTATION-GUIDE.md CLAUDE.md plans/architecture-redesign/scc-taxonomy.md && cd .. && git add CLAUDE.md ARCHITECTURE.md && git commit -m "docs: document monster pipeline + nested SCC taxonomy"
```

### Task J2: Final full-suite verification

- [ ] **Step 1: Full test suite with race + coverage**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./... -race -cover 2>&1 | tail -30'`
Expected: all pass; content + output packages keep ≥80% coverage.

- [ ] **Step 2: Full multi-book gen**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all 2>&1 | tail -20'`
Expected: no errors, no duplicate SCC across heroes/beastheart/monsters.

- [ ] **Step 3: SCC stability check**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate --scc-stable --config pipeline.yaml 2>&1 | tail -10'`
Expected: stable. Once satisfied with the monster codes, set `classification.freeze: true` in `pipeline.yaml` to freeze the new codes (matching how heroes codes are frozen) and commit.

- [ ] **Step 4: Build the site end to end**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml 2>&1 | tail -10'`
Expected: Bestiary tab populated. (Do not deploy — the user deploys.)

- [ ] **Step 5: Final bump commit**

```bash
cd /home/vexa/code/steel_compendium/workspace && git add steel-etl && git commit -m "chore: bump steel-etl (Monsters book live: statblocks, terrain, retainers, Bestiary tab)"
```

---

## Self-Review Notes (for the implementer)

- **Correctness gate is Part G Step 5** (Goblins JSON diff vs. legacy). Do not proceed to Part H until the `features[]` arrays match the legacy output field-for-field. Every statblock in the book uses the same blockquote grammar, so a parser bug found here is a bug everywhere.
- **Frozen SCC:** the hierarchy shape (`monster.<category>.statblock/<id>`) is locked once `freeze: true`. The Goblins slice validates it before you annotate 437 statblocks. The group lore page lives at `monster/<category>/<category>.md` — inside the category dir, alongside the Malice featureblock (per the user's decision), so there is no `goblins.md`-next-to-`goblins/` coexistence to worry about.
- **Edge cases to expect during annotation:** statblocks with multi-line keyword cells, minions with shared "with captain" text, villain statblocks with extra villain-action blockquotes (👤/☠️ icons — they parse as abilities, which is correct), and Rivals' nested structure. Fix the Part A helpers (with a new fixture test) when a real statblock breaks parsing; never hand-edit generated output.
- **`parser.Section` body field:** Task B1 Step 1 flags that the test helper's body field name must match `internal/parser/section.go`. Verify once at the start of Part B; it affects every parser test.
