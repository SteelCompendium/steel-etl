package site

import (
	"strings"
	"testing"
)

// A plain feature (type: feature) shares the recessed .sc-trait visual with real
// ancestry traits, but its eyebrow noun must read "<Source> Feature" — not "Trait".
// (Regression: Summoner Strike, a class feature, was labelled "Summoner Trait".)
func TestTraitEyebrow_PlainFeatureSaysFeature(t *testing.T) {
	fm := "class: summoner\nname: Summoner Strike\ntype: feature\nlevel: \"1\""
	got := renderTraitCard(fm, "\nYou have the following ability.\n")
	if !strings.Contains(got, `<div class="sc-trait__eyebrow"><span class="sc-trait__dia"></span>Summoner Feature</div>`) {
		t.Errorf("plain feature eyebrow should read \"Summoner Feature\"\n%s", got)
	}
	if strings.Contains(got, "Summoner Trait") {
		t.Errorf("plain feature must not be labelled a Trait\n%s", got)
	}
}

// A nested option card (a child under a generic container like "4th-Level Domain
// Feature") that carries a subclass must show the full leaf-style eyebrow, inheriting
// the container's class prefix. A sibling without a subclass stays eyebrow-less.
func TestRenderTraitCard_NestedChildShowsSubclassEyebrow(t *testing.T) {
	fm := "name: 4th-Level Domain Feature\ntype: feature\nclass: censor\nscc: mcdm.heroes.v1/feature.censor.level-4/4th-level-domain-feature"
	body := "\nChoose one of the following.\n\n" +
		"### Oracular Warning {data-scc=\"mcdm.heroes.v1/feature.censor.level-4/oracular-warning\" data-subclass=\"fate\"}\n\n" +
		"Premonitions help you stay alive.\n\n" +
		"### Plain Child {data-scc=\"mcdm.heroes.v1/feature.censor.level-4/plain-child\"}\n\n" +
		"No subclass here.\n"
	got := renderTraitCard(fm, body)

	if !strings.Contains(got, `<div class="sc-trait__eyebrow"><span class="sc-trait__dia"></span>Censor Feature · Fate</div>`) {
		t.Errorf("nested subclass child should carry the full eyebrow:\n%s", got)
	}
	// exactly 2 eyebrows: the container ("Censor Feature") + the Fate child.
	if n := strings.Count(got, "sc-trait__eyebrow"); n != 2 {
		t.Errorf("expected exactly 2 eyebrows (container + Fate child), got %d:\n%s", n, got)
	}
}

// A real ancestry trait (type: trait) keeps the "<Ancestry> Trait" eyebrow.
func TestTraitEyebrow_AncestryTraitSaysTrait(t *testing.T) {
	fm := "ancestry: dragon-knight\nname: Prismatic Scales\ntype: trait"
	got := renderTraitCard(fm, "\nSelect one damage immunity.\n")
	if !strings.Contains(got, "Dragon Knight Trait") {
		t.Errorf("ancestry trait eyebrow should read \"Dragon Knight Trait\"\n%s", got)
	}
}

// A plain feature (type: feature) routes through the recessed niche, same as a trait.
func TestBuildAbilityCardPage_PlainFeature(t *testing.T) {
	page := "---\ntype: feature\nname: A Beyonding of Vision\n---\n\nYour void sense reaches further.\n"
	out, ok := buildAbilityCardPage([]byte(page), nil)
	if !ok {
		t.Fatal("expected plain feature to be rendered as a card")
	}
	if !strings.Contains(string(out), "sc-trait") {
		t.Errorf("plain feature should render the recessed .sc-trait niche\n%s", string(out))
	}
}

// A feature page whose subtree contains a standalone item (statblock/featureblock)
// must be left UNCARDED so the embedItemCards post-pass can splice the proper item
// cards. Otherwise renderTraitCard renders the item as a generic .sc-trait niche —
// mangling its blockquote stat/feature content — and consumes the {data-scc}
// markers embed needs. (Bug: summoner fixture featureblocks like "The Boil"
// rendered as a Feature inside the 2nd-Level Features card.)
func TestBuildAbilityCardPage_DefersStandaloneDescendantToEmbed(t *testing.T) {
	page := "---\ntype: feature\nname: 2nd-Level Features\n" +
		"scc: mcdm.summoner.v1/feature.summoner.level-2/2nd-level-features\n---\n\n" +
		"As a 2nd-level summoner, you gain the following features.\n\n" +
		"##### The Boil {data-scc=\"mcdm.summoner.v1/monster.fixture.demon.featureblock/the-boil\"}\n\n" +
		"> ⭐️ **Hunger Thrush**\n>\n> Each enemy nearby is taunted.\n"
	standalone := map[string]bool{
		"mcdm.summoner.v1/monster.fixture.demon.featureblock/the-boil": true,
	}
	out, ok := buildAbilityCardPage([]byte(page), standalone)
	if ok {
		t.Fatalf("feature with a standalone descendant must be left uncarded for embed\n%s", string(out))
	}
	if !strings.Contains(string(out), `{data-scc="mcdm.summoner.v1/monster.fixture.demon.featureblock/the-boil"}`) {
		t.Errorf("the {data-scc} marker must survive for the embed post-pass\n%s", string(out))
	}
	if strings.Contains(string(out), "sc-trait") {
		t.Errorf("a standalone-bearing feature must NOT be pre-rendered into a trait card\n%s", string(out))
	}
}

// A feature with NO standalone descendants is still carded, even when a standalone
// set is supplied (only matching descendants defer to embed).
func TestBuildAbilityCardPage_CardsPlainFeatureDespiteStandaloneSet(t *testing.T) {
	page := "---\ntype: feature\nname: Perk\n---\n\nYou gain a perk of your choice.\n"
	standalone := map[string]bool{"mcdm.x.v1/monster.cat.statblock/unrelated": true}
	out, ok := buildAbilityCardPage([]byte(page), standalone)
	if !ok {
		t.Fatal("a feature without standalone descendants should still be carded")
	}
	if !strings.Contains(string(out), "sc-trait") {
		t.Errorf("plain feature should render the .sc-trait niche\n%s", string(out))
	}
}

// A plain feature whose body embeds a test's ≤11/12-16/17+ tier outcomes (e.g.
// the Summoner's Fairy Whispers) must render the glyph-badged tier panel inside
// the niche, NOT a plain <ul> — and without inventing a "Power Roll +" header.
func TestRenderTraitCard_TierTestPanel(t *testing.T) {
	fm := "class: summoner\nname: Fairy Whispers\ntype: feature\nlevel: \"1\""
	body := `
When the minion returns, make a Reason test:

- **≤11:** You learn an undoubtedly false common rumor.
- **12-16:** You learn a common rumor that is most likely true.
- **17+:** You learn an obscure rumor that could either be true or false.

You gain a bane on the test for each subsequent rumor.
`
	got := renderTraitCard(fm, body)
	wants := []string{
		`<div class="sc-ability__pr">`,
		`data-tier="low"><span class="badge">!</span><span class="res">You learn an undoubtedly false common rumor.</span>`,
		`data-tier="high"><span class="badge">#</span>`,
		`<p>You gain a bane`, // trailing prose still rendered
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("feature tier panel missing %q\n--- got ---\n%s", w, got)
		}
	}
	if strings.Contains(got, "<li>") {
		t.Errorf("tier outcomes must not stay a plain <ul> list\n%s", got)
	}
	if strings.Contains(got, "Power Roll +") {
		t.Errorf("a test must not synthesize a Power Roll + header\n%s", got)
	}
}

// A non-tier bullet list inside a feature must STILL render as an ordinary <ul>
// (no over-reach: only the ≤11/12-16/17+ signature becomes a tier panel).
func TestRenderTraitCard_PlainListUnchanged(t *testing.T) {
	fm := "class: summoner\nname: Formation\ntype: feature"
	body := `
Choose one of the following formations:

- **Horde:** More minions.
- **Platoon:** Extra damage.
- **Elite:** Tougher minions.
`
	got := renderTraitCard(fm, body)
	if !strings.Contains(got, "<ul>") || !strings.Contains(got, "<li>") {
		t.Errorf("a non-tier list should remain a plain <ul>\n%s", got)
	}
	if strings.Contains(got, "sc-ability__pr") {
		t.Errorf("a non-tier list must not become a tier panel\n%s", got)
	}
}

// A pure-prose trait → a flat niche with the drop-cap modifier and one paragraph.
func TestRenderTraitCard_ProseOnly(t *testing.T) {
	fm := "ancestry: dragon-knight\nname: Prismatic Scales\ntype: trait\nscc: mcdm.heroes.v1/feature.trait.dragon-knight/prismatic-scales"
	body := "\nSelect one damage immunity granted by your Wyrmplate trait. You always have this immunity.\n"
	got := renderTraitCard(fm, body)

	wants := []string{
		`<section class="sc-trait sc-trait--crest sc-trait--lead" data-action="trait">`,
		`<span class="sc-crest sc-trait__crest"><span class="sc-trait__glyph">*</span></span>`,
		`<div class="sc-trait__eyebrow"><span class="sc-trait__dia"></span>Dragon Knight Trait</div>`,
		`<h3 class="sc-trait__name">Prismatic Scales</h3>`,
		`<p>Select one damage immunity`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("trait card missing %q\n--- got ---\n%s", w, got)
		}
	}
	if strings.Contains(got, "sc-trait__nest") {
		t.Errorf("prose-only trait should have no nest rail\n%s", got)
	}
	if strings.Contains(got, "\n\n") {
		t.Errorf("card must be contiguous (no blank lines) for md_in_html")
	}
}

// A trait that grants a signature ability → lead-in + a nested .sc-ability inside
// a single nest rail, with the Signature cost badge propagated from the lead-in.
func TestRenderTraitCard_GrantsAbility(t *testing.T) {
	fm := "ancestry: dragon-knight\nname: Dragon Breath\ntype: trait\nlevel: \"1\""
	body := `
You have the following signature ability.

## Dragon Breath {data-scc="mcdm.heroes.v1/feature.ability.dragon-knight/dragon-breath"}

*A furious exhalation of energy washes over your foes.*

| **Area, Magic**        |               **Main action** |
|------------------------|------------------------------:|
| **📏 3 cube within 1** | **🎯 Each enemy in the area** |

**Power Roll + Might or Presence:**

- **≤11:** 2 damage
- **12-16:** 4 damage
- **17+:** 6 damage

**Effect:** You choose the ability's damage type.
`
	got := renderTraitCard(fm, body)

	wants := []string{
		`<div class="sc-trait__tag">Level <span class="num">1</span></div>`,
		`<p class="sc-trait__leadin"><span class="sc-trait__dia"></span>You have the following signature ability.</p>`,
		`<div class="sc-trait__nest">`,
		`<article class="sc-ability sc-fil" data-action="main">`, // nested ability plate
		`<div class="sc-ability__cost">Signature</div>`,          // signature hint propagated
		`<span class="chars">Might or Presence</span>`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("trait card missing %q\n--- got ---\n%s", w, got)
		}
	}
	// the lead-in is prose-but-leadin, so no drop cap
	if strings.Contains(got, "sc-trait--lead") {
		t.Errorf("lead-in paragraph should not trigger the drop-cap modifier\n%s", got)
	}
}

// A choose-one feature whose options are nested sub-traits (recursive niches),
// one of which itself grants an ability (deep nesting).
func TestRenderTraitCard_NestedSubTraits(t *testing.T) {
	fm := "ancestry: dragon-knight\nname: Dragon Knight Traits\ntype: trait"
	body := `
Dragon knight heroes have access to the following traits.

## Purchased Dragon Knight Traits

You have 3 ancestry points to spend on the following traits.

### Draconian Guard {data-scc="mcdm.heroes.v1/feature.trait.dragon-knight/draconian-guard"}

Whenever you or an adjacent creature takes damage, you can guard against the blow.

### Draconian Pride {data-scc="mcdm.heroes.v1/feature.trait.dragon-knight/draconian-pride"}

You have the following signature ability.

#### Draconian Pride {data-scc="mcdm.heroes.v1/feature.ability.dragon-knight/draconian-pride"}

*You let loose a mighty roar.*

**Power Roll + Might or Presence:**

- **≤11:** 2 damage
- **12-16:** 5 damage
- **17+:** 7 damage
`
	got := renderTraitCard(fm, body)

	// top niche
	if !strings.Contains(got, `<h3 class="sc-trait__name">Dragon Knight Traits</h3>`) {
		t.Errorf("missing top trait name\n%s", got)
	}
	// the organizational H2 (no scc) becomes a nested sub-trait niche...
	if !strings.Contains(got, `<h3 class="sc-trait__name">Purchased Dragon Knight Traits</h3>`) {
		t.Errorf("missing organizational sub-trait niche\n%s", got)
	}
	// ...with no eyebrow (nested traits omit the class line)
	if strings.Count(got, "sc-trait__eyebrow") != 1 {
		t.Errorf("only the top trait should carry an eyebrow; got %d\n%s", strings.Count(got, "sc-trait__eyebrow"), got)
	}
	// the H3 options are sub-trait niches
	if !strings.Contains(got, `<h3 class="sc-trait__name">Draconian Guard</h3>`) {
		t.Errorf("missing Draconian Guard sub-trait\n%s", got)
	}
	if !strings.Contains(got, `<h3 class="sc-trait__name">Draconian Pride</h3>`) {
		t.Errorf("missing Draconian Pride sub-trait\n%s", got)
	}
	// the H4 under Draconian Pride is its nested ability plate
	if !strings.Contains(got, `<article class="sc-ability sc-fil"`) {
		t.Errorf("missing deeply-nested ability plate\n%s", got)
	}
	// multiple nest rails (top + the Purchased group + the ability-granting option)
	if strings.Count(got, "sc-trait__nest") < 2 {
		t.Errorf("expected multiple nest rails for recursive structure\n%s", got)
	}
	if strings.Contains(got, "\n\n") {
		t.Errorf("card must be contiguous (no blank lines) for md_in_html")
	}
}

// A benefit/drawback labeled paragraph → titled tone segment.
func TestRenderTraitCard_Segments(t *testing.T) {
	fm := "class: censor\nname: Sworn Enemy\ntype: trait"
	body := "\n**Benefit:** You deal extra damage to your sworn enemy.\n\n**Drawback:** You have a bane on tests against other creatures.\n"
	got := renderTraitCard(fm, body)

	wants := []string{
		`<div class="sc-trait__seg" data-tone="benefit">`,
		`<span class="tag">Benefit</span>`,
		`<div class="sc-trait__seg" data-tone="drawback">`,
		`<span class="tag">Drawback</span>`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("segment trait missing %q\n--- got ---\n%s", w, got)
		}
	}
}

// A markdown table in a trait body renders as a real HTML <table>, not a raw
// pipe paragraph, with links inside cells resolved.
func TestRenderTraitCard_Table(t *testing.T) {
	fm := "class: censor\nname: Domain Effects\ntype: trait"
	body := `
Choose your domain feature.

| Domain | Feature |
|------------|-----------------------------|
| Creation | Improved [Hands of the Maker](../hands-of-the-maker.md) |
| Death | Seance |
`
	got := renderTraitCard(fm, body)
	wants := []string{
		`<table><thead><tr><th>Domain</th><th>Feature</th></tr></thead>`,
		`<tbody><tr><td>Creation</td><td>Improved <a href=`,
		`<tr><td>Death</td><td>Seance</td></tr>`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("table trait missing %q\n--- got ---\n%s", w, got)
		}
	}
	if strings.Contains(got, "<p>| Domain") {
		t.Errorf("table must not fall through to a raw-pipe paragraph\n%s", got)
	}
}

// The level pill falls back to a `level-N` segment in the scc when frontmatter has
// no level (beastheart traits).
func TestTraitTag_SCCFallback(t *testing.T) {
	if got := traitTag("", "", "mcdm.beastheart.v1/feature.trait.beastheart.level-5/there-for-each-other"); !strings.Contains(got, `<span class="num">5</span>`) {
		t.Errorf("expected level 5 from scc fallback, got %q", got)
	}
	if got := traitTag("", "", "mcdm.heroes.v1/feature.trait.censor/no-level"); got != "" {
		t.Errorf("expected empty tag when no level anywhere, got %q", got)
	}
}

// A purchased ancestry trait's point cost takes precedence and renders with the
// number emphasized; the unit label follows.
func TestTraitTag_Cost(t *testing.T) {
	got := traitTag("1 Point", "", "mcdm.heroes.v1/feature.trait.devil/barbed-tail")
	if !strings.Contains(got, `<span class="num">1</span>`) || !strings.Contains(got, "Point") {
		t.Errorf("expected emphasized cost \"1 Point\", got %q", got)
	}
}

// parseTraitTree rebuilds the heading subtree by level.
func TestParseTraitTree_Levels(t *testing.T) {
	body := `intro prose

## A {data-scc="x/feature.trait.c/a"}

a body

### A1 {data-scc="x/feature.ability.c/a1"}

a1 body

## B {data-scc="x/feature.trait.c/b"}

b body`
	intro, roots := parseTraitTree(body)
	if intro != "intro prose" {
		t.Errorf("intro = %q", intro)
	}
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	if roots[0].name != "A" || len(roots[0].children) != 1 {
		t.Fatalf("A should own one child, got %+v", roots[0])
	}
	if !roots[0].children[0].isAbility {
		t.Errorf("A1 should be flagged as ability")
	}
	if roots[1].name != "B" || len(roots[1].children) != 0 {
		t.Errorf("B should be a leaf sibling of A")
	}
}

// A callout block embedded in a feature body must render as a recessed
// `.sc-callout` aside — never leak its `<!--` comment or `>` blockquote markers
// as escaped text in a lead-in paragraph (the bug on the class/chapter pages).
func TestRenderTraitCard_CalloutBecomesAside(t *testing.T) {
	fm := "class: summoner\nname: Leader Formation\ntype: feature"
	body := "\nYou aren't affected by excess damage.\n\n" +
		"<!-- @type: callout | @owner: loose -->\n" +
		"> **Minions and Treasures**\n" +
		">\n" +
		"> [Treasures](../x.md) are worded for you to use.\n" +
		">\n" +
		"> - First guideline.\n" +
		"> - Second guideline.\n"
	got := renderTraitCard(fm, body)

	if !strings.Contains(got, `<aside class="sc-callout"`) {
		t.Errorf("callout should render as an .sc-callout aside\n%s", got)
	}
	if !strings.Contains(got, "Minions and Treasures") {
		t.Errorf("callout title missing\n%s", got)
	}
	if strings.Contains(got, "&lt;!--") || strings.Contains(got, "@type: callout") {
		t.Errorf("callout comment must not leak as text\n%s", got)
	}
	if strings.Contains(got, "&gt;") {
		t.Errorf("blockquote markers must not leak as escaped text\n%s", got)
	}
	if !strings.Contains(got, "<li>First guideline.</li>") {
		t.Errorf("callout body list should render as <ul>/<li>\n%s", got)
	}
	// The feature's own prose still renders normally.
	if !strings.Contains(got, "You aren&#39;t affected by excess damage.") {
		t.Errorf("feature prose missing\n%s", got)
	}
}

// A Summoner circle feature gets the "circle" qualifier between the class name and
// the noun: "Summoner Circle Feature".
func TestTraitEyebrow_CircleFeature(t *testing.T) {
	fm := "class: summoner\ntype: feature\nfeature_source: circle\n"
	if got := traitEyebrow(fm); got != "Summoner Circle Feature" {
		t.Errorf("traitEyebrow = %q, want %q", got, "Summoner Circle Feature")
	}
}

// A base Summoner feature (feature_source: summoner or absent) keeps "Summoner Feature".
func TestTraitEyebrow_SummonerFeatureUnchanged(t *testing.T) {
	for _, fm := range []string{
		"class: summoner\ntype: feature\nfeature_source: summoner\n",
		"class: summoner\ntype: feature\n", // absent
	} {
		if got := traitEyebrow(fm); got != "Summoner Feature" {
			t.Errorf("traitEyebrow(%q) = %q, want %q", fm, got, "Summoner Feature")
		}
	}
}
