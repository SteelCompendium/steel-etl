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
