package site

import (
	"strings"
	"testing"
)

// pantherCompanionBody is the verbatim base region (table + abilities, no
// advancement section) of a generated companion leaf body.
const pantherCompanionBody = `| Animal, Companion |           -           |                        Level 1                        |          -          |            -             |
|:-----------------:|:---------------------:|:-----------------------------------------------------:|:-------------------:|:------------------------:|
|  **1M**<br>Size   |    **7**<br>Speed     |                **= yours**<br>Stamina                 | **1**<br>Stability  | **1 + M**<br>Free Strike |
| **—**<br>Immunity | **Climb**<br>Movement | **[Sneak](../../../skill/intrigue/sneak.md)**<br>Skills |                     |                          |
|  **+2**<br>Might  |   **+2**<br>Agility   |                   **−1**<br>Reason                    | **+2**<br>Intuition |    **+1**<br>Presence    |

## Pounce {data-scc="mcdm.beastheart.v1/feature.ability.companion.beastheart.panther.level-1/pounce"}

*The panther bunches up, then uncoils into a deadly leap.*

| **Companion, Melee, Weapon** |     **Maneuver** |
|------------------------------|-----------------:|
| **📏 Melee 1**               | **🎯 One enemy** |

**Effect:** The target takes damage equal to 3 + the panther's Might score.

**Spend 1 Ferocity:** The panther can jump up to a number of squares equal to their speed.

## Mighty Spring {data-scc="mcdm.beastheart.v1/feature.companion.beastheart.panther.level-1/mighty-spring"}

Whenever the panther takes the Advance move action, they can jump up to a number of squares equal to their speed.`

func TestParseCompanionGrid_Panther(t *testing.T) {
	g := parseCompanionGrid(pantherCompanionBody)
	if g.keywords != "Animal, Companion" {
		t.Errorf("keywords = %q", g.keywords)
	}
	if g.level != "1" {
		t.Errorf("level = %q, want 1", g.level)
	}
	want := map[string]string{
		"Size": "1M", "Speed": "7", "Stamina": "= yours", "Stability": "1",
		"Free Strike": "1 + M", "Immunity": "—", "Movement": "Climb",
		"Might": "+2", "Agility": "+2", "Reason": "−1", "Intuition": "+2", "Presence": "+1",
	}
	for k, v := range want {
		if g.cells[k] != v {
			t.Errorf("cell[%q] = %q, want %q", k, g.cells[k], v)
		}
	}
	// Skills keeps its markdown link (resolved later by resolveSbLinks).
	if !strings.Contains(g.cells["Skills"], "[Sneak](") {
		t.Errorf("skills cell = %q, want a Sneak link", g.cells["Skills"])
	}
}
