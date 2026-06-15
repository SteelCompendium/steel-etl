package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A minion: organization "Minion", a single passive trait + one ability.
const minionPage = `---
name: Goblin Cutter
organization: Minion
role: Harrier
level: 1
ev: "3"
size: 1S
speed: 6
stamina: "5"
stability: "0"
free_strike: "2"
might: "1"
agility: "2"
reason: "-1"
intuition: "0"
presence: "-1"
keywords:
    - Goblin
type: statblock
---

> 🗡️ **Cutting Strike**
>
> | **Melee, Strike** | **Main action** |
> |-------------------|----------------:|
> | **📏 Melee 1**    | **🎯 One creature** |
>
> **Power Roll + 2:**
>
> - **≤11:** 2 damage
> - **12-16:** 4 damage
> - **17+:** 6 damage

> ⭐️ **Mob Tactics**
>
> The cutter deals 1 extra damage for each other goblin adjacent to its target.
`

// A summoner signature with the dice-in-title power-roll form: the dice live in
// the title and the three tiers are bare digit-led paragraphs below the table.
const summonerDicePage = `---
name: Bound Imp
organization: ""
role: Support
level: 1
ev: "4"
size: 1T
speed: 5
stamina: "8"
stability: "0"
free_strike: "1"
might: "0"
agility: "2"
reason: "1"
intuition: "1"
presence: "0"
keywords:
    - Demon
type: statblock
---

> 🏹 **Spirit Bolt 2d10 + R**
>
> | **Magic, Ranged** | **Main action** |
> |-------------------|----------------:|
> | **📏 Ranged 10**  | **🎯 One creature** |
>
> 11 damage
>
> 16 damage; pushed 1
>
> 21 damage; pushed 2
`

// goldenFixtures maps a golden basename to its source page markdown. The two
// reused constants live in statblock_page_test.go.
var goldenFixtures = map[string]string{
	"devil-high-judge": devilHighJudgePage,
	"link-test":        linkedFieldsPage,
	"minion":           minionPage,
	"summoner-dice":    summonerDicePage,
}

const goldenDir = "testdata/statblock_golden"

// islandFor reproduces exactly what buildStatblockIslandPage feeds the renderer:
// split frontmatter, strip retainer-advancement (a no-op for these fixtures),
// then build the island.
func islandFor(page string) sbIsland {
	fm, body := splitFrontmatter(page)
	base, _ := splitRetainerAdvancement(body)
	return buildStatblockIsland(fm, base)
}

// TestStatblockGolden_WriteIslandInputs regenerates the committed island JSON
// inputs the Brave capture script consumes. It only writes when
// STEEL_UPDATE_GOLDEN=1; otherwise it asserts the committed JSON still matches
// the current parser output (so a parser change that drifts the inputs fails
// loudly, telling you to regenerate + recapture).
func TestStatblockGolden_WriteIslandInputs(t *testing.T) {
	update := os.Getenv("STEEL_UPDATE_GOLDEN") == "1"
	for name, page := range goldenFixtures {
		isl := islandFor(page)
		got, err := json.MarshalIndent(isl, "", "  ")
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		got = append(got, '\n')
		path := filepath.Join(goldenDir, name+".island.json")
		if update {
			if err := os.MkdirAll(goldenDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, got, 0644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v (run STEEL_UPDATE_GOLDEN=1 go test to generate)", name, err)
		}
		if string(got) != string(want) {
			t.Errorf("%s.island.json drifted from parser output — regenerate with STEEL_UPDATE_GOLDEN=1 and recapture golden.html", name)
		}
	}
}

// normalizeStatblockHTML drops insignificant whitespace so a Go single-line
// build matches the browser's outerHTML serialization. Neither side emits
// inter-tag whitespace (the JS html string is fully concatenated; Go uses a
// single Builder), so stripping newlines/tabs + trimming is sufficient.
func normalizeStatblockHTML(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "")
	return strings.TrimSpace(s)
}

func TestStatblockCard_GoldenEquivalence(t *testing.T) {
	for name, page := range goldenFixtures {
		t.Run(name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join(goldenDir, name+".golden.html"))
			if err != nil {
				t.Fatalf("%v (run the capture script — Task 2)", err)
			}
			got := renderStatblockCard(islandFor(page))
			if g, w := normalizeStatblockHTML(got), normalizeStatblockHTML(string(want)); g != w {
				t.Errorf("renderStatblockCard != golden for %s\n--- got ---\n%s\n--- want ---\n%s", name, g, w)
			}
		})
	}
}

func TestRenderStatblockHead_OmitsEmptyEV(t *testing.T) {
	withEV := renderStatblockHead(sbIsland{Name: "X", Level: "1", Role: "Brute", RoleKey: "brute", EV: "32"})
	if !strings.Contains(withEV, "EV 32") {
		t.Errorf("expected EV when present: %s", withEV)
	}
	noEV := renderStatblockHead(sbIsland{Name: "Panther", Level: "1", Role: "Companion", RoleKey: "leader", EV: ""})
	if strings.Contains(noEV, `class="sb__ev"`) {
		t.Errorf("expected no EV div when EV empty: %s", noEV)
	}
}
