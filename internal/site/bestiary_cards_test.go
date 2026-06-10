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
