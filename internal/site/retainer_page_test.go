package site

import (
	"strings"
	"testing"
)

// A trimmed but verbatim slice of the generated Goblin Guide body: two base
// features (one tabled ability, one passive) then two advancement tiers.
const goblinGuideBody = `###### Goblin Guide

| Goblin, Humanoid | Level 1 | Harrier Retainer |

> 🗡 **Stabbity Stab (Signature Ability)**
>
> **Effect:** The target can't make opportunity attacks until the end of the guide's turn.

> ⭐️ **Crafty**
>
> The guide doesn't provoke opportunity attacks by moving.

###### Level 4 Retainer Advancement Ability

> 🗡 **Weaving Knives (Encounter)**
>
> **Effect:** The guide shifts up to their speed before and after the strike.

###### Level 7 Retainer Advancement Ability

> 🗡 **Sneak and Stab (Encounter)**
>
> **Effect:** If the guide is hidden from the target, this ability has a double edge.`

func TestSplitRetainerAdvancement(t *testing.T) {
	base, groups := splitRetainerAdvancement(goblinGuideBody)

	if want := "⭐️ **Crafty**"; !strings.Contains(base, want) {
		t.Errorf("base should keep the base passive %q", want)
	}
	if dont := "Weaving Knives"; strings.Contains(base, dont) {
		t.Errorf("base must NOT contain advancement ability %q", dont)
	}
	if len(groups) != 2 {
		t.Fatalf("want 2 advancement groups, got %d", len(groups))
	}
	if groups[0].Level != 4 || groups[1].Level != 7 {
		t.Errorf("want levels [4 7], got [%d %d]", groups[0].Level, groups[1].Level)
	}
	if !strings.Contains(groups[0].Body, "Weaving Knives") {
		t.Errorf("group 0 body missing its ability: %q", groups[0].Body)
	}
	if strings.Contains(groups[0].Body, "Sneak and Stab") {
		t.Errorf("group 0 body leaked the level-7 ability")
	}
	if strings.Contains(groups[0].Body, "Retainer Advancement Ability") {
		t.Errorf("group body should not include the heading line")
	}
}

func TestSplitRetainerAdvancement_NoHeadings(t *testing.T) {
	body := "> 🗡 **Just A Monster**\n>\n> **Effect:** nothing special."
	base, groups := splitRetainerAdvancement(body)
	if base != body {
		t.Errorf("base should be the whole body unchanged, got %q", base)
	}
	if groups != nil {
		t.Errorf("non-retainer statblock should yield no groups, got %v", groups)
	}
}
