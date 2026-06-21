package content

import (
	"reflect"
	"strings"
	"testing"
)

const cursespitterGrid = "" +
	"| Goblin, Humanoid  |           -           |      Level 1      |      Horde Hexer      |         EV 3         |\n" +
	"|:-----------------:|:---------------------:|:-----------------:|:---------------------:|:--------------------:|\n" +
	"|  **1S**<br>Size   |    **5**<br>Speed     | **10**<br>Stamina |  **0**<br>Stability   | **1**<br>Free Strike |\n" +
	"| **-**<br>Immunity | **Climb**<br>Movement |         -         | **-**<br>With Captain |  **-**<br>Weakness   |\n" +
	"|  **-2**<br>Might  |   **+1**<br>Agility   |  **0**<br>Reason  |  **+2**<br>Intuition  |  **0**<br>Presence   |\n"

// TestParseStatGridKeywordDomains covers the Summoner-book elemental keyword
// cell shape "Elemental (Air, Earth)": the comma inside the parenthetical is a
// domain qualifier, not a keyword separator, so it must be distributed onto the
// base keyword ("Elemental (Air)", "Elemental (Earth)") rather than naively
// split into the broken "Elemental (Air" / "Earth)" pair. Top-level commas
// (outside parentheses) still separate distinct keywords.
func TestParseStatGridKeywordDomains(t *testing.T) {
	tests := []struct {
		name   string
		header string
		wantKW []string
	}{
		{
			name:   "domains distributed onto base keyword",
			header: "| Elemental (Air, Earth) | - | - | Minion Hexer | 3 essence for two minions |",
			wantKW: []string{"Elemental (Air)", "Elemental (Earth)"},
		},
		{
			name:   "single domain stays single",
			header: "| Elemental (Fire) | - | - | Minion Artillery | - |",
			wantKW: []string{"Elemental (Fire)"},
		},
		{
			name:   "four domains all distributed",
			header: "| Elemental (Air, Green, Fire, Void) | - | - | Minion Support | - |",
			wantKW: []string{"Elemental (Air)", "Elemental (Green)", "Elemental (Fire)", "Elemental (Void)"},
		},
		{
			name:   "top-level commas still separate keywords",
			header: "| Abyssal, Demon | - | - | Minion Brute | - |",
			wantKW: []string{"Abyssal", "Demon"},
		},
		{
			name:   "bare keyword unchanged",
			header: "| Undead | - | - | Minion Defender | - |",
			wantKW: []string{"Undead"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grid := tc.header + "\n|:-:|:-:|:-:|:-:|:-:|\n"
			got := parseStatGrid(grid).header.keywords
			if !reflect.DeepEqual(got, tc.wantKW) {
				t.Errorf("keywords: got %v, want %v", got, tc.wantKW)
			}
		})
	}
}

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

// TestParseStatblockFields_Flavor verifies the leading prose paragraph (the
// flavor text under a statblock heading, before the stat grid) is lifted into the
// `flavor` frontmatter field so it survives the v2 .sb-wrap card render — first
// seen on Summoner-book portfolio summons (e.g. the Ensnarer).
func TestParseStatblockFields_Flavor(t *testing.T) {
	const flavor = "This vaguely humanoid form is warped and distorted by a demon nestled inside them."
	body := flavor + "\n\n" + cursespitterGrid
	fm := ParseStatblockFields("Ensnarer", body)
	if fm["flavor"] != flavor {
		t.Errorf("flavor: got %q, want %q", fm["flavor"], flavor)
	}
}

func TestParseStatblockFields_NoFlavorWhenAbsent(t *testing.T) {
	fm := ParseStatblockFields("Cursespitter", cursespitterGrid)
	if v, ok := fm["flavor"]; ok {
		t.Errorf("flavor present when body has no leading prose: %q", v)
	}
}

// escapedPipeGrid models a summoner minion grid where Stamina holds three
// echelon values joined by escaped pipes ("4 \| 4 \| 4"). The escaped pipes are
// literal cell content, not column separators.
const escapedPipeGrid = "" +
	"| — | Humanoid | Level 1 | Minion | EV 1 |\n" +
	"|:-:|:--------:|:-------:|:------:|:----:|\n" +
	"| **1M**<br>Size | **5**<br>Speed | **4 \\| 4 \\| 4**<br>Stamina | **0**<br>Stability | **3**<br>Free Strike |\n"

func TestParseStatGridEscapedPipeStamina(t *testing.T) {
	got := parseStatGrid(escapedPipeGrid)
	if got.labels["Stamina"] != "4 | 4 | 4" {
		t.Errorf("Stamina: got %q, want %q", got.labels["Stamina"], "4 | 4 | 4")
	}
}

// summonerCostGrids exercise the Summoner-book header column layout (canonical,
// post column-fix): the trailing cell holds an in-play summon cost ("N essence …",
// "N Malice …") rather than an Encounter Value, so it lands in `cost`, not `ev`.
// Rivals keep a real "EV …" cell, which must still resolve to `ev` (not `cost`).
func TestParseStatGridCostVsEV(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		wantEV   string
		wantCost string
		wantOrg  string
		wantRole string
		wantLvl  int
		wantKW   []string
	}{
		{
			name:     "summoner minion essence cost",
			header:   "| Abyssal, Demon | - | - | Minion Brute | 3 essence for two minions |",
			wantCost: "3 essence for two minions",
			wantOrg:  "Minion", wantRole: "Brute",
			wantKW: []string{"Abyssal", "Demon"},
		},
		{
			name:    "rival keeps EV",
			header:  "| Humanoid, Rival | - | Level 2 | Elite Controller | EV 16 |",
			wantEV:  "16",
			wantOrg: "Elite", wantRole: "Controller", wantLvl: 2,
			wantKW: []string{"Humanoid", "Rival"},
		},
		{
			name:    "retainer has neither",
			header:  "| Devil, Infernal | - | Level 1 | Retainer Controller | - |",
			wantOrg: "Retainer", wantRole: "Controller", wantLvl: 1,
			wantKW: []string{"Devil", "Infernal"},
		},
		{
			name:     "rival minion malice cost",
			header:   "| Undead | - | - | Minion Defender | 2 Malice for two minions |",
			wantCost: "2 Malice for two minions",
			wantOrg:  "Minion", wantRole: "Defender",
			wantKW: []string{"Undead"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grid := tc.header + "\n|:-:|:-:|:-:|:-:|:-:|\n"
			got := parseStatGrid(grid).header
			if got.ev != tc.wantEV {
				t.Errorf("ev: got %q, want %q", got.ev, tc.wantEV)
			}
			if got.cost != tc.wantCost {
				t.Errorf("cost: got %q, want %q", got.cost, tc.wantCost)
			}
			if got.organization != tc.wantOrg {
				t.Errorf("org: got %q, want %q", got.organization, tc.wantOrg)
			}
			if got.role != tc.wantRole {
				t.Errorf("role: got %q, want %q", got.role, tc.wantRole)
			}
			if got.level != tc.wantLvl {
				t.Errorf("level: got %d, want %d", got.level, tc.wantLvl)
			}
			if !reflect.DeepEqual(got.keywords, tc.wantKW) {
				t.Errorf("keywords: got %v, want %v", got.keywords, tc.wantKW)
			}
		})
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

// A feature title may carry scc links in name-position: a linked word in the name
// ("[Solo](scc:…) Monster"), a linked Malice cost ("(2 [Malice](scc:…))"), or a
// linked "(Signature Ability)". The structured name/cost/ability_type fields must be
// link-free (display text only). Critically, the markdown link's own ")" must not
// break the cost-paren split (sbParenRe).
func TestParseStatblockFeature_LinkedTitle_FieldsAreLinkFree(t *testing.T) {
	// Linked Malice cost.
	cost := ParseStatblockFeatures("> 🏹 **Devilish Charm (2 [Malice](scc:mcdm.monsters.v1/rule.monster/malice))**\n>\n> The target is charmed.\n")
	if len(cost) != 1 {
		t.Fatalf("cost feature: got %d, want 1", len(cost))
	}
	if cost[0]["name"] != "Devilish Charm" {
		t.Errorf("name = %v, want 'Devilish Charm' (link + cost paren stripped from name)", cost[0]["name"])
	}
	if cost[0]["cost"] != "2 Malice" {
		t.Errorf("cost = %v, want '2 Malice' (linked Malice must split + strip; the link's ) must not break sbParenRe)", cost[0]["cost"])
	}

	// Linked word in the name proper.
	solo := ParseStatblockFeatures("> ☠️ **[Solo](scc:mcdm.monsters.v1/rule.organization/solo) Monster**\n>\n> Acts alone.\n")
	if solo[0]["name"] != "Solo Monster" {
		t.Errorf("name = %v, want 'Solo Monster' (link stripped from name)", solo[0]["name"])
	}

	// Linked "(Signature Ability)" must still classify as ability_type, link-free.
	sig := ParseStatblockFeatures("> 🗡 **Blade of the Gol King ([Signature Ability](scc:mcdm.heroes.v1/rule.combat/signature-ability))**\n>\n> **Power Roll + 4:**\n>\n> - **≤11:** 5 damage\n> - **12-16:** 8 damage\n> - **17+:** 11 damage\n")
	if sig[0]["name"] != "Blade of the Gol King" {
		t.Errorf("name = %v, want 'Blade of the Gol King'", sig[0]["name"])
	}
	if sig[0]["ability_type"] != "Signature Ability" {
		t.Errorf("ability_type = %v, want 'Signature Ability' (link-free)", sig[0]["ability_type"])
	}
}

// The ability-table cells (keywords / usage / distance / target) are structured
// fields and must be link-free even when the source links a term inside them
// (e.g. a linked "Triggered Action" in the usage cell).
func TestParseStatblockFeature_LinkedTableCells_AreLinkFree(t *testing.T) {
	block := "> 🩸 **Blood Drain**\n" +
		">\n" +
		"> | **[Magic](scc:mcdm.heroes.v1/x), Strike** | **Free [triggered action](scc:mcdm.heroes.v1/rule.combat/triggered-action)** |\n" +
		"> |---|---|\n" +
		"> | **📏 [Melee](scc:mcdm.heroes.v1/x) 1** | **🎯 One creature** |\n" +
		">\n" +
		"> The target is [bleeding](scc:mcdm.heroes.v1/condition/bleeding).\n"
	got := ParseStatblockFeatures(block)
	if len(got) != 1 {
		t.Fatalf("got %d, want 1", len(got))
	}
	if u, _ := got[0]["usage"].(string); strings.Contains(u, "scc:") {
		t.Errorf("usage = %q, want link-free", u)
	}
	if d, _ := got[0]["distance"].(string); strings.Contains(d, "scc:") {
		t.Errorf("distance = %q, want link-free", d)
	}
	for _, k := range got[0]["keywords"].([]string) {
		if strings.Contains(k, "scc:") {
			t.Errorf("keyword %q contains a link, want link-free", k)
		}
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
