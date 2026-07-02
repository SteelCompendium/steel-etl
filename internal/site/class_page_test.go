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
strong_potency: '[Might](../rule/character/might.md)'
average_potency: '[Might](../rule/character/might.md) − 1'
weak_potency: '[Might](../rule/character/might.md) − 2'
type: class
---

Intro prose.

## Basics

body

## 1st-Level Features

body

## Stormwight Kits

body
`

func TestBuildClassLandingPage(t *testing.T) {
	out, ok := buildClassLandingPage([]byte(classPageFixture))
	if !ok {
		t.Fatal("class page not transformed")
	}
	s := string(out)
	for _, want := range []string{
		`<section class="sc-classhead">`,
		`sc-head__left-primary`, // renderCardHead emitted the name slot
		`>Fury</h2>`,            // name as h2
		`Draw Steel: Heroes`,    // left deck
		`sc-classhead__pot`,     // potency strip
		`Might − 2`,             // weak potency, link-stripped
		`<nav class="sc-classnav"`,
		`<a href="#basics">Basics</a>`,
		`<a href="#1st-level-features">1st-Level Features</a>`,
		`<a href="#stormwight-kits">Stormwight Kits</a>`,
		"Intro prose.", // body preserved
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q", want)
		}
	}

	// non-class pages pass through
	if _, ok := buildClassLandingPage([]byte("---\ntype: ability\nname: X\n---\nbody\n")); ok {
		t.Error("ability page must not be transformed")
	}

	// no potency frontmatter → no potency strip (beastheart)
	noPot := strings.NewReplacer(
		"strong_potency: '[Might](../rule/character/might.md)'\n", "",
		"average_potency: '[Might](../rule/character/might.md) − 1'\n", "",
		"weak_potency: '[Might](../rule/character/might.md) − 2'\n", "",
	).Replace(classPageFixture)
	out2, _ := buildClassLandingPage([]byte(noPot))
	if strings.Contains(string(out2), "sc-classhead__pot") {
		t.Error("potency strip must be omitted when frontmatter lacks potencies")
	}
}
