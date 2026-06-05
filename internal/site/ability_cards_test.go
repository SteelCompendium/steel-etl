package site

import (
	"strings"
	"testing"
)

func TestBuildAbilityCardPage_NonAbilityUnchanged(t *testing.T) {
	in := []byte("---\ntype: kit\nname: Mountain\n---\n\nbody\n")
	out, ok := buildAbilityCardPage(in)
	if ok {
		t.Fatalf("expected ok=false for type: kit")
	}
	if string(out) != string(in) {
		t.Fatalf("non-ability data should pass through unchanged")
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
	got := renderAbilityCard(fm, body)
	wants := []string{
		`data-action="main"`,
		`<span class="sc-ability__glyph">l</span>`,
		`>Main Action</div>`,
		`<h3 class="sc-ability__name">Dragon Breath</h3>`,
		`<div class="sc-ability__cost">Signature</div>`,
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
	got := renderAbilityCard(fm, body)
	wants := []string{
		`data-action="triggered"`,
		`<span class="sc-ability__glyph">)</span>`,
		`<div class="sc-ability__cost"><span class="num">11</span> Wrath</div>`, // numeric prefix in mono
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

func TestRenderAbilityCard_ProseTrait(t *testing.T) {
	fm := "name: Remember Your Oath\ntype: trait"
	body := `
As a maneuver, you can recite the following oath.

*Even should the sun stop in the sky.*
`
	got := renderAbilityCard(fm, body)
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
	got := renderAbilityCard(fm, body)
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
