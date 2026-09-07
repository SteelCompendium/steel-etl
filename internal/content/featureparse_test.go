package content

import (
	"strings"
	"testing"
)

func TestParseRichFeatures_Passive(t *testing.T) {
	body := "Intro prose line.\n\n" +
		"> 🔳 **Walleye (7 Malice)**\n" +
		">\n" +
		"> A basilisk spews reflective spittle across an adjacent vertical surface in a 3-square-by-3-square area. The basilisk can use their Petrifying Eye Beams ability to target a square in the area, causing the area and distance of that ability to become a 20 x 3 line within 1 square of the wall.\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	if f.Icon != "🔳" {
		t.Errorf("Icon = %q, want 🔳", f.Icon)
	}
	if f.Name != "Walleye" {
		t.Errorf("Name = %q, want Walleye", f.Name)
	}
	if f.Cost != "7 Malice" {
		t.Errorf("Cost = %q, want '7 Malice'", f.Cost)
	}
	if want := "A basilisk spews reflective spittle"; len(f.Body) == 0 || f.Body[:len(want)] != want {
		t.Errorf("Body = %q, want prefix %q", f.Body, want)
	}
	if f.PowerRoll != nil || f.Usage != "" || len(f.Sections) != 0 {
		t.Errorf("passive feature should have no PowerRoll/Usage/Sections: %+v", f)
	}
}

// A test-based feature (no spec table) leads with intro prose that sets up the
// roll, then the tier list: "As a maneuver, … can make a Might test." then ≤11 /
// 12-16 / 17+. That lead-in must land in Intro (rendered ABOVE the power roll),
// not Body (rendered below it). Verbatim shape: Pavise Shield's Deactivate.
func TestParseRichFeatures_IntroBeforeTiers(t *testing.T) {
	body := "> 🌀 **Deactivate**\n" +
		">\n" +
		"> As a maneuver, a creature adjacent to a pavise shield controlled by another creature can make a **Might test**.\n" +
		">\n" +
		"> - **≤11:** The creature controlling the shield retains control of it.\n" +
		"> - **12-16:** The creature controlling the shield retains control of it.\n" +
		"> - **17+:** The creature making the test grabs the shield.\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	if f.PowerRoll == nil {
		t.Fatalf("PowerRoll = nil, want a bare-test power roll")
	}
	want := "As a maneuver, a creature adjacent to a pavise shield controlled by another creature can make a **Might test**."
	if f.Intro != want {
		t.Errorf("Intro = %q, want %q", f.Intro, want)
	}
	if f.Body != "" {
		t.Errorf("Body = %q, want empty (lead-in prose belongs in Intro)", f.Body)
	}
}

func TestParseRichFeatures_SignatureCost(t *testing.T) {
	body := "> 🗡 **Blade of the Gol King (Signature Ability)**\n>\n> Some text.\n"
	feats := ParseRichFeatures(body)
	if len(feats) != 1 || feats[0].Name != "Blade of the Gol King" || feats[0].Cost != "Signature" {
		t.Fatalf("got %+v, want name 'Blade of the Gol King' cost 'Signature'", feats)
	}
}

// A malice-feature title may link the Malice cost: "**Solo Action (5 [Malice](scc:…))**".
// The name/cost must be link-free (a markdown link's own ")" otherwise breaks sbParenRe,
// leaving the whole linked paren stuck in the name field).
func TestParseRichFeatures_LinkedMaliceCost(t *testing.T) {
	body := "> ☠️ **Solo Action (5 [Malice](scc:mcdm.monsters.v1/rule.monster/malice))**\n>\n> The creature takes an additional turn.\n"
	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	if feats[0].Name != "Solo Action" {
		t.Errorf("name = %q, want 'Solo Action' (linked cost stripped from name)", feats[0].Name)
	}
	if feats[0].Cost != "5 Malice" {
		t.Errorf("cost = %q, want '5 Malice' (link-free)", feats[0].Cost)
	}
}

func TestParseRichFeatures_AbilityWithTableAndTiers(t *testing.T) {
	body := "> 🔳 **Upchuck (5 Malice)**\n" +
		">\n" +
		">\n" +
		"> | **Area, Weapon**        |               **Main action** |\n" +
		"> |-------------------------|------------------------------:|\n" +
		"> | **📏 3 cube within 10** | **🎯 Each enemy in the area** |\n" +
		">\n" +
		"> **Effect:** The basilisk spits up a chunk of partly digested stone.\n" +
		">\n" +
		"> **Power Roll + 2:**\n" +
		">\n" +
		"> - **≤11:** 4 damage\n" +
		"> - **12-16:** 4 damage; A < 1 2 damage, prone\n" +
		"> - **17+:** 4 damage; A < 2 5 damage, prone and can't stand (save ends)\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	if f.Name != "Upchuck" || f.Cost != "5 Malice" {
		t.Errorf("name/cost = %q/%q", f.Name, f.Cost)
	}
	if got := strings.Join(f.Keywords, ","); got != "Area,Weapon" {
		t.Errorf("Keywords = %q, want Area,Weapon", got)
	}
	if f.Usage != "Main action" {
		t.Errorf("Usage = %q, want 'Main action'", f.Usage)
	}
	if f.Distance != "3 cube within 10" {
		t.Errorf("Distance = %q", f.Distance)
	}
	if f.Target != "Each enemy in the area" {
		t.Errorf("Target = %q", f.Target)
	}
	if len(f.Sections) != 1 || f.Sections[0].Label != "Effect" {
		t.Fatalf("Sections = %+v, want one Effect section", f.Sections)
	}
	if f.PowerRoll == nil || f.PowerRoll.Formula != "+ 2" {
		t.Fatalf("PowerRoll = %+v, want formula '+ 2'", f.PowerRoll)
	}
	if f.PowerRoll.Tiers["low"] != "4 damage" {
		t.Errorf("low tier = %q", f.PowerRoll.Tiers["low"])
	}
	if f.PowerRoll.Tiers["mid"] != "4 damage; A < 1 2 damage, prone" {
		t.Errorf("mid tier = %q", f.PowerRoll.Tiers["mid"])
	}
	if f.PowerRoll.Tiers["high"] != "4 damage; A < 2 5 damage, prone and can't stand (save ends)" {
		t.Errorf("high tier = %q", f.PowerRoll.Tiers["high"])
	}
	if f.Body != "" {
		t.Errorf("ability with table should use Trailing, not Body: %q", f.Body)
	}
}

func TestParseRichFeatures_Enhancement(t *testing.T) {
	body := "> 🗡 **Blade of the Gol King (Signature Ability)**\n" +
		">\n" +
		"> | **Charge, Magic, Melee, Strike, Weapon** |                 **Main Action** |\n" +
		"> |------------------------------------------|--------------------------------:|\n" +
		"> | **📏 Melee 1**                           | **🎯 Two creatures or objects** |\n" +
		">\n" +
		"> **Effect:** Ajax shifts up to 2 squares between striking each target.\n" +
		">\n" +
		"> **1+ Malice:** Ajax can strike one additional target for each Malice spent.\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	if len(f.Enhancements) != 1 {
		t.Fatalf("Enhancements = %+v, want 1", f.Enhancements)
	}
	if f.Enhancements[0].Cost != "1+ Malice" {
		t.Errorf("enhancement cost = %q", f.Enhancements[0].Cost)
	}
	if len(f.Sections) != 1 || f.Sections[0].Label != "Effect" {
		t.Errorf("Sections = %+v", f.Sections)
	}
}

func TestParseRichFeatures_MultipleBlocks(t *testing.T) {
	body := "Intro.\n\n" +
		"> 👤 **Reason (2 Malice)**\n>\n> Opposed Reason test text.\n" +
		"\n" +
		"> ☠️ **Solo Action (5 Malice)**\n>\n> Ajax takes an additional main action on his turn.\n"
	feats := ParseRichFeatures(body)
	if len(feats) != 2 {
		t.Fatalf("got %d features, want 2", len(feats))
	}
	if feats[0].Name != "Reason" || feats[1].Name != "Solo Action" {
		t.Errorf("names = %q, %q", feats[0].Name, feats[1].Name)
	}
}

func TestParseRichFeatures_LevelLabels(t *testing.T) {
	body := "> ⭐️ **Hunger Thrush**\n>\n> Base feature text.\n" +
		"\n" +
		"> **Level 5 Fixture Advancement Feature**\n" +
		">\n" +
		"> ⭐️ **Soul Rancor**\n" +
		">\n" +
		"> You gain a surge.\n" +
		"\n" +
		"> **Level 9 Fixture Advancement Feature**\n" +
		">\n" +
		"> ⭐️ **Size Increase**\n" +
		">\n" +
		"> The boil is now size 3.\n" +
		">\n" +
		"> ⭐️ **Fester Field**\n" +
		">\n" +
		"> Each non-abyssal enemy takes 5 corruption damage.\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 4 {
		t.Fatalf("got %d features, want 4 (label blocks are not features)", len(feats))
	}
	wantLevels := map[string]int{
		"Hunger Thrush": 0, "Soul Rancor": 5, "Size Increase": 9, "Fester Field": 9,
	}
	for _, f := range feats {
		if f.Level != wantLevels[f.Name] {
			t.Errorf("%s: Level = %d, want %d", f.Name, f.Level, wantLevels[f.Name])
		}
	}
}

func TestParseRichFeatures_DiceInTitle(t *testing.T) {
	body := "> 🏹 **Hurl Bone 2d10 + [R](scc:mcdm.heroes.v1/rule.characteristic/reason)**\n" +
		">\n" +
		"> | **Ranged, Strike** |        **Main action** |\n" +
		"> |--------------------|------------------------:|\n" +
		"> | **📏 Ranged 5**    | **🎯 One creature** |\n" +
		">\n" +
		"> 2 damage\n" +
		">\n" +
		"> 4 damage\n" +
		">\n" +
		"> 6 damage\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	if f.Name != "Hurl Bone" {
		t.Errorf("Name = %q, want 'Hurl Bone'", f.Name)
	}
	if f.PowerRoll == nil || f.PowerRoll.Formula != "2d10 + R" {
		t.Fatalf("PowerRoll = %+v, want formula '2d10 + R' (link stripped)", f.PowerRoll)
	}
	if f.PowerRoll.Tiers["low"] != "2 damage" || f.PowerRoll.Tiers["mid"] != "4 damage" || f.PowerRoll.Tiers["high"] != "6 damage" {
		t.Errorf("tiers = %+v", f.PowerRoll.Tiers)
	}
}

func TestParseRichFeatures_DiceInTitleWithCost(t *testing.T) {
	// Real summoner signature grammar: bare characteristic + trailing
	// "(Signature Ability)" cost. Verbatim shape from the Summoner book.
	body := "> 🗡 **Mind Twist 2d10 + R (Signature Ability)**\n" +
		">\n" +
		"> | **Magic, Ranged, Strike** |        **Main action** |\n" +
		"> |---------------------------|------------------------:|\n" +
		"> | **📏 Ranged 10**          | **🎯 One creature** |\n" +
		">\n" +
		"> 2 psychic damage\n" +
		">\n" +
		"> 5 psychic damage\n" +
		">\n" +
		"> 7 psychic damage\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	if f.Name != "Mind Twist" {
		t.Errorf("Name = %q, want 'Mind Twist'", f.Name)
	}
	if f.Cost != "Signature" {
		t.Errorf("Cost = %q, want 'Signature'", f.Cost)
	}
	if f.PowerRoll == nil || f.PowerRoll.Formula != "2d10 + R" {
		t.Fatalf("PowerRoll = %+v, want formula '2d10 + R'", f.PowerRoll)
	}
	if f.PowerRoll.Tiers["low"] != "2 psychic damage" || f.PowerRoll.Tiers["mid"] != "5 psychic damage" || f.PowerRoll.Tiers["high"] != "7 psychic damage" {
		t.Errorf("tiers = %+v", f.PowerRoll.Tiers)
	}
}

func TestRichFeature_ToMap(t *testing.T) {
	f := RichFeature{
		Icon: "🔳", Name: "Upchuck", Cost: "5 Malice", Usage: "Main action",
		Keywords: []string{"Area", "Weapon"},
		Distance: "3 cube within 10", Target: "Each enemy in the area",
		PowerRoll:    &RichPowerRoll{Formula: "+ 2", Tiers: map[string]string{"low": "4 damage"}},
		Sections:     []RichSection{{Label: "Effect", Text: "Spits a stone."}},
		Enhancements: []RichEnhancement{{Cost: "2 Malice", Text: "More."}},
		Level:        5,
	}
	m := f.ToMap()
	if m["name"] != "Upchuck" || m["icon"] != "🔳" || m["cost"] != "5 Malice" {
		t.Errorf("scalars wrong: %+v", m)
	}
	pr, ok := m["power_roll"].(map[string]any)
	if !ok || pr["formula"] != "+ 2" {
		t.Fatalf("power_roll = %+v", m["power_roll"])
	}
	secs, ok := m["sections"].([]map[string]any)
	if !ok || len(secs) != 1 || secs[0]["label"] != "Effect" {
		t.Fatalf("sections = %+v", m["sections"])
	}
	if m["level"] != 5 {
		t.Errorf("level = %v, want 5", m["level"])
	}

	// Empty fields are omitted entirely.
	min := RichFeature{Name: "Walleye", Body: "Text."}
	mm := min.ToMap()
	for _, absent := range []string{"icon", "cost", "usage", "keywords", "distance",
		"target", "power_roll", "sections", "enhancements", "trailing", "level"} {
		if _, ok := mm[absent]; ok {
			t.Errorf("empty field %q should be omitted", absent)
		}
	}
	if mm["body"] != "Text." {
		t.Errorf("body = %v", mm["body"])
	}
}

// --- SC-308: multi-roll features (a second, later tier list must not silently
// overwrite the first) ---

// Snackies-shaped: a labeled roll, then a Special section whose prose names an
// "Agility test", then a header-less second list. Both rolls must survive, the
// first keeping its usual PowerRoll slot, and the second recorded in Post
// (labeled "Agility Test", derived from the Special section it follows)
// immediately after that Special section — not overwriting the first.
func TestParseRichFeatures_MultiRoll_Snackies(t *testing.T) {
	body := "> ☠️ **Snackies for Sweeties (Villain Action 1)**\n" +
		">\n" +
		"> | **Area, Magic** |                            **-** |\n" +
		"> |-----------------|---------------------------------:|\n" +
		"> | **📏 5 burst**  | **🎯 Each creature in the area** |\n" +
		">\n" +
		"> **Effect:** The hag attaches an ornate explosive pastry to each target.\n" +
		">\n" +
		"> **Power Roll + 3:**\n" +
		">\n" +
		"> - **≤11:** 6 poison damage\n" +
		"> - **12-16:** 10 poison damage\n" +
		"> - **17+:** 13 poison damage\n" +
		">\n" +
		"> **Special:** A creature wearing a pastry can attempt an **Agility test** to remove the pastry as a maneuver.\n" +
		">\n" +
		"> - **≤11:** The hag makes the power roll for all pastries.\n" +
		"> - **12-16:** The pastry is not removed.\n" +
		"> - **17+:** The pastry is removed and can no longer explode.\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]

	if f.PowerRoll == nil || f.PowerRoll.Formula != "+ 3" || f.PowerRoll.Tiers["low"] != "6 poison damage" {
		t.Fatalf("first PowerRoll = %+v, want formula '+ 3' with the poison tiers", f.PowerRoll)
	}
	if f.PowerRoll.Label != "" {
		t.Errorf("first PowerRoll.Label = %q, want empty (it has its own header)", f.PowerRoll.Label)
	}

	// Post must be [Effect section, Special section, second roll] in document
	// order — the second roll attached immediately after the Special section it
	// follows (not the Effect section, which precedes the first roll).
	if len(f.Post) != 3 {
		t.Fatalf("Post = %+v, want 3 blocks (Effect section, Special section, second roll)", f.Post)
	}
	if f.Post[0].Section == nil || f.Post[0].Section.Label != "Effect" {
		t.Fatalf("Post[0] = %+v, want the Effect section", f.Post[0])
	}
	if f.Post[1].Section == nil || f.Post[1].Section.Label != "Special" {
		t.Fatalf("Post[1] = %+v, want the Special section", f.Post[1])
	}
	second := f.Post[2].Roll
	if second == nil {
		t.Fatalf("Post[2] = %+v, want the second roll", f.Post[2])
	}
	if second.Label != "Agility Test" {
		t.Errorf("second roll Label = %q, want 'Agility Test'", second.Label)
	}
	if second.Formula != "" {
		t.Errorf("second roll Formula = %q, want empty (header-less)", second.Formula)
	}
	if second.Tiers["low"] != "The hag makes the power roll for all pastries." {
		t.Errorf("second roll tiers = %+v", second.Tiers)
	}

	// ToMap must carry both: the first roll under power_roll, the second inside
	// the ordered `post` sequence (SC-308) — nothing silently dropped.
	m := f.ToMap()
	post, ok := m["post"].([]map[string]any)
	if !ok || len(post) != 3 {
		t.Fatalf("ToMap post = %+v, want 3 entries", m["post"])
	}
	roll2, ok := post[2]["roll"].(map[string]any)
	if !ok || roll2["label"] != "Agility Test" {
		t.Fatalf("ToMap post[2].roll = %+v, want label 'Agility Test'", post[2]["roll"])
	}
}

// No Escape-shaped: two LABELED power rolls (each its own "**Power Roll + N:**"
// header) with a bare prose paragraph between them. Neither roll derives a
// label from prose (both already have an explicit header); the prose lands in
// Post between the two rolls, matching document order.
func TestParseRichFeatures_MultiRoll_NoEscape(t *testing.T) {
	body := "> ☠️ **No Escape (Villain Action 3)**\n" +
		">\n" +
		"> | **Ranged**       |                           **-** |\n" +
		"> |------------------|--------------------------------:|\n" +
		"> | **📏 Ranged 10** | **🎯 Two creatures or objects** |\n" +
		">\n" +
		"> **Effect:** The cryptic makes an initial power roll that calls down stone pillars from the ceiling.\n" +
		">\n" +
		"> **Power Roll + 3:**\n" +
		">\n" +
		"> - **≤11:** 5 damage; prone\n" +
		"> - **12-16:** 9 damage; prone\n" +
		"> - **17+:** 12 damage; prone\n" +
		">\n" +
		"> The cryptic then makes a second power roll that raises stone pillars from the floor.\n" +
		">\n" +
		"> **Power Roll + 3:**\n" +
		">\n" +
		"> - **≤11:** 2 damage; vertical slide 2\n" +
		"> - **12-16:** 3 damage; vertical slide 4\n" +
		"> - **17+:** 4 damage; vertical slide 6\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]

	if f.PowerRoll == nil || f.PowerRoll.Tiers["low"] != "5 damage; prone" {
		t.Fatalf("first PowerRoll = %+v", f.PowerRoll)
	}
	if len(f.Post) != 3 {
		t.Fatalf("Post = %+v, want 3 blocks (Effect section, prose, second roll)", f.Post)
	}
	if f.Post[0].Section == nil || f.Post[0].Section.Label != "Effect" {
		t.Fatalf("Post[0] = %+v, want the Effect section", f.Post[0])
	}
	if f.Post[1].Prose == "" {
		t.Fatalf("Post[1] = %+v, want the prose paragraph between the two rolls", f.Post[1])
	}
	second := f.Post[2].Roll
	if second == nil {
		t.Fatalf("Post[2] = %+v, want the second roll", f.Post[2])
	}
	if second.Formula != "+ 3" {
		t.Errorf("second roll Formula = %q, want '+ 3' (it has its own header)", second.Formula)
	}
	if second.Label != "" {
		t.Errorf("second roll Label = %q, want empty (a headered roll doesn't derive a label)", second.Label)
	}
	if second.Tiers["low"] != "2 damage; vertical slide 2" {
		t.Errorf("second roll tiers = %+v", second.Tiers)
	}
}

// Roll-the-Wheel-shaped: a labeled first roll, then a bare second list with no
// preceding "**<Characteristic> test**" phrase — it must render bare (no
// derived label), per the existing bare-test convention.
func TestParseRichFeatures_MultiRoll_BareSecondPanel(t *testing.T) {
	body := "> 🌀 **Roll the Wheel**\n" +
		">\n" +
		"> | **Area**       |                  **Main action** |\n" +
		"> |----------------|----------------------------------:|\n" +
		"> | **📏 Special** | **🎯 Each creature in the area** |\n" +
		">\n" +
		"> **Effect:** The wheel rolls, moving 2 squares in a straight line.\n" +
		">\n" +
		"> **Power Roll + 2:**\n" +
		">\n" +
		"> - **≤11:** 5 damage; push 1\n" +
		"> - **12-16:** 9 damage; push 2\n" +
		"> - **17+:** 12 damage; push 3\n" +
		">\n" +
		"> If the wheel is reduced to 0 Stamina, its movement stops and it explodes.\n" +
		">\n" +
		"> - **≤11:** 5 damage; push 1\n" +
		"> - **12-16:** 9 damage; push 2\n" +
		"> - **17+:** 12 damage; push 3\n" +
		">\n" +
		"> A burning creature takes 1d6 fire damage at the start of each of their turns.\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]

	if len(f.Post) != 4 {
		t.Fatalf("Post = %+v, want 4 blocks (Effect section, prose, second roll, prose)", f.Post)
	}
	second := f.Post[2].Roll
	if second == nil {
		t.Fatalf("Post[2] = %+v, want the second roll", f.Post[2])
	}
	if second.Label != "" || second.Formula != "" {
		t.Errorf("second roll = %+v, want fully bare (no preceding test phrase to derive from)", second)
	}
}

// Overpower-shaped: no spec table at all; BOTH tier lists are header-less,
// each preceded by its own prose paragraph naming "**Reason test**". Both
// rolls derive the same label — the first keeps its PowerRoll slot (with the
// derived label, since the ruling doesn't exempt the first list of a
// multi-list feature), the second lands in Post.
func TestParseRichFeatures_MultiRoll_Overpower(t *testing.T) {
	body := "> 🌀 **Overpower (7 Malice)**\n" +
		">\n" +
		"> Lord Syuul sends out a psionic burst. He makes a **Reason test** (2d10 + 4).\n" +
		">\n" +
		"> - **≤11:** Lord Syuul has damage weakness 5.\n" +
		"> - **12-16:** Lord Syuul has damage immunity 2.\n" +
		"> - **17+:** Lord Syuul has damage immunity 5.\n" +
		">\n" +
		"> Whenever an Overpower effect is active, a hero can push back by making a **Reason test**.\n" +
		">\n" +
		"> - **≤11:** Lord Syuul has damage immunity 5.\n" +
		"> - **12-16:** Lord Syuul has damage immunity 2.\n" +
		"> - **17+:** Lord Syuul has damage weakness 5.\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]

	if f.PowerRoll == nil || f.PowerRoll.Label != "Reason Test" {
		t.Fatalf("first PowerRoll = %+v, want Label 'Reason Test'", f.PowerRoll)
	}
	if want := "Lord Syuul sends out a psionic burst. He makes a **Reason test** (2d10 + 4)."; f.Intro != want {
		t.Errorf("Intro = %q, want %q", f.Intro, want)
	}
	// Post = [prose lead-in to the second test, second roll].
	if len(f.Post) != 2 || f.Post[1].Roll == nil || f.Post[1].Roll.Label != "Reason Test" {
		t.Fatalf("Post = %+v, want [prose, second roll labeled 'Reason Test']", f.Post)
	}
}

// TestDeriveTestLabel_UsesLastPhrase locks the SC-308 review's INFO-2 rule: a
// paragraph naming two characteristics picks the LAST one — the one nearest
// the tier list that follows — not the first.
func TestDeriveTestLabel_UsesLastPhrase(t *testing.T) {
	got := DeriveTestLabel("Each target must make either a **Might test** or an **Agility test**.")
	if got != "Agility Test" {
		t.Errorf("DeriveTestLabel = %q, want 'Agility Test' (the last-named characteristic)", got)
	}
}
