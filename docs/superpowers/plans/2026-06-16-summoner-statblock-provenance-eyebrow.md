# Summoner-book Statblock Provenance Eyebrow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the useless `sb__kw` eyebrow on summoner-book creature statblocks with a provenance label derived from the page's SCC code (e.g. "Rival Summoner Summon · Echelon 4", "Summoner Minion · Undead").

**Architecture:** A new pure function `summonerProvenanceEyebrow(scc string) string` (new file `internal/site/summoner_provenance.go`) maps a summoner-book statblock SCC code to its eyebrow label, returning `""` for anything else. `buildStatblockIsland` (`internal/site/statblock_page.go`) calls it and overrides `sbIsland.Ancestry` when the result is non-empty. No DOM/CSS change; the override flows to every island consumer because it is keyed on the existing `scc` frontmatter field.

**Tech Stack:** Go (go1.26.1 via devbox). Tests run with `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/...'`.

**Reference spec:** `docs/superpowers/specs/2026-06-15-summoner-statblock-provenance-eyebrow-design.md`

---

## Background facts (so you don't re-derive them)

- All Go/test commands need devbox: bare `go` is not on PATH. Use
  `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run X -v'`
  from the workspace root (`/home/scott/code/steelCompendium/workspace`).
- The four affected SCC type-path shapes, all under source prefix `mcdm.summoner.`
  (the full code is `<source>/<type-path>/<item>`):
  - `monster.rivals.{ech}.summoner.minion` — rival's summoned minions
  - `monster.rivals.{ech}.statblock` — the Rival Summoner elite itself
  - `monster.minion.summoner.{circle}.statblock` — heroic-summoner portfolio minions
  - `monster.champion.summoner.{circle}.statblock` — summoned champions
- `{ech}` is a segment like `4th-echelon`; `{circle}` is `undead`/`demon`/`elemental`/`fey`.
- The Monsters book has a look-alike `mcdm.monsters.v1/monster.rivals.{ech}.statblock`
  tree (28 entries) that MUST stay untouched — the `mcdm.summoner.` source-prefix
  gate is what excludes it.
- Reuse the existing `titleCase(s string)` helper (`internal/site/build.go:1396`)
  for the circle segment — do NOT add a new title-caser.

---

### Task 1: `summonerProvenanceEyebrow` helper (TDD)

**Files:**
- Create: `steel-etl/internal/site/summoner_provenance.go`
- Test: `steel-etl/internal/site/summoner_provenance_test.go`

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/site/summoner_provenance_test.go`:

```go
package site

import "testing"

func TestSummonerProvenanceEyebrow(t *testing.T) {
	cases := []struct {
		name string
		scc  string
		want string
	}{
		{
			name: "rival minion",
			scc:  "mcdm.summoner.v1/monster.rivals.4th-echelon.summoner.minion/zombie-titan",
			want: "Rival Summoner Summon · Echelon 4",
		},
		{
			name: "rival elite",
			scc:  "mcdm.summoner.v1/monster.rivals.1st-echelon.statblock/rival-summoner",
			want: "Rival Summoner · Echelon 1",
		},
		{
			name: "portfolio minion",
			scc:  "mcdm.summoner.v1/monster.minion.summoner.undead.statblock/skeleton",
			want: "Summoner Minion · Undead",
		},
		{
			name: "champion",
			scc:  "mcdm.summoner.v1/monster.champion.summoner.demon.statblock/demon-lords-aspect",
			want: "Summoner Champion · Demon",
		},
		{
			// CRITICAL non-match: Monsters-book rivals share the shape but are a
			// different book and must be left alone.
			name: "monsters-book rival is not matched",
			scc:  "mcdm.monsters.v1/monster.rivals.4th-echelon.statblock/rival-fury",
			want: "",
		},
		{
			name: "unrelated summoner code is not matched",
			scc:  "mcdm.summoner.v1/class/summoner",
			want: "",
		},
		{
			name: "empty",
			scc:  "",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := summonerProvenanceEyebrow(tc.scc); got != tc.want {
				t.Errorf("summonerProvenanceEyebrow(%q) = %q, want %q", tc.scc, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails (compile error — function undefined)**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestSummonerProvenanceEyebrow'`
Expected: FAIL — `undefined: summonerProvenanceEyebrow`.

- [ ] **Step 3: Write the implementation**

Create `steel-etl/internal/site/summoner_provenance.go`:

```go
package site

import "strings"

// summonerProvenanceEyebrow maps a summoner-book creature statblock's SCC code to
// the provenance label shown in the statblock head's sb__kw eyebrow, overriding
// the otherwise-useless keyword line (which today renders "—" or, for portfolio
// minions, the creature's own name). Returns "" for any code that is not a
// summoner-book creature statblock.
//
// The Monsters-book `mcdm.monsters.v1/monster.rivals.{ech}.statblock` tree shares
// the `monster.rivals.*` shape but is a different book and must stay untouched —
// the `mcdm.summoner.` source-prefix gate is what excludes it.
//
// Recognized type-paths (the segment between source and item):
//
//	monster.rivals.{ech}.summoner.minion         → "Rival Summoner Summon · Echelon N"
//	monster.rivals.{ech}.statblock               → "Rival Summoner · Echelon N"
//	monster.minion.summoner.{circle}.statblock   → "Summoner Minion · {Circle}"
//	monster.champion.summoner.{circle}.statblock → "Summoner Champion · {Circle}"
func summonerProvenanceEyebrow(scc string) string {
	scc = strings.TrimSpace(scc)
	src, rest, ok := strings.Cut(scc, "/")
	if !ok || !strings.HasPrefix(src, "mcdm.summoner.") {
		return ""
	}
	// Drop the trailing /item to leave the type-path.
	typePath, _, ok := strings.Cut(rest, "/")
	if !ok {
		return ""
	}
	seg := strings.Split(typePath, ".")

	switch {
	// monster.rivals.{ech}.summoner.minion
	case len(seg) == 5 && seg[0] == "monster" && seg[1] == "rivals" &&
		seg[3] == "summoner" && seg[4] == "minion":
		if n := echelonNum(seg[2]); n != "" {
			return "Rival Summoner Summon · Echelon " + n
		}
	// monster.rivals.{ech}.statblock
	case len(seg) == 4 && seg[0] == "monster" && seg[1] == "rivals" &&
		seg[3] == "statblock":
		if n := echelonNum(seg[2]); n != "" {
			return "Rival Summoner · Echelon " + n
		}
	// monster.minion.summoner.{circle}.statblock
	case len(seg) == 5 && seg[0] == "monster" && seg[1] == "minion" &&
		seg[2] == "summoner" && seg[4] == "statblock":
		return "Summoner Minion · " + titleCase(seg[3])
	// monster.champion.summoner.{circle}.statblock
	case len(seg) == 5 && seg[0] == "monster" && seg[1] == "champion" &&
		seg[2] == "summoner" && seg[4] == "statblock":
		return "Summoner Champion · " + titleCase(seg[3])
	}
	return ""
}

// echelonNum extracts the leading number N from an "Nth-echelon" segment
// (e.g. "4th-echelon" → "4"). Returns "" if the segment isn't of that shape.
func echelonNum(seg string) string {
	rest, ok := strings.CutSuffix(seg, "-echelon")
	if !ok {
		return ""
	}
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		i++
	}
	if i == 0 {
		return ""
	}
	return rest[:i]
}
```

Note: `titleCase` already exists at `internal/site/build.go:1396` (capitalizes the
first letter of each whitespace-separated word) — reuse it, do not redefine.

- [ ] **Step 4: Run test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestSummonerProvenanceEyebrow -v'`
Expected: PASS (all seven sub-tests).

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add internal/site/summoner_provenance.go internal/site/summoner_provenance_test.go
git commit -m "feat(site): derive summoner-book statblock provenance eyebrow from SCC"
```

---

### Task 2: Override `sbIsland.Ancestry` in `buildStatblockIsland`

**Files:**
- Modify: `steel-etl/internal/site/statblock_page.go` (function `buildStatblockIsland`, the `Ancestry:` field at line ~172)
- Test: `steel-etl/internal/site/statblock_page_test.go` (create if absent, else append)

- [ ] **Step 1: Write the failing test**

Append to `steel-etl/internal/site/statblock_page_test.go` (create the file with the
`package site` + import header if it does not yet exist):

```go
func TestBuildStatblockIsland_ProvenanceEyebrowOverridesKeywords(t *testing.T) {
	// A rival summoner minion: keywords say "—", but the scc carries echelon +
	// rival context, so the eyebrow (Ancestry) must be the derived provenance.
	fm := "name: Zombie Titan\n" +
		"organization: Minion\n" +
		"role: Defender\n" +
		"keywords:\n    - —\n" +
		"scc: mcdm.summoner.v1/monster.rivals.4th-echelon.summoner.minion/zombie-titan\n"
	got := buildStatblockIsland(fm, "")
	if got.Ancestry != "Rival Summoner Summon · Echelon 4" {
		t.Errorf("Ancestry = %q, want %q", got.Ancestry, "Rival Summoner Summon · Echelon 4")
	}
}

func TestBuildStatblockIsland_NonSummonerKeepsKeywords(t *testing.T) {
	// A Monsters-book statblock keeps its real keyword-derived ancestry.
	fm := "name: Goblin Warrior\n" +
		"organization: Minion\n" +
		"role: Harrier\n" +
		"keywords:\n    - Humanoid\n    - Goblin\n" +
		"scc: mcdm.monsters.v1/monster.goblins.statblock/goblin-warrior\n"
	got := buildStatblockIsland(fm, "")
	if got.Ancestry != "Humanoid, Goblin" {
		t.Errorf("Ancestry = %q, want %q", got.Ancestry, "Humanoid, Goblin")
	}
}
```

If you must create the file, the header is:

```go
package site

import "testing"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildStatblockIsland_ -v'`
Expected: `TestBuildStatblockIsland_ProvenanceEyebrowOverridesKeywords` FAILS
(`Ancestry = "—"`); the non-summoner test PASSES already.

- [ ] **Step 3: Implement the override**

In `steel-etl/internal/site/statblock_page.go`, inside `buildStatblockIsland`,
replace the inline `Ancestry:` field initializer with a local variable computed
before the `return`. Change the block starting at line ~150.

Find:

```go
	captain := "—"
	if m := sbCaptainRe.FindStringSubmatch(body); m != nil {
		captain = strings.TrimSpace(m[1])
	}

	return sbIsland{
		ID:       slugify(name),
		Name:     name,
		Ancestry: strings.Join(parseFrontmatterList(fm, "keywords"), ", "),
```

Replace with:

```go
	captain := "—"
	if m := sbCaptainRe.FindStringSubmatch(body); m != nil {
		captain = strings.TrimSpace(m[1])
	}

	// The keyword line (sb__kw eyebrow) is "—" or junk for summoner-book
	// statblocks; replace it with a provenance label derived from the scc code.
	ancestry := strings.Join(parseFrontmatterList(fm, "keywords"), ", ")
	if eb := summonerProvenanceEyebrow(parseFrontmatterField(fm, "scc")); eb != "" {
		ancestry = eb
	}

	return sbIsland{
		ID:       slugify(name),
		Name:     name,
		Ancestry: ancestry,
```

- [ ] **Step 4: Run the new tests + the whole site package to verify pass + no regressions**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/...'`
Expected: PASS. In particular the existing `.sb-wrap` golden-equivalence test
must still pass — its fixtures are non-summoner, so the override never fires.

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add internal/site/statblock_page.go internal/site/statblock_page_test.go
git commit -m "feat(site): show provenance eyebrow on summoner-book statblocks"
```

---

### Task 3: End-to-end verification on the real site output

**Files:** none (verification only).

- [ ] **Step 1: Regenerate the summoner book + site**

From the workspace root, regenerate just what we need to inspect. The summoner
book is a configured book, so use `--book`:

Run:
```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --book summoner && go run ./cmd/steel-etl site --config ../v2/site.yaml'
```
Expected: completes without error.

- [ ] **Step 2: Confirm the eyebrow on one of each of the four types**

Run (from workspace root):
```bash
for f in \
  v2/docs/Browse/monster/rivals/4th-echelon/summoner/minion/zombie-titan.md \
  v2/docs/Browse/monster/rivals/4th-echelon/rival-summoner.md \
  v2/docs/Browse/monster/minion/summoner/undead/skeleton.md \
  v2/docs/Browse/monster/champion/summoner/demon/demon-lords-aspect.md ; do
  echo "$f:"; grep -oE '<div class="sb__kw">[^<]*</div>' "$f" | head -1
done
```
Expected:
- rival minion → `<div class="sb__kw">Rival Summoner Summon · Echelon 4</div>`
- rival elite → `<div class="sb__kw">Rival Summoner · Echelon 4</div>`
- portfolio minion → `<div class="sb__kw">Summoner Minion · Undead</div>`
- champion → `<div class="sb__kw">Summoner Champion · Demon</div>`

- [ ] **Step 3: Confirm a Monsters-book rival is untouched**

Run (from workspace root):
```bash
grep -oE '<div class="sb__kw">[^<]*</div>' \
  v2/docs/Browse/monster/rivals/4th-echelon/*fury*.md | head
```
Expected: its keyword line is whatever it was before (NOT a "Rival Summoner …"
label) — i.e. the summoner override did not leak into the Monsters book.

- [ ] **Step 4: Do NOT commit generated output**

`v2/docs/Browse/**` and `data/data-*` are generated and are committed only by the
`just deploy*` recipes. Leave them; this task is inspection only. If `git status`
in the workspace shows regenerated files, revert them:
`git -C /home/scott/code/steelCompendium/workspace checkout -- v2/docs data 2>/dev/null || true`
(deploy decides when regenerated output lands — separate from this feature).

---

## Self-Review

- **Spec coverage:** All four label shapes (Task 1) ✓; `mcdm.summoner.` source gate
  excluding Monsters-book rivals (Task 1 test + Task 3 step 3) ✓; override in
  `buildStatblockIsland` keyed on `scc` (Task 2) ✓; reuse existing `titleCase`,
  no DOM/CSS change ✓; golden test unaffected (Task 2 step 4) ✓; fixtures/backlink/
  data-format out of scope — no tasks needed ✓.
- **Placeholder scan:** none — every code/command step is concrete.
- **Type consistency:** `summonerProvenanceEyebrow(string) string`, `echelonNum(string) string`,
  and the field `sbIsland.Ancestry` are used identically across Tasks 1–2.
