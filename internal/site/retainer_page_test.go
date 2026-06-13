package site

import (
	"encoding/json"
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

**Level 4 Retainer Advancement Ability**

> 🗡 **Weaving Knives (Encounter)**
>
> **Effect:** The guide shifts up to their speed before and after the strike.

**Level 7 Retainer Advancement Ability**

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

func TestSplitRetainerAdvancement_HeadingForm(t *testing.T) {
	body := "> base feature\n\n###### Level 4 Retainer Advancement Ability\n\n> 🗡 **Adv Ability**\n>\n> **Effect:** does a thing."
	base, groups := splitRetainerAdvancement(body)
	if strings.Contains(base, "Adv Ability") {
		t.Errorf("heading-form advancement should be split out of base")
	}
	if len(groups) != 1 || groups[0].Level != 4 {
		t.Fatalf("want 1 group at level 4, got %v", groups)
	}
	if !strings.Contains(groups[0].Body, "Adv Ability") {
		t.Errorf("group body should contain the advancement ability")
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

func TestRetainerRoleKey(t *testing.T) {
	// Real site input: singular role scalar.
	if got := retainerRoleKey("role: Harrier\norganization: Retainer\n"); got != "harrier" {
		t.Errorf("scalar role: want harrier, got %q", got)
	}
	// Defensive fallback: roles list (md-dse-linked variant).
	if got := retainerRoleKey("roles:\n  - Support Retainer\n"); got != "support" {
		t.Errorf("roles-list fallback: want support, got %q", got)
	}
	if got := retainerRoleKey("role: Bogus\n"); got != "" {
		t.Errorf("unknown role should snap to empty, got %q", got)
	}
	if got := retainerRoleKey("name: x\n"); got != "" {
		t.Errorf("no role should yield empty, got %q", got)
	}
}

func TestRenderRetainerAdvancement(t *testing.T) {
	fm := "name: Goblin Guide\nrole: Harrier\norganization: Retainer\n"
	_, groups := splitRetainerAdvancement(goblinGuideBody)
	out := renderRetainerAdvancement(fm, groups)

	for _, want := range []string{
		`class="fb-wrap"`, `data-role="harrier"`,
		"Advancement Abilities", // card name
		"Harrier Retainer",      // eyebrow
		`class="fb__band--adv" data-level="4"`,
		"Level 4 Advancement", // adv sub-head
		"Weaving Knives",      // the level-4 ability
		`data-level="7"`, "Sneak and Stab",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("advancement card missing %q\n---\n%s", want, out)
		}
	}
	// The card must NOT contain the base features (those stay in the island).
	if strings.Contains(out, "Stabbity Stab") {
		t.Errorf("advancement card leaked a base feature")
	}
}

func TestRenderRetainerAdvancement_Empty(t *testing.T) {
	if out := renderRetainerAdvancement("name: x\n", nil); out != "" {
		t.Errorf("no groups should render nothing, got %q", out)
	}
}

func TestBuildStatblockIslandPage_RetainerSplit(t *testing.T) {
	page := "---\nname: Goblin Guide\ntype: statblock\nrole: Harrier\norganization: Retainer\n---\n\n" + goblinGuideBody
	out, ok := buildStatblockIslandPage([]byte(page))
	if !ok {
		t.Fatal("retainer statblock should be handled")
	}
	s := string(out)

	// 1. The advancement card is appended.
	if !strings.Contains(s, `class="fb-wrap"`) || !strings.Contains(s, "Weaving Knives") {
		t.Errorf("page should contain the advancement card")
	}
	// 2. The island JSON must NOT include the advancement abilities.
	marker := `class="sc-statblock-data">`
	start := strings.Index(s, marker)
	if start < 0 {
		t.Fatal("island script not found")
	}
	jsonStart := start + len(marker)
	jsonEnd := strings.Index(s[jsonStart:], "</script>")
	islandJSON := strings.TrimSpace(s[jsonStart : jsonStart+jsonEnd])
	var island struct {
		Features []struct {
			Name string `json:"name"`
		} `json:"features"`
	}
	if err := json.Unmarshal([]byte(islandJSON), &island); err != nil {
		t.Fatalf("island JSON parse: %v\n%s", err, islandJSON)
	}
	names := map[string]bool{}
	for _, f := range island.Features {
		names[f.Name] = true
	}
	if !names["Crafty"] {
		t.Errorf("island should keep base feature Crafty; got %v", names)
	}
	if names["Weaving Knives"] || names["Sneak and Stab"] {
		t.Errorf("island must NOT include advancement abilities; got %v", names)
	}
}
