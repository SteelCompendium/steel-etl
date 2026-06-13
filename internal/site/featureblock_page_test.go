package site

import (
	"strings"
	"testing"
)

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
		`class="fb md-typeset"`, `class="fb__head"`,
		`class="fb__eyebrow"`, "Malice Features",
		`class="fb__name"`, "Basilisk Malice",
		`class="fb__flavor"`, "spend Malice to activate",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
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
		`data-role="hexer"`, "Level 2 Hazard · Hexer",
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
		`class="fb__feat-name`, "Walleye",
		`class="sc-ability__cost"`, "Malice", // cost badge "7 Malice"
		`class="fb__feat-body"`, "reflective spittle",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
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
