package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goblinWarriorFM = `agility: 2
ev: "3"
keywords:
    - Goblin
    - Humanoid
level: 1
name: Goblin Warrior
organization: Horde
role: Harrier
size: 1S
speed: 6
`

func TestStatblockCard(t *testing.T) {
	got := statblockCard(goblinWarriorFM, "", "goblin-warrior.md", "Goblin Warrior")
	for _, want := range []string{
		`class="sc-card sc-fil"`,
		`href="goblin-warrior/"`,
		`Horde Harrier`,                 // organization + role type label
		`<div class="sc-card__name">Goblin Warrior</div>`,
		`<span class="sc-tag">Goblin</span>`,
		`<span class="sc-tag">Humanoid</span>`,
		`>EV</div>`, `>Level</div>`, `>Size</div>`, `>Speed</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("statblockCard missing %q in:\n%s", want, got)
		}
	}
}

func TestStatblockTypeLabel(t *testing.T) {
	for _, tc := range []struct{ org, role, want string }{
		{"Horde", "Harrier", "Horde Harrier"},
		{"Horde", "", "Horde"},
		{"", "Harrier", "Harrier"},
		{"", "", "Statblock"},
	} {
		fm := "organization: " + tc.org + "\nrole: " + tc.role + "\n"
		if got := statblockTypeLabel(fm); got != tc.want {
			t.Errorf("statblockTypeLabel(org=%q role=%q) = %q, want %q", tc.org, tc.role, got, tc.want)
		}
	}
}

const pillarFM = `ev: "3"
level: "2"
name: Pillar
size: One square that can't be moved through
`

func TestTerrainCard(t *testing.T) {
	got := terrainCard(pillarFM, "This stone pillar can be toppled.", "pillar.md", "Pillar")
	for _, want := range []string{`href="pillar/"`, `Dynamic Terrain`,
		`<div class="sc-card__name">Pillar</div>`, `>EV</div>`, `>Level</div>`, `>Size</div>`,
		`<div class="sc-card__flavor">`, `stone pillar`} {
		if !strings.Contains(got, want) {
			t.Errorf("terrainCard missing %q in:\n%s", want, got)
		}
	}
}

const hopperFM = `ev: '-'
immunities:
    - Poison 2
keywords:
    - Angulotl
    - Humanoid
level: 1
name: Angulotl Hopper
role: Harrier
size: 1S
`

func TestRetainerCard(t *testing.T) {
	got := retainerCard(hopperFM, "", "angulotl-hopper.md", "Angulotl Hopper")
	for _, want := range []string{`href="angulotl-hopper/"`, `Retainer Harrier`,
		`<span class="sc-tag">Angulotl</span>`, `Poison 2`, `>Level</div>`, `>Size</div>`} {
		if !strings.Contains(got, want) {
			t.Errorf("retainerCard missing %q in:\n%s", want, got)
		}
	}
}

func TestRetainerCardNoRole(t *testing.T) {
	got := retainerCard("level: 1\nname: Plain\n", "", "plain.md", "Plain")
	if !strings.Contains(got, `<div class="sc-card__type">Retainer</div>`) {
		t.Errorf("roleless retainer should label \"Retainer\" with no trailing space:\n%s", got)
	}
}

func writeMD(t *testing.T, path, fm string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\n"+fm+"---\n\n# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

const goblinMaliceFM = "name: Goblin Malice\ntype: featureblock\n"

func TestMonsterGroupContent_Flat(t *testing.T) {
	root := t.TempDir()
	grp := filepath.Join(root, "monster", "goblins")
	writeMD(t, filepath.Join(grp, "goblin-malice.md"), goblinMaliceFM)
	writeMD(t, filepath.Join(grp, "statblock", "goblin-warrior.md"), goblinWarriorFM)

	got, ok := buildMonsterGroupContent(grp, "goblins",
		[]string{"goblin-malice.md"}, []string{"statblock"})
	if !ok {
		t.Fatal("expected goblins to be a monster group")
	}
	for _, want := range []string{
		"# Goblins\n\n---\n\n",                         // strippable head for mergeGroupLanding
		`href="goblin-malice/"`,                        // featureblock card
		`>Goblin Malice<`,
		`href="statblock/goblin-warrior/"`,             // statblock card, group-relative href
		`Horde Harrier`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("flat group missing %q in:\n%s", want, got)
		}
	}
}

func TestMonsterGroupContent_Echelon(t *testing.T) {
	root := t.TempDir()
	grp := filepath.Join(root, "monster", "demons")
	writeMD(t, filepath.Join(grp, "1st-echelon", "demon-malice-level-1.md"), "name: Demon Malice (Level 1)\ntype: featureblock\n")
	writeMD(t, filepath.Join(grp, "1st-echelon", "statblock", "spite.md"), goblinWarriorFM)

	got, ok := buildMonsterGroupContent(grp, "demons", nil, []string{"1st-echelon"})
	if !ok {
		t.Fatal("expected demons to be a monster group")
	}
	for _, want := range []string{
		"## 1st Echelon",                                    // per-echelon sub-header
		`href="1st-echelon/demon-malice-level-1/"`,          // echelon-relative featureblock href
		`href="1st-echelon/statblock/spite/"`,               // echelon-relative statblock href
	} {
		if !strings.Contains(got, want) {
			t.Errorf("echelon group missing %q in:\n%s", want, got)
		}
	}
}

func TestMonsterGroupContent_NotAGroup(t *testing.T) {
	// the monster/ root itself and a statblock/ leaf are NOT groups
	if _, ok := buildMonsterGroupContent("/x/monster", "monster", nil, []string{"goblins"}); ok {
		t.Error("monster root should not be a group")
	}
	if _, ok := buildMonsterGroupContent("/x/monster/goblins/statblock", "statblock", []string{"a.md"}, nil); ok {
		t.Error("statblock leaf should not be a group")
	}
}

func TestBuildCardsContent_Bestiary(t *testing.T) {
	root := t.TempDir()
	sb := filepath.Join(root, "monster", "goblins", "statblock")
	writeMD(t, filepath.Join(sb, "goblin-warrior.md"), goblinWarriorFM)
	got, ok := buildCardsContent(sb, "statblock", []string{"goblin-warrior.md"}, nil)
	if !ok {
		t.Fatal("expected statblock leaf to produce cards")
	}
	if !strings.Contains(got, "Horde Harrier") || !strings.Contains(got, `class="sc-cards"`) {
		t.Errorf("statblock grid wrong:\n%s", got)
	}

	rt := filepath.Join(root, "retainer", "statblock")
	writeMD(t, filepath.Join(rt, "angulotl-hopper.md"), hopperFM)
	// routing is by segment presence, not case order: "monster" is absent from
	// a retainer/statblock path, so the monster-statblock case can't fire.
	got, ok = buildCardsContent(rt, "statblock", []string{"angulotl-hopper.md"}, nil)
	if !ok || !strings.Contains(got, "Retainer Harrier") {
		t.Errorf("retainer/statblock path should route to retainerCard, not statblockCard:\n%s", got)
	}

	dt := filepath.Join(root, "dynamic-terrain", "mechanisms")
	writeMD(t, filepath.Join(dt, "pillar.md"), pillarFM)
	got, ok = buildCardsContent(dt, "mechanisms", []string{"pillar.md"}, nil)
	if !ok || !strings.Contains(got, "Dynamic Terrain") {
		t.Errorf("terrain leaf wrong:\n%s", got)
	}
}
