package site

import (
	"strings"
	"testing"
)

func TestPySlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Basics", "basics"},
		{"1st-Level Features", "1st-level-features"},
		{"Stormwight Kits", "stormwight-kits"},
		{"Gods & Religion", "gods-religion"},
	}
	for _, c := range cases {
		if got := pySlugify(c.in); got != c.want {
			t.Errorf("pySlugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHeadingText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"## Basics", "Basics"},
		{`## 2nd-Level Features {data-scc="mcdm.heroes.v1/x/y"}`, "2nd-Level Features"},
		{"## [Kits](../kit/index.md)", "Kits"},
	}
	for _, c := range cases {
		if got := headingText(c.in); got != c.want {
			t.Errorf("headingText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

const classPageFixture = `---
name: Fury
printing_book: "Draw Steel: Heroes"
scc: mcdm.heroes.v1/class/fury
flavor: You do not temper the heat of battle within you. You unleash it!
strong_potency: '[Might](../rule/character/might.md)'
average_potency: '[Might](../rule/character/might.md) − 1'
weak_potency: '[Might](../rule/character/might.md) − 2'
primary_characteristics:
    - Might
    - Agility
starting_stamina: 21
stamina_per_level: 9
recoveries: 10
skills:
    - 'You gain the [Nature](../skill/lore/nature.md) skill.'
type: class
---

You do not temper the heat of battle within you. You [unleash](../feature/x.md) it!

**Bold summary stays.**

## Basics

body

## 1st-Level Features

body

## 2nd-Level Features

body

## Stormwight Kits

body
`

func TestBuildClassLandingPage(t *testing.T) {
	out, ok := buildClassLandingPage([]byte(classPageFixture), nil)
	if !ok {
		t.Fatal("class page not transformed")
	}
	s := string(out)
	for _, want := range []string{
		`<section class="sc-classhead">`,
		`sc-head__left-primary`, // renderCardHead emitted the name slot
		`>Fury</h2>`,            // name as h2
		`sc-head__right-eyebrow sc-head__slot--chip">Draw Steel: Heroes`, // book chip
		`sc-head__right-primary sc-head__slot--mini">Might · Agility`,    // primaries mini
		`>primary characteristics<`, // rail deck = the mini's caption
		`sc-classhead__flavor">You do not temper`, // flavor inside the card
		`sc-classhead__stats`,                     // base-stat strip
		`>21</span>`,                              // starting stamina
		`>+9</span>`,                              // stamina per level
		`>10</span>`,                              // recoveries
		`sc-classhead__pot`,                       // potency strip
		`Might − 2`,                               // weak potency, link-stripped
		`sc-classhead__skills`,                    // skills footer
		`You gain the Nature skill.`,              // link-stripped skills prose
		`<nav class="sc-classnav"`,
		`<a href="#basics">Basics</a>`,
		`sc-classnav__lvls`, // level headings collapse into the numbered group
		`<a href="#1st-level-features" title="1st-Level Features">1</a>`,
		`<a href="#2nd-level-features" title="2nd-Level Features">2</a>`,
		`<a href="#stormwight-kits">Stormwight Kits</a>`,
		"**Bold summary stays.**", // body preserved
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q", want)
		}
	}

	// the body's opening paragraph duplicated the card flavor → dropped
	// (the linked "[unleash]" form exists only in that body paragraph)
	if strings.Contains(s, "[unleash]") {
		t.Error("duplicate flavor paragraph must be dropped from the body")
	}
	// only one "Nth-Level Features" text pill replacement (group, not ten pills)
	if strings.Contains(s, `>1st-Level Features</a>`) {
		t.Error("level headings must not render as individual text pills")
	}

	// non-class pages pass through
	if _, ok := buildClassLandingPage([]byte("---\ntype: ability\nname: X\n---\nbody\n"), nil); ok {
		t.Error("ability page must not be transformed")
	}

	// no potency frontmatter → no potency strip (beastheart)
	noPot := strings.NewReplacer(
		"strong_potency: '[Might](../rule/character/might.md)'\n", "",
		"average_potency: '[Might](../rule/character/might.md) − 1'\n", "",
		"weak_potency: '[Might](../rule/character/might.md) − 2'\n", "",
	).Replace(classPageFixture)
	out2, _ := buildClassLandingPage([]byte(noPot), nil)
	if strings.Contains(string(out2), "sc-classhead__pot") {
		t.Error("potency strip must be omitted when frontmatter lacks potencies")
	}
}
