package site

import (
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"gopkg.in/yaml.v3"
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
      effects:
          - effect: A basilisk spews reflective spittle across an adjacent vertical surface.
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
	if !strings.Contains(s, ` id="sc-feat-`) {
		t.Errorf("featureblock feature heads must carry sc-feat- ids (SC-306):\n%s", s)
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
      effects:
          - effect: The beehive can't be deactivated.
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
      effects:
          - roll: Power Roll + 2
            tier1: P < 1 slowed (EoT)
            tier2: P < 2 slowed and weakened (EoT)
            tier3: P < 3 frightened (EoT)
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
		`sc-head__left-primary sc-head__slot--line" id="sc-feat-walleye">Walleye</h3>`,
		`sc-head__right-primary sc-head__slot--mini">7 Malice</div>`, // cost is now the right-primary mini
		// SC-308 round 3b: a passive's single paragraph renders as an ordinary
		// nameless effects entry (.fb__feat-trailing), not the old dedicated
		// .fb__feat-body class — they shared the same base CSS declaration.
		`class="fb__feat-trailing"`, "reflective spittle",
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
	if !strings.Contains(s, `sc-head__left-primary sc-head__slot--line" id="sc-feat-reload">Reload</h3>`) {
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

// SC-308 round 3b: a test's lead-in prose ("As a maneuver, … Might test.")
// attaches to the tier list that directly follows it (the attachment rule) —
// ONE effects entry, prose then panel, inside the same fb__feat-trailing
// paragraph's sibling panel — no special .fb__feat-intro styling any more
// (.fb__feat-intro/.fb__feat-body/.fb__feat-trailing shared the same base CSS
// declaration; the render class no longer needs to distinguish before/after,
// since attachment already places the panel correctly). Regression for
// Pavise Shield's Deactivate.
func TestRenderFbFeat_ProseWithAttachedRoll(t *testing.T) {
	feat := fbFeature{
		Icon: "🌀", Name: "Deactivate",
		Effects: []fbEffect{{
			Effect: "As a maneuver, a creature can make a **Might test**.",
			Tier1:  "retains control", Tier3: "grabs the shield",
		}},
	}
	s := renderFbFeats([]fbFeature{feat})
	if strings.Contains(s, "fb__feat-intro") {
		t.Errorf("no fb__feat-intro class should remain in the SC-308 model:\n%s", s)
	}
	if !strings.Contains(s, `class="fb__feat-trailing">As a maneuver`) {
		t.Fatalf("missing the prose paragraph in:\n%s", s)
	}
	idxProse := strings.Index(s, "As a maneuver")
	idxPR := strings.Index(s, `class="sc-ability__pr"`)
	if idxPR < 0 || idxProse > idxPR {
		t.Errorf("prose (%d) must render before its attached power roll (%d):\n%s", idxProse, idxPR, s)
	}
	// Header-less: the site derives "Might Test" from this entry's own prose.
	if !strings.Contains(s, `<span class="pre">Might Test</span>`) {
		t.Errorf("expected a derived 'Might Test' head in:\n%s", s)
	}
}

// TestRenderFbFeat_NoEffectsFallsBackToFlatBody locks SC-308 review r3's C-2
// fix: not every fbFeature producer goes through parseRichFeature —
// collectChildFeatures (internal/content/monster.go, the beastheart companion
// advancement blocks) builds a {name, body, level}-only feature with no
// Effects at all. renderFbFeat must fall back to the flat fields instead of
// rendering a bare heading.
func TestRenderFbFeat_NoEffectsFallsBackToFlatBody(t *testing.T) {
	feat := fbFeature{Name: "Foes Forever Frozen", Body: "The gaze turns a foe to stone.", Level: 3}
	s := renderFbFeats([]fbFeature{feat})
	if !strings.Contains(s, "Foes Forever Frozen") {
		t.Fatalf("missing feature name in:\n%s", s)
	}
	if !strings.Contains(s, `class="fb__feat-body">The gaze turns a foe to stone.</div>`) {
		t.Errorf("missing the fallback-rendered body in:\n%s", s)
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

// --- SC-308 review I-1: the multi-roll `post` render path had no test at all —
// unlike the other two parsers, buildFeatureblockPage reads ALREADY-GENERATED
// frontmatter, so Overpower/Roll the Wheel's correct render depends on `post`
// surviving a ToMap -> yaml.Marshal -> yaml.Unmarshal -> renderFbFeat round
// trip. buildFbPageFromBlock puts that round trip itself under test by
// building the frontmatter the same way the real pipeline does: parse the
// source-shaped blockquote with content.ParseRichFeatures, convert with
// content.RichFeatureMaps (== RichFeature.ToMap per feature), then yaml.Marshal
// it into a page exactly like a generated md-linked page's frontmatter. ---

// buildFbPageFromBlock parses a single feature blockquote and marshals it into
// a `type: featureblock`/`dynamic-terrain` page the way `steel-etl gen` would
// have written it, so buildFeatureblockPage's caller sees the same
// already-generated-frontmatter shape production does.
func buildFbPageFromBlock(t *testing.T, name, ftype, block string) []byte {
	t.Helper()
	feats := content.ParseRichFeatures(block)
	if len(feats) != 1 {
		t.Fatalf("buildFbPageFromBlock: got %d features, want 1", len(feats))
	}
	fm := map[string]any{"name": name, "type": ftype, "features": content.RichFeatureMaps(feats)}
	y, err := yaml.Marshal(fm)
	if err != nil {
		t.Fatalf("buildFbPageFromBlock: yaml.Marshal: %v", err)
	}
	return []byte("---\n" + string(y) + "---\n\nbody\n")
}

const rollTheWheelBlock = "" +
	"> 🌀 **Roll the Wheel**\n" +
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

// TestBuildFeatureblockPage_MultiRoll_RollTheWheel locks the SC-308 fix on the
// path that round-trips through generated YAML frontmatter (Roll the Wheel is
// `type: dynamic-terrain`): two `.sc-ability__pr` panels must render, in
// document order, with the second bare (no head at all — no preceding
// "**<Characteristic> test**" phrase to derive a label from) and the prose
// paragraphs interleaved between/after them exactly as in the source.
func TestBuildFeatureblockPage_MultiRoll_RollTheWheel(t *testing.T) {
	page := buildFbPageFromBlock(t, "Exploding Mill Wheel", "dynamic-terrain", rollTheWheelBlock)
	out, ok := buildFeatureblockPage(page)
	if !ok {
		t.Fatal("dynamic-terrain page should be handled")
	}
	full := string(out)
	// The page is frontmatter (which itself echoes the prose/section text
	// inside the `post`/`sections` YAML) followed by the rendered HTML card —
	// index only the card so a frontmatter echo of the same plain text can't
	// masquerade as the card's own, much-later occurrence.
	cardStart := strings.Index(full, `<div class="fb-wrap"`)
	if cardStart < 0 {
		t.Fatalf("no rendered card in:\n%s", full)
	}
	s := full[cardStart:]

	if n := strings.Count(s, `<div class="sc-ability__pr">`); n != 2 {
		t.Fatalf("got %d .sc-ability__pr panels, want 2:\n%s", n, s)
	}

	// SC-308 round 3b: the Effect paragraph and the first roll MERGE into one
	// entry (attachment rule) — the panel nests INSIDE the Effect section, not
	// hoisted above it. The trailing prose paragraph and the second roll merge
	// too.
	effectIdx := strings.Index(s, `<span class="tag">Effect</span>`)
	firstPR := strings.Index(s, `<div class="sc-ability__pr">`)
	prose1Idx := strings.Index(s, "its movement stops and it explodes")
	secondPR := strings.LastIndex(s, `<div class="sc-ability__pr">`)
	prose2Idx := strings.Index(s, "A burning creature takes 1d6 fire damage")
	for name, idx := range map[string]int{
		"Effect section": effectIdx, "first panel": firstPR, "second prose (attaches the second roll)": prose1Idx,
		"second panel": secondPR, "trailing prose": prose2Idx,
	} {
		if idx < 0 {
			t.Fatalf("missing %s in:\n%s", name, s)
		}
	}
	if !(effectIdx < firstPR && firstPR < prose1Idx && prose1Idx < secondPR && secondPR < prose2Idx) {
		t.Errorf("wrong DOM order (want: Effect < first panel (nested inside it) < prose1 < second panel (nested inside it) < trailing prose):\n%s", s)
	}

	// The second panel is fully bare: no .sc-ability__pr-head at all between its
	// opening div and the first tier row.
	secondPanel := s[secondPR:]
	if end := strings.Index(secondPanel, `<div class="sc-ability__pr-rows">`); end >= 0 {
		secondPanel = secondPanel[:end]
	}
	if strings.Contains(secondPanel, "sc-ability__pr-head") {
		t.Errorf("second (bare) panel should have no pr-head:\n%s", secondPanel)
	}
}

const overpowerBlock = "" +
	"> 🌀 **Overpower (7 Malice)**\n" +
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

// TestBuildFeatureblockPage_MultiRoll_Overpower covers a malice featureblock
// with NO spec table and no first-roll formula: both tier lists are
// header-less, each labeled "Reason Test" from the "**Reason test**" phrase it
// follows (the label rule applies to the first list too within a multi-roll
// feature — see docs/statblocks.md).
func TestBuildFeatureblockPage_MultiRoll_Overpower(t *testing.T) {
	page := buildFbPageFromBlock(t, "Lord Syuul's Malice", "featureblock", overpowerBlock)
	out, ok := buildFeatureblockPage(page)
	if !ok {
		t.Fatal("featureblock page should be handled")
	}
	s := string(out)

	if n := strings.Count(s, `<span class="pre">Reason Test</span>`); n != 2 {
		t.Fatalf("got %d 'Reason Test' heads, want 2:\n%s", n, s)
	}
	if strings.Contains(s, `class="chars"`) {
		t.Errorf("a derived-label head must have no chars span:\n%s", s)
	}
	firstHead := strings.Index(s, `<span class="pre">Reason Test</span>`)
	secondHead := strings.LastIndex(s, `<span class="pre">Reason Test</span>`)
	if firstHead == secondHead {
		t.Fatalf("expected two distinct Reason Test heads:\n%s", s)
	}
}
