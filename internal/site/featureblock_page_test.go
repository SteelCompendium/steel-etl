package site

import (
	"strings"
	"testing"
)

func TestRenderFbFeats_AdvancementBands(t *testing.T) {
	feats := []fbFeature{
		{Icon: "⭐️", Name: "Base One", Body: "always on"},
		{Icon: "⭐️", Name: "Base Two", Body: "also on"},
		{Icon: "⭐️", Name: "Tier Five", Body: "at L5", Level: 5},
		{Icon: "⭐️", Name: "Tier Nine A", Body: "at L9", Level: 9},
		{Icon: "⭐️", Name: "Tier Nine B", Body: "also L9", Level: 9},
	}
	got := renderFbFeats(feats)
	// base features are NOT in a band
	idxBase := strings.Index(got, "Base One")
	idxBand := strings.Index(got, `class="fb__band--adv"`)
	if idxBase == -1 || idxBand == -1 || idxBase > idxBand {
		t.Fatalf("base features must render before the first advancement band")
	}
	for _, want := range []string{
		`<div class="fb__band--adv" data-level="5">`,
		`<div class="fb__adv-head">Level 5 Advancement</div>`,
		"Tier Five",
		`<div class="fb__band--adv" data-level="9">`,
		`<div class="fb__adv-head">Level 9 Advancement</div>`,
		"Tier Nine A", "Tier Nine B",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q", want)
		}
	}
	// exactly two bands (one per level)
	if n := strings.Count(got, `class="fb__band--adv"`); n != 2 {
		t.Errorf("band count = %d, want 2", n)
	}
}

func TestRenderFbFeats_NoLevelsNoBands(t *testing.T) {
	// backward-compat: existing featureblock/terrain features (Level 0) → no band
	got := renderFbFeats([]fbFeature{{Icon: "⭐️", Name: "Flat", Body: "x"}})
	if strings.Contains(got, "fb__band--adv") {
		t.Error("Level-0 features must not emit an advancement band")
	}
}

const fbMalicePage = `---
name: Basilisk Malice
type: featureblock
kind: malice
flavor: At the start of any basilisk's turn, you can spend Malice to activate one of the following features.
features:
    - icon: "🔳"
      name: Walleye
      cost: 7 Malice
      body: A basilisk spews reflective spittle across an adjacent vertical surface.
---

At the start of any basilisk's turn, you can spend Malice to activate one of the following features.

> 🔳 **Walleye (7 Malice)**
>
> A basilisk spews reflective spittle across an adjacent vertical surface.
`

func TestBuildFeatureblockPage_NonFeatureblockPassesThrough(t *testing.T) {
	in := []byte("---\nname: Foo\ntype: ability\n---\n\nbody\n")
	out, ok := buildFeatureblockPage(in)
	if ok {
		t.Fatalf("ability page should not be handled by the featureblock renderer")
	}
	if string(out) != string(in) {
		t.Fatalf("non-featureblock data must be returned unchanged")
	}
}

func TestBuildFeatureblockPage_MaliceWrap(t *testing.T) {
	out, ok := buildFeatureblockPage([]byte(fbMalicePage))
	if !ok {
		t.Fatal("featureblock page should be handled")
	}
	s := string(out)
	// frontmatter preserved
	if !strings.HasPrefix(s, "---\n") || !strings.Contains(s, "type: featureblock") {
		t.Errorf("frontmatter not preserved:\n%s", s)
	}
	for _, want := range []string{
		`class="fb-wrap"`, `data-role="malice"`, `data-kind="malice"`,
		`class="fb md-typeset"`, `class="sc-head fb__head"`,
		`sc-head__left-eyebrow sc-head__slot--line">Malice</div>`,
		`sc-head__left-primary sc-head__slot--line">Basilisk Malice</h2>`,
		`class="fb__flavor"`, "spend Malice to activate",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

const fbTerrainPage = `---
name: Angry Beehive
type: dynamic-terrain
level: 2
terrain_type: Hazard
role: Hexer
flavor: This beehive is full of angry bees.
stats:
    - name: EV
      value: "2"
    - name: Stamina
      value: "3 per square"
features:
    - icon: "🌀"
      name: Deactivate
      body: The beehive can't be deactivated.
    - icon: "❗️"
      name: Your Fears Become Manifest
      usage: Main action
      keywords:
        - Area
        - Magic
      distance: 10 burst
      power_roll:
        formula: + 2
        tiers:
            low: P < 1 slowed (EoT)
            mid: P < 2 slowed and weakened (EoT)
            high: P < 3 frightened (EoT)
---

body
`

func TestRenderFbStats(t *testing.T) {
	out, ok := buildFeatureblockPage([]byte(fbTerrainPage))
	if !ok {
		t.Fatal("terrain page should be handled")
	}
	s := string(out)
	for _, want := range []string{
		`sc-head__right-eyebrow sc-head__slot--chip">Level 2</div>`,
		`sc-head__right-primary sc-head__slot--mini" data-role="hexer">Hazard Hexer</div>`,
		`class="fb__stats"`,
		`class="fb__stat"`, `class="fb__stat-l">EV<`, `class="fb__stat-v">2<`,
		`class="fb__stat-l">Stamina<`, "3 per square",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestRenderFbStats_EmptyWhenAbsent(t *testing.T) {
	out, _ := buildFeatureblockPage([]byte(fbMalicePage))
	if strings.Contains(string(out), `class="fb__stats"`) {
		t.Error("malice block has no stats; fb__stats container should be omitted")
	}
}

func TestRenderFbFeats_PassiveMalice(t *testing.T) {
	out, _ := buildFeatureblockPage([]byte(fbMalicePage))
	s := string(out)
	for _, want := range []string{
		`class="fb__feats"`,
		`class="sc-ability fb__feat" data-action="passive"`, // 🔳 → no usage/cost-table → passive
		`class="fb__feat-icon"`, "🔳",
		`sc-head__left-primary sc-head__slot--line">Walleye</h3>`,
		`sc-head__right-primary sc-head__slot--mini">7 Malice</div>`, // cost is now the right-primary mini
		`class="fb__feat-body"`, "reflective spittle",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestFeatureblockCard_EyebrowOverrideToDeck(t *testing.T) {
	// A synthetic Eyebrow (e.g. retainer advancement provenance) renders as the
	// left-deck provenance line; the kind-noun stays in the left-eyebrow.
	got := renderFeatureblockCard(fbDoc{Name: "X", Eyebrow: "Harrier Retainer", Kind: "advancement"})
	if !strings.Contains(got, `sc-head__left-deck sc-head__slot--line">Harrier Retainer</div>`) {
		t.Errorf("synthetic Eyebrow should render as left-deck:\n%s", got)
	}
	if !strings.Contains(got, `sc-head__left-eyebrow sc-head__slot--line">Advancement</div>`) {
		t.Errorf("kind-noun should read Advancement:\n%s", got)
	}
}

func TestRenderFbFeats_TerrainSpecialAndPowerRoll(t *testing.T) {
	out, _ := buildFeatureblockPage([]byte(fbTerrainPage))
	s := string(out)
	for _, want := range []string{
		`data-action="special"`, "Deactivate", // 🌀 → special (icon fallback, not passive)
		`data-action="main"`, "Your Fears Become Manifest", // usage "Main action" → main
		`class="sc-ability__chip">Area<`, `class="sc-ability__chip">Magic<`,
		`class="sc-ability__rail"`, "10 burst",
		`class="sc-ability__pr"`, "Power Roll", "+ 2",
		`class="sc-ability__tier" data-tier="low"`, "slowed",
		`class="sc-ability__tier" data-tier="high"`, "frightened",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

// fbFeatureAction's icon fallback must be deterministic even for an icon string
// that (however unexpectedly) contains more than one mapped glyph: fbIconAction
// used to be a map ranged with an early return on the first Contains match, so
// the winner depended on Go's per-iteration randomized map order — the same
// input could resolve to a different action from one call to the next within a
// single process. Regression for FOLLOWUPS #29 (deterministic output ordering).
func TestFbFeatureAction_CompoundIconIsDeterministic(t *testing.T) {
	feat := fbFeature{Icon: "🗡👤", Name: "Compound Icon"}
	first := fbFeatureAction(feat)
	for i := 0; i < 500; i++ {
		if got := fbFeatureAction(feat); got != first {
			t.Fatalf("fbFeatureAction(%q) not deterministic: got %q then %q on call %d", feat.Icon, first, got, i)
		}
	}
	// Priority order is 🗡 before 👤 in fbIconAction, so "main" always wins.
	if first != "main" {
		t.Fatalf("fbFeatureAction(%q) = %q, want %q (priority: 🗡 before 👤)", feat.Icon, first, "main")
	}
}

// Each single-glyph icon still resolves to its documented action (no behavior
// change from the map → ordered-slice conversion for the realistic case).
func TestFbFeatureAction_SingleIcon(t *testing.T) {
	cases := map[string]string{
		"🗡": "main", "🏹": "main", "❇": "main",
		"👤": "maneuver",
		"❗": "triggered", "❕": "triggered",
		"⭐": "passive",
		"☠": "villain",
		"🌀": "special",
	}
	for icon, want := range cases {
		got := fbFeatureAction(fbFeature{Icon: icon})
		if got != want {
			t.Errorf("fbFeatureAction(%q) = %q, want %q", icon, got, want)
		}
	}
}

// A feature's usage ("Main action (Adjacent creature)") must render as the
// right-deck chip. Regression for the Field Ballista's Reload/Spot, whose usage was
// parsed but only fed the data-action accent — never shown — leaving the cards bare.
func TestRenderFbFeat_UsageChip(t *testing.T) {
	feat := fbFeature{
		Icon: "⭐️", Name: "Reload", Usage: "Main action (Adjacent creature)",
		Sections: []fbSection{{Label: "Effect", Text: "The field ballista is reloaded."}},
	}
	s := renderFbFeats([]fbFeature{feat})
	if !strings.Contains(s, `sc-head__right-deck sc-head__slot--chip">Main action (Adjacent creature)</div>`) {
		t.Fatalf("usage should render as the right-deck chip in:\n%s", s)
	}
	if !strings.Contains(s, `sc-head__left-primary sc-head__slot--line">Reload</h3>`) {
		t.Fatalf("name should render as left-primary in:\n%s", s)
	}
}

// A feature with no usage (a passive/trait) must NOT emit an empty right-deck chip.
func TestRenderFbFeat_NoUsageNoChip(t *testing.T) {
	feat := fbFeature{Icon: "⭐️", Name: "Upgrades", Body: "Some passive prose."}
	s := renderFbFeats([]fbFeature{feat})
	if strings.Contains(s, "sc-head__right-deck") {
		t.Errorf("usage-less feature must not emit a right-deck chip in:\n%s", s)
	}
}

// Placeholder "-" keywords (Field Ballista's Reload/Spot) must not render an empty
// chip row; real keywords mixed with dashes keep only the real ones.
func TestRenderFbFeat_DashKeywordsDropped(t *testing.T) {
	dashOnly := renderFbFeats([]fbFeature{{Name: "Reload", Keywords: []string{"-"}}})
	if strings.Contains(dashOnly, "sc-ability__kw") {
		t.Errorf("dash-only keywords should drop the chip row:\n%s", dashOnly)
	}
	for _, dash := range []string{"-", "—", "–"} {
		s := renderFbFeats([]fbFeature{{Name: "Strike", Keywords: []string{"Ranged", dash, "Weapon"}}})
		if !strings.Contains(s, ">Ranged<") || !strings.Contains(s, ">Weapon<") {
			t.Errorf("real keywords dropped for dash %q:\n%s", dash, s)
		}
		if strings.Contains(s, ">"+dash+"<") {
			t.Errorf("dash %q kept as a chip:\n%s", dash, s)
		}
	}
}

// When BOTH Distance and Target are blank-or-dash the whole rail row is dropped;
// if either carries a real value the row stays (the dash cell shows an em-dash).
func TestRenderFbFeat_DashRailDropped(t *testing.T) {
	for _, dt := range [][2]string{{"-", "-"}, {"", "—"}, {"–", ""}} {
		s := renderFbFeats([]fbFeature{{Name: "Reload", Distance: dt[0], Target: dt[1]}})
		if strings.Contains(s, "sc-ability__rail") {
			t.Errorf("rail should drop for Distance=%q Target=%q:\n%s", dt[0], dt[1], s)
		}
	}
	// one real value keeps the row; the dash cell renders as an em-dash, not "-"
	s := renderFbFeats([]fbFeature{{Name: "Burst", Distance: "10 burst", Target: "-"}})
	if !strings.Contains(s, "sc-ability__rail") || !strings.Contains(s, "10 burst") {
		t.Fatalf("rail with one real value should render:\n%s", s)
	}
	railIdx := strings.Index(s, "sc-ability__rail")
	if strings.Contains(s[railIdx:], ">-<") {
		t.Errorf("dash Target should render as em-dash, not literal '-':\n%s", s)
	}
}

// A test feature's lead-in (Intro) must render ABOVE the power roll. Regression
// for Pavise Shield's Deactivate, whose "As a maneuver, … Might test." rendered
// below the tiers because it was stored as Body.
func TestRenderFbFeat_IntroAbovePowerRoll(t *testing.T) {
	feat := fbFeature{
		Icon: "🌀", Name: "Deactivate",
		Intro:     "As a maneuver, a creature can make a **Might test**.",
		PowerRoll: &fbPowerRoll{Tiers: map[string]string{"low": "retains control", "high": "grabs the shield"}},
	}
	s := renderFbFeats([]fbFeature{feat})
	if !strings.Contains(s, `class="fb__feat-intro"`) {
		t.Fatalf("missing fb__feat-intro in:\n%s", s)
	}
	idxIntro := strings.Index(s, `class="fb__feat-intro"`)
	idxPR := strings.Index(s, `class="sc-ability__pr"`)
	if idxPR < 0 || idxIntro > idxPR {
		t.Errorf("intro (%d) must render before power roll (%d):\n%s", idxIntro, idxPR, s)
	}
}

func TestFeatureblockCard_SixSlotHead(t *testing.T) {
	doc := fbDoc{
		Kind: "dynamic-terrain", Name: "Spike Pit", TerrainType: "Trap", Role: "Hazard", Level: 1,
		Stats: []fbStat{{Name: "EV", Value: "2"}, {Name: "Stamina", Value: "3 per square"}},
	}
	got := renderFeatureblockCard(doc)
	for _, want := range []string{
		`sc-head__left-eyebrow sc-head__slot--line">Dynamic Terrain</div>`,
		`sc-head__left-primary sc-head__slot--line">Spike Pit</h2>`,
		`sc-head__right-eyebrow sc-head__slot--chip">Level 1</div>`,
		`sc-head__right-primary sc-head__slot--mini" data-role="hazard">Trap Hazard</div>`,
		`sc-head__right-deck sc-head__slot--chip">EV 2</div>`,
		`class="fb__stats"`, // remaining loose stats still render in the body grid
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestFbOrigin_Fixture(t *testing.T) {
	if got := fbOrigin("mcdm.summoner.v1/monster.fixture.demon.featureblock/the-boil"); got != "Summoner · Demon" {
		t.Errorf("fbOrigin fixture = %q, want %q", got, "Summoner · Demon")
	}
	if got := fbOrigin("mcdm.monsters.v1/monster.basilisk.malice/x"); got != "" {
		t.Errorf("fbOrigin non-fixture should be empty, got %q", got)
	}
}
