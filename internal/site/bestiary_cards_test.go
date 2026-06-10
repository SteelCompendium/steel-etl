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
		`>EV</div>`, `>Level</div>`, `>Size</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("statblockCard missing %q in:\n%s", want, got)
		}
	}
}
