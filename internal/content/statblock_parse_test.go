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

// Item A: dice-in-title with a link-wrapped characteristic (e.g. [R](scc:…)) in
// the roll suffix. The parser must strip links to display text in the roll field
// but preserve them verbatim in tier values.
func TestStatblock_DiceTitle_ToleratesLinkedCharacteristic(t *testing.T) {
	block := "" +
		"> 🏹 **Mind Twist 2d10 + [R](scc:mcdm.heroes.v1/rule.character/reason) (Signature Ability)**\n" +
		">\n" +
		"> | **Magic, Ranged, Strike** | **Main action** |\n" +
		"> |---------------------------|----------------:|\n" +
		"> | **📏 Ranged 5** | **🎯 One creature** |\n" +
		">\n" +
		"> 4 damage; P < WEAK [slowed](scc:mcdm.heroes.v1/condition/slowed) (save ends)\n" +
		"> 6 damage; P < AVERAGE slowed (save ends)\n" +
		"> 8 damage; P < STRONG slowed (save ends)\n"
	got := ParseStatblockFeatures(block)
	if len(got) != 1 {
		t.Fatalf("got %d features, want 1", len(got))
	}
	a := got[0]
	if a["name"] != "Mind Twist" {
		t.Errorf("name = %v, want Mind Twist (dice + linked characteristic stripped from title)", a["name"])
	}
	effects, _ := a["effects"].([]map[string]any)
	if len(effects) != 1 {
		t.Fatalf("effects = %v, want 1", a["effects"])
	}
	e := effects[0]
	if e["roll"] != "2d10 + R" {
		t.Errorf("roll = %v, want '2d10 + R' (link stripped to display in the structured roll)", e["roll"])
	}
	// Tier VALUES keep the raw link verbatim (data-field convention).
	if e["tier1"] != "4 damage; P < WEAK [slowed](scc:mcdm.heroes.v1/condition/slowed) (save ends)" {
		t.Errorf("tier1 = %v (linked tier value must be preserved verbatim)", e["tier1"])
	}
}

// Item B: tier lines (labeled and bare) must accept link-wrapped values verbatim.
// These are regression guards — no code change expected.
func TestStatblock_TierLines_PreserveLinks(t *testing.T) {
	bare := "4 damage; the target is [prone](scc:mcdm.heroes.v1/condition/prone)"
	if !sbBareTierRe.MatchString(bare) {
		t.Errorf("bare tier line no longer recognized after linking its value")
	}
	labeled := "- **≤11:** 2 damage; [grabbed](scc:mcdm.heroes.v1/condition/grabbed)"
	m := sbTierRe.FindStringSubmatch(labeled)
	if m == nil {
		t.Fatalf("labeled tier line not matched: %q", labeled)
	}
	if want := "2 damage; [grabbed](scc:mcdm.heroes.v1/condition/grabbed)"; m[2] != want {
		t.Errorf("tier value = %q, want %q (verbatim)", m[2], want)
	}
}

// Item C: stat-grid cell with a link-wrapped value. cellRe must still match, and
// linkDisplay must recover the display text. The stat-grid consumer strips links
// from structured stat values (size/speed/stamina/movement/etc.).
func TestStatblock_Cell_ToleratesLinkedValue(t *testing.T) {
	cell := "**[Teleport](scc:mcdm.heroes.v1/movement/teleport)**<br>Movement"
	m := cellRe.FindStringSubmatch(cell)
	if m == nil {
		t.Fatalf("linked stat-grid cell not matched: %q", cell)
	}
	if got := linkDisplay(m[1]); got != "Teleport" {
		t.Errorf("cell value display = %q, want Teleport", got)
	}
	if m[2] != "Movement" {
		t.Errorf("cell label = %q, want Movement", m[2])
	}
}

func TestStatblock_LabeledPowerRoll_ToleratesLinkedHeader(t *testing.T) {
	// A future editor links "Power Roll" in the labeled header form. The parser
	// must still find the power-roll block and its tiers.
	block := "" +
		"> 🗡 **Hop and Chop (Signature Ability)**\n" +
		">\n" +
		"> | **Melee, Strike, Weapon** | **Main action** |\n" +
		"> |---------------------------|----------------:|\n" +
		"> | **📏 Melee 1** | **🎯 One creature** |\n" +
		">\n" +
		"> **[Power Roll](scc:mcdm.heroes.v1/rule.dice/power-roll) + 2:**\n" +
		">\n" +
		"> - **≤11:** 2 damage\n" +
		"> - **12-16:** 4 damage\n" +
		"> - **17+:** 5 damage\n"

	got := ParseStatblockFeatures(block)
	if len(got) != 1 {
		t.Fatalf("got %d features, want 1", len(got))
	}
	effects, _ := got[0]["effects"].([]map[string]any)
	if len(effects) != 1 {
		t.Fatalf("effects = %v, want 1 (linked Power Roll header must still be recognized)", got[0]["effects"])
	}
	e := effects[0]
	if e["tier1"] == nil || e["tier1"] == "" || e["tier3"] == nil || e["tier3"] == "" {
		t.Errorf("tiers not extracted after linking the header (tier1=%v tier3=%v)", e["tier1"], e["tier3"])
	}
	// The roll must be recognized and stored as link-free display text.
	// Without the hardened regex, sbPowerRollRe does NOT match the linked header,
	// so roll stays "" (empty) — this assertion is what catches the regression.
	if e["roll"] != "Power Roll + 2" {
		t.Errorf("roll = %v, want 'Power Roll + 2' (linked header must be recognized and link stripped)", e["roll"])
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
