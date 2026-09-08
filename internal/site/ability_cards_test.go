package site

import (
	"strings"
	"testing"
)

func TestBuildAbilityCardPage_NonAbilityUnchanged(t *testing.T) {
	in := []byte("---\ntype: kit\nname: Mountain\n---\n\nbody\n")
	out, ok := buildAbilityCardPage(in, nil)
	if ok {
		t.Fatalf("expected ok=false for type: kit")
	}
	if string(out) != string(in) {
		t.Fatalf("non-ability data should pass through unchanged")
	}
}

// A subclass ability's leaf card must surface the subclass on the page itself
// (not just the preview/filter cards). In the 6-slot header that lives in the
// left-deck provenance line as "<Class> · <Subclass>".
func TestRenderAbilityCard_SubclassInDeck(t *testing.T) {
	fm := "type: ability\nname: Black Ash Teleport\nclass: shadow\nsubclass: black-ash\nlevel: \"1\"\naction_type: Maneuver"
	got := renderAbilityCard(fm, "\n*In a swirl of black ash, you step from one place to another.*\n", "")
	if !strings.Contains(got, `sc-head__left-deck sc-head__slot--line">Shadow · Black Ash</div>`) {
		t.Errorf("ability leaf should surface subclass in the left-deck:\n%s", got)
	}
}

func TestRenderAbilityCard_MainPowerRoll(t *testing.T) {
	fm := "action_type: Main action\nname: Dragon Breath\nsubtype: signature\ntype: ability"
	body := `
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
	got := renderAbilityCard(fm, body, "")
	wants := []string{
		`data-action="main"`,
		`<span class="sc-ability__glyph">l</span>`,
		`>Main Action</div>`,
		`sc-head__left-primary sc-head__slot--line">Dragon Breath</h3>`,
		`sc-head__right-primary sc-head__slot--mini">Signature</div>`,
		`<span class="sc-ability__chip">Area</span>`,
		`<div class="v">3 cube within 1</div>`, // emoji stripped from rail
		`<div class="v">Each enemy in the area</div>`,
		`<span class="chars">Might or Presence</span>`, // multi-characteristic roll survives
		`data-tier="low"><span class="badge">!</span><span class="res">2 damage</span>`,
		`data-tier="high"><span class="badge">#</span><span class="res">6 damage</span>`,
		`<span class="tag">Effect</span>`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("card missing %q\n--- got ---\n%s", w, got)
		}
	}
	if strings.Contains(got, "\n\n") {
		t.Errorf("card must be a contiguous block (no blank lines) for md_in_html")
	}
}

// The SCC linking sweep wraps the Power-Roll header and its characteristics in
// markdown links ("**[Power Roll](…) + [Might](…) or [Agility](…):**"). The card
// must still detect the power-roll panel (glyph-badged tiers) AND render the
// characteristic links, rather than dropping the line into a generic section.
func TestRenderAbilityCard_LinkedPowerRollHeader(t *testing.T) {
	fm := "action_type: Main action\nname: Protective Attack\nsubtype: signature\ntype: ability"
	body := `
*The strength of your assault makes it impossible for your foe to ignore you.*

**[Power Roll](../../../rule/dice/power-roll.md) + [Might](../../../rule/character/might.md) or [Agility](../../../rule/character/agility.md):**

- **≤11:** 5 + M or A damage
- **12-16:** 8 + M or A damage
- **17+:** 11 + M or A damage

**Effect:** The target is taunted.
`
	got := renderAbilityCard(fm, body, "")
	wants := []string{
		`sc-ability__pr-head`,                   // power-roll panel detected, not a plain section
		`<span class="pre">Power Roll +</span>`, // fixed eyebrow label
		`>Might</a> or <a`,                      // characteristic links rendered (not escaped/dropped)
		`data-tier="low"><span class="badge">!</span><span class="res">5 + M or A damage</span>`,
		`data-tier="high"><span class="badge">#</span>`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("linked power-roll card missing %q\n--- got ---\n%s", w, got)
		}
	}
	// The header characteristics must NOT leak as escaped markdown link syntax.
	if strings.Contains(got, "[Might]") {
		t.Errorf("characteristic markdown should be rendered as a link, not escaped: %q", got)
	}
}

// A "test" reuses the ≤11/12-16/17+ tier outcomes but has NO "**Power Roll +**"
// header — just a "make a … test:" lead-in followed by the tier bullets. The card
// must still render the glyph-badged tier panel, WITHOUT synthesizing a fake
// "Power Roll +" header (a test carries no characteristic to show there).
func TestRenderAbilityCard_HeaderlessTierTest(t *testing.T) {
	fm := "type: ability\nname: Scrying"
	body := `
Make a Reason test:

- **≤11:** A false rumor.
- **12-16:** A likely rumor.
- **17+:** An obscure rumor.

**Effect:** You learn something.
`
	got := renderAbilityCard(fm, body, "")
	wants := []string{
		`<div class="sc-ability__pr">`, // tier panel detected
		`data-tier="low"><span class="badge">!</span><span class="res">A false rumor.</span>`,
		`data-tier="mid"><span class="badge">@</span><span class="res">A likely rumor.</span>`,
		`data-tier="high"><span class="badge">#</span><span class="res">An obscure rumor.</span>`,
		`<span class="tag">Effect</span>`, // trailing section still parsed
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("header-less tier card missing %q\n--- got ---\n%s", w, got)
		}
	}
	// A bare test must NOT invent a "Power Roll +" header.
	if strings.Contains(got, "sc-ability__pr-head") || strings.Contains(got, "Power Roll +") {
		t.Errorf("header-less test must not synthesize a Power Roll + header\n%s", got)
	}
	// The tiers must not survive as a plain bullet list.
	if strings.Contains(got, "<li>") {
		t.Errorf("tier outcomes should render as the panel, not a <li> list\n%s", got)
	}
}

// SC-310 review r1, LOW-3: a header-less tier list attached to a paragraph
// whose OWN prose names a bold "**<Char> test**" phrase must derive that
// phrase as the panel's display head (content.DeriveTestLabel), rendering it
// as a bare "pre" label with no synthesized "Power Roll +" — the branch
// TestRenderAbilityCard_HeaderlessTierTest never reaches (its lead-in is
// unbolded prose, so it only pins the "no label found" fallback).
func TestRenderAbilityCard_HeaderlessTierDerivedLabel(t *testing.T) {
	fm := "type: ability\nname: Foresight"
	body := `
You glimpse the immediate future and make an **Agility test**:

- **≤11:** You are surprised.
- **12-16:** You are not surprised.
- **17+:** You are not surprised and gain an edge on your first strike.
`
	got := renderAbilityCard(fm, body, "")
	if !strings.Contains(got, `<span class="pre">Agility Test</span>`) {
		t.Errorf("expected the derived Agility Test label, got:\n%s", got)
	}
	if strings.Contains(got, "Power Roll +") {
		t.Errorf("a derived test label must not read as a synthesized Power Roll + header\n%s", got)
	}
	if !strings.Contains(got, `data-tier="low"><span class="badge">!</span><span class="res">You are surprised.</span>`) {
		t.Errorf("tier1 missing:\n%s", got)
	}
}

// Divine Dragon (SC-310): two "Power Roll + Intuition" tier panels, each
// sitting below its own bare-prose paragraph, must BOTH render — in source
// order, neither overwriting the other (the pre-fix bug: the tier loop
// overwrote, so only the second list's values ever reached the card).
func TestRenderAbilityCard_DivineDragonTwoRolls(t *testing.T) {
	fm := "action_type: Main action\nname: Divine Dragon\ncost: 11 Piety\ntype: ability"
	body := `
*From nothing but divine will, you create a powerful ally.*

| **Magic, Ranged**  | **Main action** |
|---------------------|----------------:|
| **📏 Ranged 10**   |  **🎯 Special** |

**Effect:** You conjure a size 4 dragon that appears in an unoccupied space.

On subsequent turns, you can use a main action to command the dragon to breathe magic fire. Make the following power roll targeting each enemy in the area.

**Power Roll + Intuition:**

- **≤11:** 5 fire damage
- **12-16:** 9 fire damage
- **17+:** 12 fire damage

Additionally, you can use a maneuver to move the dragon, or to make a melee weapon strike with their claw.

**Power Roll + Intuition:**

- **≤11:** 3 + I damage
- **12-16:** 5 + I damage
- **17+:** 8 + I damage
`
	got := renderAbilityCard(fm, body, "")
	if n := strings.Count(got, `class="sc-ability__pr"`); n != 2 {
		t.Fatalf("expected 2 power-roll panels, got %d\n--- got ---\n%s", n, got)
	}
	for _, w := range []string{
		`data-tier="low"><span class="badge">!</span><span class="res">5 fire damage</span>`,
		`data-tier="high"><span class="badge">#</span><span class="res">12 fire damage</span>`,
		`data-tier="low"><span class="badge">!</span><span class="res">3 + I damage</span>`,
		`data-tier="high"><span class="badge">#</span><span class="res">8 + I damage</span>`,
		`<span class="tag">Effect</span>`,
	} {
		if !strings.Contains(got, w) {
			t.Errorf("card missing %q\n--- got ---\n%s", w, got)
		}
	}
	// Document order: Effect, then the breath roll, then the claw roll.
	effectIdx := strings.Index(got, ">Effect<")
	breathIdx := strings.Index(got, "5 fire damage")
	clawIdx := strings.Index(got, "3 + I damage")
	if effectIdx < 0 || breathIdx < 0 || clawIdx < 0 || !(effectIdx < breathIdx && breathIdx < clawIdx) {
		t.Errorf("expected Effect, then breath tiers, then claw tiers, in that order:\n%s", got)
	}
}

// Instantaneous Excavation states its Effect BEFORE the power roll. Per the
// SC-310 attachment rule (mirroring SC-308) the roll must nest inside/below
// the Effect section it attaches to — it must NOT hoist above the Effect (the
// pre-fix behavior: a single global power roll rendered right after the
// rail, before every section).
func TestRenderAbilityCard_EffectBeforeRollNested(t *testing.T) {
	fm := "action_type: Maneuver\nname: Instantaneous Excavation\ntype: ability"
	body := `
| **Earth, Magic** | **Maneuver** |
| --- | ---: |
| **Ranged 10** | **Special** |

**Effect:** You open up two holes with 1-square openings.

**Power Roll + Reason:**

- **≤11:** The target shifts 1 square.
- **12-16:** The target falls in.
- **17+:** The target falls in and is restrained.
`
	got := renderAbilityCard(fm, body, "")
	if n := strings.Count(got, `class="sc-ability__section"`); n != 1 {
		t.Fatalf("expected 1 section container (Effect, roll attached to it), got %d\n--- got ---\n%s", n, got)
	}
	if n := strings.Count(got, `class="sc-ability__pr"`); n != 1 {
		t.Fatalf("expected 1 power-roll panel, got %d\n--- got ---\n%s", n, got)
	}
	effectIdx := strings.Index(got, ">Effect<")
	rollIdx := strings.Index(got, `class="sc-ability__pr"`)
	if effectIdx < 0 || rollIdx < 0 || rollIdx < effectIdx {
		t.Errorf("the power-roll panel must not hoist above the Effect section it attaches to:\n%s", got)
	}
	if !strings.Contains(got, `data-tier="low"><span class="badge">!</span><span class="res">The target shifts 1 square.</span>`) {
		t.Errorf("tier1 missing:\n%s", got)
	}
}

func TestRenderAbilityCard_TriggeredCostAndSections(t *testing.T) {
	fm := "action_type: Triggered\ncost: 11 Wrath\nname: Fulfill Your Destiny\ntype: ability"
	body := `
*You have looked at various futures.*

| **Magic, Ranged**  |   **Triggered** |
|--------------------|----------------:|
| **📏 Ranged 10**   | **🎯 One ally** |

**Trigger:** You or another hero ends their turn.

**Effect:** The target takes their turn after the triggering hero.
`
	got := renderAbilityCard(fm, body, "")
	wants := []string{
		`data-action="triggered"`,
		`<span class="sc-ability__glyph">)</span>`,
		`sc-head__right-primary sc-head__slot--mini">11 Wrath</div>`, // cost is now the right-primary mini-title
		`<span class="tag">Trigger</span>`,
		`<span class="tag">Effect</span>`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("card missing %q\n--- got ---\n%s", w, got)
		}
	}
	// Trigger must come before Effect (document order).
	if strings.Index(got, ">Trigger<") > strings.Index(got, ">Effect<") {
		t.Errorf("Trigger section should precede Effect section")
	}
}

// Keyword chips can carry SCC cross-reference links (Melee, Ranged, Strike, …).
// The card is raw HTML MkDocs never post-processes, so a chip's markdown link
// must be resolved to a real <a> (like flavor/tiers) — not left as literal
// "[Ranged](…)" text that renders verbatim in the browser.
func TestRenderAbilityCard_KeywordLinks(t *testing.T) {
	fm := "action_type: Main action\nname: Holy Strike\ntype: ability"
	body := `
| **[Melee](../../../rule/combat/melee.md), [Strike](../../../rule/combat/strike.md), Weapon** | **Main action** |
|----|----:|
| **📏 Melee 1** | **🎯 One creature** |
`
	got := renderAbilityCard(fm, body, "")
	if strings.Contains(got, "[Melee]") || strings.Contains(got, "[Strike]") {
		t.Errorf("keyword chip leaked literal markdown link syntax\n--- got ---\n%s", got)
	}
	wants := []string{
		`<span class="sc-ability__chip"><a href=`,
		`>Melee</a></span>`,
		`>Strike</a></span>`,
		`<span class="sc-ability__chip">Weapon</span>`, // plain keyword unchanged
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("card missing %q\n--- got ---\n%s", w, got)
		}
	}
}

func TestRenderAbilityCard_ProseTrait(t *testing.T) {
	fm := "name: Remember Your Oath\ntype: trait"
	body := `
As a maneuver, you can recite the following oath.

*Even should the sun stop in the sky.*
`
	got := renderAbilityCard(fm, body, "")
	wants := []string{
		`data-action="trait"`,
		`>Trait</div>`,
		`<p class="sc-ability__flavor">Even should the sun stop in the sky.</p>`,
		`<div class="sc-ability__section-body"><p>As a maneuver, you can recite the following oath.</p></div>`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("trait card missing %q\n--- got ---\n%s", w, got)
		}
	}
	// A pure-prose trait has no cost badge and no power-roll panel.
	if strings.Contains(got, "sc-ability__cost") {
		t.Errorf("trait should have no cost badge")
	}
	if strings.Contains(got, "sc-ability__pr") {
		t.Errorf("trait should have no power-roll panel")
	}
}

func TestRichInline_LinksAndBold(t *testing.T) {
	got := richInline("A [prone target](../../../../condition/prone.md), then **5 damage** & more")
	// Relative .md links become real anchors, .md → directory URL, plus one extra
	// "../" for use_directory_urls depth on a standalone (non-index) page.
	if !strings.Contains(got, `<a href="../../../../../condition/prone/">prone target</a>`) {
		t.Errorf("relative link should resolve to a directory-URL anchor with +1 depth: %q", got)
	}
	if strings.Contains(got, ".md") {
		t.Errorf("no raw .md target should survive: %q", got)
	}
	if !strings.Contains(got, "<b>5 damage</b>") {
		t.Errorf("bold should render: %q", got)
	}
	if !strings.Contains(got, "&amp;") {
		t.Errorf("ampersand should be escaped: %q", got)
	}
}

func TestRenderAbilityCard_MultiParagraphEffectOneContainer(t *testing.T) {
	fm := "action_type: Maneuver\nname: Judgment\ntype: ability"
	body := `
**Effect:** The target is judged by you.

Whenever a creature judged by you uses a main action, you can react.

Additionally, you can spend 1 wrath to take one of the following:

- When an adjacent creature shifts, you make a free strike.
- When a creature makes a power roll, they take a bane.

You can choose only one option at a time.
`
	got := renderAbilityCard(fm, body, "")
	// Exactly one section container (the Effect), holding every paragraph + the list.
	if n := strings.Count(got, `class="sc-ability__section"`); n != 1 {
		t.Fatalf("expected 1 section container, got %d\n--- got ---\n%s", n, got)
	}
	if n := strings.Count(got, `class="sc-ability__section-head"`); n != 1 {
		t.Errorf("expected exactly 1 section head (Effect), got %d", n)
	}
	for _, w := range []string{
		"<span class=\"tag\">Effect</span>",
		"<p>The target is judged by you.</p>",
		"<p>Whenever a creature judged by you uses a main action, you can react.</p>",
		"<ul><li>When an adjacent creature shifts, you make a free strike.</li>",
		"<li>When a creature makes a power roll, they take a bane.</li></ul>",
		"<p>You can choose only one option at a time.</p>",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("section body missing %q\n--- got ---\n%s", w, got)
		}
	}
}

func TestCardHref_ExternalAndAnchorPassThrough(t *testing.T) {
	for _, target := range []string{"https://example.com", "#frag", "mailto:a@b.com"} {
		if got := cardHref(target); got != target {
			t.Errorf("cardHref(%q) = %q, want unchanged", target, got)
		}
	}
}

func TestAbilityCard_SixSlotHead(t *testing.T) {
	fm := "name: Black Ash Teleport\ntype: ability\naction_type: Maneuver\ncost: Signature\nclass: shadow\nsubclass: college-of-black-ash\nlevel: 1"
	got := renderAbilityCard(fm, "", "")
	for _, want := range []string{
		`<header class="sc-head">`,
		`sc-head__left-eyebrow sc-head__slot--line">Ability</div>`,
		`sc-head__left-primary sc-head__slot--line">Black Ash Teleport</h3>`,
		`sc-head__left-deck sc-head__slot--line">Shadow · College Of Black Ash</div>`,
		`sc-head__right-eyebrow sc-head__slot--chip">Level 1</div>`,
		`sc-head__right-primary sc-head__slot--mini">Signature</div>`,
		`sc-head__right-deck sc-head__slot--chip">Maneuver</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestAbilityCard_KitSignatureNameSplit(t *testing.T) {
	// A kit signature ability carries the combined "Kit (Ability)" name plus a
	// `kit` field: the card shows the ability name as the primary and the kit name
	// as the deck.
	fm := "name: Battlemind (Unmooring)\ntype: ability\naction_type: Main action\ncost: Signature\nsubtype: signature\nkit: battlemind"
	got := renderAbilityCard(fm, "", "")
	for _, want := range []string{
		`sc-head__left-primary sc-head__slot--line">Unmooring</h3>`,
		`sc-head__left-deck sc-head__slot--line">Battlemind</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Battlemind (Unmooring)") {
		t.Errorf("combined name should be split, not shown verbatim:\n%s", got)
	}
}
