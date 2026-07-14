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
	if !strings.Contains(got, `sc-head__left-eyebrow sc-head__slot--line">Feature</div>`) {
		t.Errorf("plain feature kind-noun should read \"Feature\"\n%s", got)
	}
	if !strings.Contains(got, `sc-head__left-deck sc-head__slot--line">Summoner</div>`) {
		t.Errorf("plain feature provenance deck should read \"Summoner\"\n%s", got)
	}
	if strings.Contains(got, "Trait") {
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

	// container: noun "Feature" eyebrow + "Censor" provenance deck.
	if !strings.Contains(got, `sc-head__left-eyebrow sc-head__slot--line">Feature</div>`) {
		t.Errorf("container should carry the Feature kind-noun:\n%s", got)
	}
	if !strings.Contains(got, `sc-head__left-deck sc-head__slot--line">Censor</div>`) {
		t.Errorf("container provenance deck should read \"Censor\":\n%s", got)
	}
	// Fate child: inherits the source and appends its own subclass → "Censor · Fate".
	if !strings.Contains(got, `sc-head__left-deck sc-head__slot--line">Censor · Fate</div>`) {
		t.Errorf("nested subclass child deck should read \"Censor · Fate\":\n%s", got)
	}
	// only the container carries a kind-noun eyebrow; nested nodes do not.
	if n := strings.Count(got, "sc-head__left-eyebrow"); n != 1 {
		t.Errorf("expected exactly 1 kind-noun eyebrow (container only), got %d:\n%s", n, got)
	}
	// two provenance decks: the container ("Censor") + the Fate child; the plain child has none.
	if n := strings.Count(got, "sc-head__left-deck"); n != 2 {
		t.Errorf("expected exactly 2 provenance decks (container + Fate child), got %d:\n%s", n, got)
	}
}

// A nested ABILITY child (its scc contains feature.ability.*) must surface its own
// class·subclass in the left-deck, exactly like the standalone ability page. The
// renderer must NOT rely on the parent feature's subclass: a "choose one ability
// according to your subclass" container has none of its own, yet each child ability
// belongs to a specific subclass.
func TestRenderTraitCard_NestedAbilityShowsSubclassDeck(t *testing.T) {
	// Container feature with no subclass of its own; nested ability is Black Ash.
	fm := "name: 1st-Level College Features\ntype: feature\nclass: shadow\nscc: mcdm.heroes.v1/feature.shadow.level-1/1st-level-college-features"
	body := "\nYour college grants you one ability.\n\n" +
		"## Black Ash Teleport {data-scc=\"mcdm.heroes.v1/feature.ability.shadow.level-1/black-ash-teleport\" data-subclass=\"black-ash\"}\n\n" +
		"*In a swirl of black ash.*\n\n" +
		"**Effect:** You teleport up to 5 squares.\n"
	got := renderTraitCard(fm, body)

	if !strings.Contains(got, `<article class="sc-ability`) {
		t.Fatalf("expected a nested .sc-ability card:\n%s", got)
	}
	// the nested ability card must carry its own provenance deck "Shadow · Black Ash".
	if !strings.Contains(got, `sc-head__left-deck sc-head__slot--line">Shadow · Black Ash</div>`) {
		t.Errorf("nested ability deck should read \"Shadow · Black Ash\":\n%s", got)
	}
	// the container itself has no subclass → bare "Shadow" deck (no " · ").
	if !strings.Contains(got, `sc-head__left-deck sc-head__slot--line">Shadow</div>`) {
		t.Errorf("container deck should read bare \"Shadow\":\n%s", got)
	}
}

// A nested ABILITY child must surface the same right-rail slots as its standalone
// page: the resource cost (RenderSubtree stamps it as data-cost, e.g. "5 Focus")
// and the level chip (derived from the level-N segment of its scc). Regression:
// synthAbilityFM dropped both, so every nested ability card — and the class pages
// that embed those container features — lost the cost and the level.
func TestRenderTraitCard_NestedAbilityShowsCostAndLevel(t *testing.T) {
	fm := "name: 2nd-Level Doctrine Ability\ntype: feature\nclass: tactician\nscc: mcdm.heroes.v1/feature.tactician.level-2/2nd-level-doctrine-ability"
	body := "\nYour tactical doctrine grants one ability.\n\n" +
		"## Try Me Instead {data-scc=\"mcdm.heroes.v1/feature.ability.tactician.level-2/try-me-instead\" data-cost=\"5 Focus\" data-subclass=\"insurgent\"}\n\n" +
		"*Try picking on someone my size.*\n\n" +
		"**Effect:** You shift up to your speed.\n"
	got := renderTraitCard(fm, body)

	if !strings.Contains(got, `<article class="sc-ability`) {
		t.Fatalf("expected a nested .sc-ability card:\n%s", got)
	}
	if !strings.Contains(got, `sc-head__right-primary sc-head__slot--mini">5 Focus</div>`) {
		t.Errorf("nested ability should surface its \"5 Focus\" cost:\n%s", got)
	}
	if !strings.Contains(got, `sc-head__right-eyebrow sc-head__slot--chip">Level 2</div>`) {
		t.Errorf("nested ability should surface its \"Level 2\" chip:\n%s", got)
	}
}

// A real ancestry trait (type: trait) keeps the "<Ancestry> Trait" eyebrow.
func TestTraitEyebrow_AncestryTraitSaysTrait(t *testing.T) {
	fm := "ancestry: dragon-knight\nname: Prismatic Scales\ntype: trait"
	got := renderTraitCard(fm, "\nSelect one damage immunity.\n")
	if !strings.Contains(got, `sc-head__left-eyebrow sc-head__slot--line">Trait</div>`) {
		t.Errorf("ancestry trait kind-noun should read \"Trait\"\n%s", got)
	}
	if !strings.Contains(got, `sc-head__left-deck sc-head__slot--line">Dragon Knight</div>`) {
		t.Errorf("ancestry trait provenance deck should read \"Dragon Knight\"\n%s", got)
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

// A list item with indented continuation paragraphs (CommonMark multi-paragraph
// list item, 4-space indent) keeps those paragraphs inside the same <li> as <p>
// blocks — they must not leak out as top-level prose or collapse into the lead
// sentence. (Regression: SC-81, Companion Rules "Companion Actions".)
func TestRenderTraitCard_ListContinuationParagraphs(t *testing.T) {
	fm := "class: beastheart\nname: Companion Rules\ntype: feature"
	body := `
- **Companion Actions.** Your companion is your ally.

    You and your companion each take your own move action.

    You and your companion share one turn during montage tests.

- **Ranged Free Strikes.** Your companion doesn't have a ranged free strike.
`
	got := renderTraitCard(fm, body)
	want := "<li><p><b>Companion Actions.</b> Your companion is your ally.</p>" +
		"<p>You and your companion each take your own move action.</p>" +
		"<p>You and your companion share one turn during montage tests.</p></li>"
	if !strings.Contains(got, want) {
		t.Errorf("continuation paragraphs should render as <p> blocks inside the same <li>\nwant substring:\n%s\ngot:\n%s", want, got)
	}
	if !strings.Contains(got, "<li><b>Ranged Free Strikes.</b>") {
		t.Errorf("the following single-paragraph item should stay a bare <li>\n%s", got)
	}
	if strings.Count(got, "<ul>") != 1 {
		t.Errorf("the whole block should stay one <ul>, got %d\n%s", strings.Count(got, "<ul>"), got)
	}
}

// A blank line between bullet items makes the list loose in CommonMark — it does
// NOT start a second list. The card renderer must keep such items in one <ul>.
// (Regression: SC-81, the Companion Rules list split at a PDF column break.)
func TestRenderTraitCard_BlankLineSeparatedListStaysOneUL(t *testing.T) {
	fm := "class: beastheart\nname: Companion Rules\ntype: feature"
	body := `
- **Shared Space.** You can move through each other's spaces.

- **Surges.** Surges go into a shared pool.
`
	got := renderTraitCard(fm, body)
	if strings.Count(got, "<ul>") != 1 {
		t.Errorf("blank-line-separated items must stay one <ul>, got %d\n%s", strings.Count(got, "<ul>"), got)
	}
	if strings.Count(got, "<li>") != 2 {
		t.Errorf("want 2 <li> items, got %d\n%s", strings.Count(got, "<li>"), got)
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
		`sc-head__left-eyebrow sc-head__slot--line">Trait</div>`,
		`sc-head__left-deck sc-head__slot--line">Dragon Knight</div>`,
		`sc-head__left-primary sc-head__slot--line">Prismatic Scales</h3>`,
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
		`sc-head__right-eyebrow sc-head__slot--chip">Level 1</div>`,
		`<p class="sc-trait__leadin"><span class="sc-trait__dia"></span>You have the following signature ability.</p>`,
		`<div class="sc-trait__nest">`,
		`<article class="sc-ability sc-fil" data-action="main">`,      // nested ability plate
		`sc-head__right-primary sc-head__slot--mini">Signature</div>`, // signature hint propagated
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
	if !strings.Contains(got, `<h3 class="sc-head__slot sc-head__left-primary sc-head__slot--line">Dragon Knight Traits</h3>`) {
		t.Errorf("missing top trait name\n%s", got)
	}
	// the organizational H2 (no scc) becomes a nested sub-trait niche...
	if !strings.Contains(got, `<h3 class="sc-head__slot sc-head__left-primary sc-head__slot--line">Purchased Dragon Knight Traits</h3>`) {
		t.Errorf("missing organizational sub-trait niche\n%s", got)
	}
	// ...with no eyebrow (nested sub-traits omit the kind-noun line). Only the top
	// trait carries the "Trait" noun; nested sub-trait niches carry none. (A nested
	// ability plate has its own "Ability" eyebrow — that's a different noun.)
	if n := strings.Count(got, `sc-head__left-eyebrow sc-head__slot--line">Trait</div>`); n != 1 {
		t.Errorf("only the top trait should carry a Trait eyebrow; got %d\n%s", n, got)
	}
	// the H3 options are sub-trait niches
	if !strings.Contains(got, `<h3 class="sc-head__slot sc-head__left-primary sc-head__slot--line">Draconian Guard</h3>`) {
		t.Errorf("missing Draconian Guard sub-trait\n%s", got)
	}
	if !strings.Contains(got, `<h3 class="sc-head__slot sc-head__left-primary sc-head__slot--line">Draconian Pride</h3>`) {
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

// The level chip (right-eyebrow) falls back to a `level-N` segment in the scc when
// frontmatter has no level (beastheart traits), and is omitted when none exists.
func TestTraitCard_LevelFromSCCFallback(t *testing.T) {
	got := renderTraitCard("ancestry: beastheart\nname: There For Each Other\ntype: trait\nscc: mcdm.beastheart.v1/feature.trait.beastheart.level-5/there-for-each-other", "Body.")
	if !strings.Contains(got, `sc-head__right-eyebrow sc-head__slot--chip">Level 5</div>`) {
		t.Errorf("expected Level 5 chip from scc fallback:\n%s", got)
	}
	noLevel := renderTraitCard("name: X\ntype: trait\nclass: censor\nscc: mcdm.heroes.v1/feature.trait.censor/no-level", "Body.")
	if strings.Contains(noLevel, "sc-head__right-eyebrow") {
		t.Errorf("expected no level chip when none anywhere:\n%s", noLevel)
	}
}

// A purchased ancestry trait's point cost renders as the right-primary mini-title.
func TestTraitCard_CostInRightPrimary(t *testing.T) {
	got := renderTraitCard("name: Barbed Tail\ntype: trait\nancestry: devil\ncost: 1 Point", "Body.")
	if !strings.Contains(got, `sc-head__right-primary sc-head__slot--mini">1 Point</div>`) {
		t.Errorf("expected cost \"1 Point\" in right-primary mini:\n%s", got)
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

// A Summoner circle feature gets the "circle" qualifier appended to the source
// (the kind-noun is now a separate slot): traitSource → "Summoner Circle".
func TestTraitSource_CircleFeature(t *testing.T) {
	fm := "class: summoner\ntype: feature\nfeature_source: circle\n"
	if got := traitSource(fm); got != "Summoner Circle" {
		t.Errorf("traitSource = %q, want %q", got, "Summoner Circle")
	}
}

// A base Summoner feature (feature_source: summoner or absent) keeps "Summoner".
func TestTraitSource_SummonerBaseUnchanged(t *testing.T) {
	for _, fm := range []string{
		"class: summoner\ntype: feature\nfeature_source: summoner\n",
		"class: summoner\ntype: feature\n", // absent
	} {
		if got := traitSource(fm); got != "Summoner" {
			t.Errorf("traitSource(%q) = %q, want %q", fm, got, "Summoner")
		}
	}
}

func TestTraitCard_SixSlotHead(t *testing.T) {
	fm := "name: Black Ash Teleport\ntype: feature\nclass: shadow\nsubclass: college-of-black-ash\nlevel: 1"
	got := renderTraitCard(fm, "Some body.")
	for _, want := range []string{
		`sc-head__left-eyebrow sc-head__slot--line">Feature</div>`,
		`sc-head__left-primary sc-head__slot--line">Black Ash Teleport</h3>`,
		`sc-head__left-deck sc-head__slot--line">Shadow · College Of Black Ash</div>`,
		`sc-head__right-eyebrow sc-head__slot--chip">Level 1</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// A trait tree with grandchildren (sections that themselves contain sub-features)
// is a SECTION WRAPPER, not a card: on ancestry pages the "X Traits" root wraps
// 1,500-2,300px of nested sections, and the card panel's 3px act spine renders as
// a stray full-height purple line in the page gutter (SC-84). Such roots carry
// sc-trait--section so the CSS can drop the panel chrome (the crested header stays).
func TestRenderTraitCard_GrandchildrenMakeSectionWrapper(t *testing.T) {
	fm := "name: Polder Traits\ntype: trait\nancestry: polder\nscc: mcdm.heroes.v1/trait.ancestry.polder/polder-traits"
	body := "\nIntro prose.\n\n" +
		"### Purchased Polder Traits {data-scc=\"mcdm.heroes.v1/trait.ancestry.polder/purchased\"}\n\n" +
		"You have 4 ancestry points.\n\n" +
		"#### Nimblestep {data-scc=\"mcdm.heroes.v1/trait.ancestry.polder/nimblestep\" data-cost=\"2 Points\"}\n\n" +
		"A light step serves you well.\n\n" +
		"#### Fearless {data-scc=\"mcdm.heroes.v1/trait.ancestry.polder/fearless\" data-cost=\"2 Points\"}\n\n" +
		"Courage is all you know.\n"
	got := renderTraitCard(fm, body)
	if !strings.Contains(got, "sc-trait--section") {
		t.Errorf("root with a collection child should carry sc-trait--section:\n%s", got)
	}
}

// A flat leaf trait (children but no grandchildren) keeps the card panel — no
// sc-trait--section. This covers both plain leaves and small "choose one" cards.
func TestRenderTraitCard_LeafKeepsPanel(t *testing.T) {
	fm := "name: Nimblestep\ntype: trait\nancestry: polder\nscc: mcdm.heroes.v1/trait.ancestry.polder/nimblestep"
	body := "\nA light step serves you well when speed is of the essence.\n"
	if got := renderTraitCard(fm, body); strings.Contains(got, "sc-trait--section") {
		t.Errorf("flat leaf must not be a section wrapper:\n%s", got)
	}
	// one level of children (an ability grant) is still a card
	body = "\nChoose the following.\n\n" +
		"### Shadowmeld {data-scc=\"mcdm.heroes.v1/feature.ability.polder/shadowmeld\"}\n\n" +
		"> ability body\n"
	if got := renderTraitCard(fm, body); strings.Contains(got, "sc-trait--section") {
		t.Errorf("single-level tree must not be a section wrapper:\n%s", got)
	}
}

// A "choose one" card whose options each carry a SINGLE grant (e.g. a class
// page's "1st-Level Domain Feature": option → one granted ability) has
// grandchildren but no collection child — it stays a card. Only a child that
// is itself a collection (≥2 children) makes the root a section wrapper.
func TestRenderTraitCard_SingleGrantOptionsKeepPanel(t *testing.T) {
	fm := "name: 1st-Level Domain Feature\ntype: feature\nclass: censor\nscc: mcdm.heroes.v1/feature.censor.level-1/1st-level-domain-feature"
	body := "\nChoose one.\n\n" +
		"### Oracular Warning {data-scc=\"mcdm.heroes.v1/feature.censor.level-1/oracular-warning\" data-subclass=\"fate\"}\n\n" +
		"Premonitions.\n\n" +
		"#### Warning Ability {data-scc=\"mcdm.heroes.v1/feature.ability.censor.level-1/warning\"}\n\n" +
		"> ability body\n\n" +
		"### Stone Ward {data-scc=\"mcdm.heroes.v1/feature.censor.level-1/stone-ward\" data-subclass=\"creation\"}\n\n" +
		"Wards.\n\n" +
		"#### Ward Ability {data-scc=\"mcdm.heroes.v1/feature.ability.censor.level-1/ward\"}\n\n" +
		"> ability body\n"
	if got := renderTraitCard(fm, body); strings.Contains(got, "sc-trait--section") {
		t.Errorf("single-grant options must not make a section wrapper:\n%s", got)
	}
}
