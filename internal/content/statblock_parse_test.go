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
	got := ParseStatblockFeatures(cursespitterFeatures)
	if len(got) != 2 {
		t.Fatalf("got %d features, want 2", len(got))
	}

	ability := got[0]
	if ability["name"] != "Eye of Surlach" {
		t.Errorf("name: got %v", ability["name"])
	}
	// Monster actions stay feature_type=ability; passives stay trait (Crafty,
	// below). The taxonomy refactor must not disturb the statblock split.
	if ability["feature_type"] != "ability" {
		t.Errorf("feature_type: got %v, want ability", ability["feature_type"])
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

// Summoner minion/fixture/champion signature abilities encode the power roll in
// the TITLE as dice notation ("Nd10 + <char>"), followed by three bare digit-led
// tier outcome lines (no "≤11:"/"12-16:" labels). The parser must strip the dice
// to effects.roll, clean the name, and map the three lines to tier1/2/3.
const moltenStrikeFeature = "" +
	"> 🏹 **Molten Strike 2d10 + R (Signature Ability)**\n" +
	">\n" +
	"> | **Magic, Melee, Strike** |        **Main action** |\n" +
	"> |--------------------------|----------------:|\n" +
	"> | **📏 Melee 2** | **🎯 One creature or object per minion** |\n" +
	">\n" +
	"> 4 fire damage; shift 3\n" +
	">\n" +
	"> 6 fire damage; shift 4\n" +
	">\n" +
	"> 8 fire damage; shift 5\n" +
	">\n" +
	"> **Effect:** Each square that the flow shifts into becomes wreathed in flames.\n"

func TestParseStatblockFeatureDiceInTitle(t *testing.T) {
	got := ParseStatblockFeatures(moltenStrikeFeature)
	if len(got) != 1 {
		t.Fatalf("got %d features, want 1", len(got))
	}
	a := got[0]
	if a["name"] != "Molten Strike" {
		t.Errorf("name: got %v, want Molten Strike (dice stripped)", a["name"])
	}
	if a["ability_type"] != "Signature Ability" {
		t.Errorf("ability_type: got %v", a["ability_type"])
	}
	if a["feature_type"] != "ability" {
		t.Errorf("feature_type: got %v, want ability", a["feature_type"])
	}
	effects, _ := a["effects"].([]map[string]any)
	if len(effects) != 1 {
		t.Fatalf("effects: got %v, want 1 entry", a["effects"])
	}
	e := effects[0]
	if e["roll"] != "2d10 + R" {
		t.Errorf("roll: got %v, want '2d10 + R'", e["roll"])
	}
	if e["tier1"] != "4 fire damage; shift 3" {
		t.Errorf("tier1: got %v", e["tier1"])
	}
	if e["tier2"] != "6 fire damage; shift 4" {
		t.Errorf("tier2: got %v", e["tier2"])
	}
	if e["tier3"] != "8 fire damage; shift 5" {
		t.Errorf("tier3: got %v", e["tier3"])
	}
}

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
	got := ParseStatblockFeatures(block)
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
