package site

import (
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
		`<div class="sc-card__name">Pillar</div>`, `>EV</div>`, `>Level</div>`} {
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
		`<span class="sc-tag">Angulotl</span>`, `Poison 2`, `>Level</div>`} {
		if !strings.Contains(got, want) {
			t.Errorf("retainerCard missing %q in:\n%s", want, got)
		}
	}
}
