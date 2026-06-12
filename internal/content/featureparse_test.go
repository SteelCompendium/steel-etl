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

func TestParseRichFeatures_SignatureCost(t *testing.T) {
	body := "> 🗡 **Blade of the Gol King (Signature Ability)**\n>\n> Some text.\n"
	feats := ParseRichFeatures(body)
	if len(feats) != 1 || feats[0].Name != "Blade of the Gol King" || feats[0].Cost != "Signature" {
		t.Fatalf("got %+v, want name 'Blade of the Gol King' cost 'Signature'", feats)
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
