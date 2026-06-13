package site

import (
	"strings"
	"testing"
)

const fixtureBarrowGates = `---
name: Barrow Gates
role: Defender
size: "2"
stamina: 20 + your level
statblock_kind: fixture
terrain_type: Fortification
type: statblock
---

*Fortification Defender*

| **Stamina:** 20 + your level | **Size:** 2 |
|------------------------------|------------:|

> ⭐️ **The Bell Tolls**
>
> Each enemy that starts their turn within 3 squares is frightened.

> ⭐️ **Undead Dominion**
>
> Each undead minion has damage immunity 2 within 3 squares.

> **Level 5 Fixture Advancement Feature**
>
> ⭐️ **Memento Mori**
>
> You gain a surge the first time a minion dies.

> **Level 9 Fixture Advancement Feature**
>
> ⭐️ **Size Increase**
>
> The gates are now size 3.
>
> ⭐️ **Open the Gates**
>
> You can use Rise! as a free triggered action.
`

func TestBuildFixturePage(t *testing.T) {
	out, ok := buildFixturePage([]byte(fixtureBarrowGates))
	if !ok {
		t.Fatal("buildFixturePage returned ok=false for a fixture")
	}
	got := string(out)
	// frontmatter preserved
	if !strings.Contains(got, "statblock_kind: fixture") {
		t.Error("frontmatter not preserved")
	}
	// Forged Band card, role-keyed
	for _, want := range []string{
		`<div class="fb-wrap" data-role="defender"`,
		"Fortification · Defender", // eyebrow: "Fortification · Defender"
		"Barrow Gates",
		// loose stats from the 2-col grid
		`<div class="fb__stat-l">Stamina</div>`,
		`<div class="fb__stat-l">Size</div>`,
		// base features
		"The Bell Tolls", "Undead Dominion",
		// advancement groups, leveled
		`data-level="5"`, "Memento Mori",
		`data-level="9"`, "Size Increase", "Open the Gates",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}
	// the redundant raw italic role line and broken 2-col grid table are gone
	if strings.Contains(got, "*Fortification Defender*") {
		t.Error("raw italic role line leaked into the card body")
	}
}

func TestBuildFixturePage_NonFixturePassesThrough(t *testing.T) {
	// a normal creature statblock must NOT be handled here
	creature := "---\nname: Goblin\ntype: statblock\nrole: Minion\n---\n\nbody\n"
	if _, ok := buildFixturePage([]byte(creature)); ok {
		t.Error("buildFixturePage handled a non-fixture statblock")
	}
	// a featureblock must NOT be handled here
	fb := "---\nname: X\ntype: featureblock\n---\n\nbody\n"
	if _, ok := buildFixturePage([]byte(fb)); ok {
		t.Error("buildFixturePage handled a featureblock")
	}
}
