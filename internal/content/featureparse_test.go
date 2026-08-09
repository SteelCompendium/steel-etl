package content

import (
	"os"
	"reflect"
	"strconv"
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
		PowerRoll: &RichPowerRoll{Formula: "+ 2", Tiers: map[string]string{"low": "4 damage"}},
		Sections:  []RichSection{{Label: "Effect", Text: "Spits a stone."}},
		Enhancements: []RichEnhancement{{Cost: "2 Malice", Text: "More."}},
		Level: 5,
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

// TestParseRichFeatures_FlagLabel: a standalone bold flag ("**Champion Action**")
// is promoted to a labelled section in document order — ABOVE the Effect it
// governs — rather than falling through to bare prose that trails the card.
func TestParseRichFeatures_FlagLabel(t *testing.T) {
	body := "> ❗️ **Reality Flense**\n" +
		">\n" +
		"> | **—** | **1 Eidos** |\n" +
		"> |-------|------------:|\n" +
		"> | **📏 20 burst** | **🎯 Self and each non-minion ally in the area** |\n" +
		">\n" +
		"> **Champion Action**\n" +
		">\n" +
		"> **Effect:** Each target teleports up to their speed.\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	want := []RichSection{
		{Label: "Champion Action", Text: ""},
		{Label: "Effect", Text: "Each target teleports up to their speed."},
	}
	if !reflect.DeepEqual(f.Sections, want) {
		t.Errorf("Sections = %+v, want %+v", f.Sections, want)
	}
	if f.Trailing != "" {
		t.Errorf("Trailing = %q, want empty (the flag must not trail the card)", f.Trailing)
	}
	// The flag survives into the data formats as a text-less section, so the
	// featureblock schema's required "text" key is still present.
	secs, ok := f.ToMap()["sections"].([]map[string]any)
	if !ok || len(secs) != 2 {
		t.Fatalf("ToMap sections = %+v", f.ToMap()["sections"])
	}
	if secs[0]["label"] != "Champion Action" || secs[0]["text"] != "" {
		t.Errorf("ToMap flag section = %+v", secs[0])
	}
}

// TestParseRichFeatures_FlagLabelVocabularyIsClosed: a bold-only paragraph that
// is NOT in fbFlagLabels keeps its old handling (bare prose). Guards the
// SC-137 failure mode in the other direction — the promotion must not become a
// shape rule that silently restructures unrelated content.
func TestParseRichFeatures_FlagLabelVocabularyIsClosed(t *testing.T) {
	body := "> ❗️ **Reality Flense**\n" +
		">\n" +
		"> | **—** | **1 Eidos** |\n" +
		"> |-------|------------:|\n" +
		"> | **📏 20 burst** | **🎯 Self** |\n" +
		">\n" +
		"> **Not A Known Flag**\n" +
		">\n" +
		"> **Effect:** Text.\n"

	feats := ParseRichFeatures(body)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	if len(f.Sections) != 1 || f.Sections[0].Label != "Effect" {
		t.Errorf("Sections = %+v, want only Effect", f.Sections)
	}
	if f.Trailing != "**Not A Known Flag**" {
		t.Errorf("Trailing = %q, want the unpromoted bold paragraph", f.Trailing)
	}
}

// wantFlagPromotions is the exact, corpus-wide set of source lines that
// fbFlagLabels promotes: "<book>:<line>" → the canonical label. Keyed by book
// directory rather than full path so the guard reads as a content inventory.
var wantFlagPromotions = map[string]string{
	// The four Portfolio Champions' Level 10 abilities (Summoner, SC-138).
	"summoner:2819": "Champion Action", // Reality Flense (Demon Lord's Aspect)
	"summoner:2888": "Champion Action", // A Breath Felt in a Hurricane (Dragon's Portent)
	"summoner:2961": "Champion Action", // A Shower of Dust (Celestial Attendant)
	"summoner:3034": "Champion Action", // Gravemaker (Avatar of Death)
}

// TestBookSources_FlagLabelPromotionSet pins the blast radius of the
// fbFlagLabels promotion (SC-138). Every standalone bold blockquote line in all
// four books is scanned; the ones the vocabulary promotes must be exactly
// wantFlagPromotions. A new book line that happens to read "**Champion
// Action**" — or a new vocabulary entry — fails here rather than silently
// restructuring a rendered ability.
func TestBookSources_FlagLabelPromotionSet(t *testing.T) {
	books := map[string]string{
		"heroes":     "../../input/heroes/Draw Steel Heroes.md",
		"monsters":   "../../input/monsters/Draw Steel Monsters.md",
		"beastheart": "../../input/beastheart/Draw Steel Beastheart.md",
		"summoner":   "../../input/summoner/Draw Steel Summoner.md",
	}
	got := map[string]string{}
	scanned := 0
	for book, path := range books {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Skipf("skipping corpus guard: %v", err)
		}
		for i, raw := range strings.Split(string(data), "\n") {
			line := strings.TrimSpace(raw)
			if !strings.HasPrefix(line, ">") {
				continue
			}
			line = strings.TrimSpace(strings.TrimLeft(line, "> "))
			m := fbFlagLabelRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			scanned++
			if label, ok := fbFlagLabels[strings.ToLower(strings.TrimSpace(m[1]))]; ok {
				got[book+":"+strconv.Itoa(i+1)] = label
			}
		}
	}
	if scanned < 100 {
		t.Errorf("only %d standalone bold blockquote lines scanned; the guard is not seeing the corpus", scanned)
	}
	if !reflect.DeepEqual(got, wantFlagPromotions) {
		t.Errorf("flag promotions changed corpus-wide.\n got: %v\nwant: %v\n"+
			"\thint: update wantFlagPromotions only after checking the new lines really are labels", got, wantFlagPromotions)
	}
}
