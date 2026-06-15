package site

import (
	"strings"
	"testing"
)

// pantherCompanionBody is the verbatim base region (table + abilities, no
// advancement section) of a generated companion leaf body.
const pantherCompanionBody = `| Animal, Companion |           -           |                        Level 1                        |          -          |            -             |
|:-----------------:|:---------------------:|:-----------------------------------------------------:|:-------------------:|:------------------------:|
|  **1M**<br>Size   |    **7**<br>Speed     |                **= yours**<br>Stamina                 | **1**<br>Stability  | **1 + M**<br>Free Strike |
| **—**<br>Immunity | **Climb**<br>Movement | **[Sneak](../../../skill/intrigue/sneak.md)**<br>Skills |                     |                          |
|  **+2**<br>Might  |   **+2**<br>Agility   |                   **−1**<br>Reason                    | **+2**<br>Intuition |    **+1**<br>Presence    |

## Pounce {data-scc="mcdm.beastheart.v1/feature.ability.companion.beastheart.panther.level-1/pounce"}

*The panther bunches up, then uncoils into a deadly leap.*

| **Companion, Melee, Weapon** |     **Maneuver** |
|------------------------------|-----------------:|
| **📏 Melee 1**               | **🎯 One enemy** |

**Effect:** The target takes damage equal to 3 + the panther's Might score.

**Spend 1 Ferocity:** The panther can jump up to a number of squares equal to their speed.

## Mighty Spring {data-scc="mcdm.beastheart.v1/feature.companion.beastheart.panther.level-1/mighty-spring"}

Whenever the panther takes the Advance move action, they can jump up to a number of squares equal to their speed.`

func TestCompanionFeatures_Panther(t *testing.T) {
	feats := companionFeatures(pantherCompanionBody)
	if len(feats) != 2 {
		t.Fatalf("features = %d, want 2", len(feats))
	}
	pounce := feats[0]
	if pounce.Name != "Pounce" || pounce.Action != "maneuver" || pounce.Kind != "ability" {
		t.Errorf("pounce name/action/kind = %q/%q/%q", pounce.Name, pounce.Action, pounce.Kind)
	}
	if strings.Join(pounce.Keywords, ",") != "Companion,Melee,Weapon" {
		t.Errorf("pounce keywords = %v", pounce.Keywords)
	}
	if pounce.Distance != "Melee 1" || pounce.Target != "One enemy" {
		t.Errorf("pounce dist/target = %q/%q", pounce.Distance, pounce.Target)
	}
	if len(pounce.Sections) == 0 || pounce.Sections[0].Label != "Effect" {
		t.Errorf("pounce sections = %+v", pounce.Sections)
	}
	if len(pounce.Enhancements) == 0 || !strings.Contains(pounce.Enhancements[0].Cost, "Spend 1 Ferocity") {
		t.Errorf("pounce enhancements = %+v", pounce.Enhancements)
	}
	spring := feats[1]
	if spring.Name != "Mighty Spring" || spring.Kind != "passive" || spring.Body == "" {
		t.Errorf("mighty spring = %+v", spring)
	}
}

func TestBuildCompanionStatblockIsland_Panther(t *testing.T) {
	fm := "name: Panther\nlevel: \"1\"\ncompanion: panther\ntype: feature-group\nscc: mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther"
	d := buildCompanionStatblockIsland(fm, pantherCompanionBody)
	if d.Name != "Panther" || d.ID != "panther" {
		t.Errorf("name/id = %q/%q", d.Name, d.ID)
	}
	if d.Ancestry != "Animal, Companion" || d.Level != "1" {
		t.Errorf("ancestry/level = %q/%q", d.Ancestry, d.Level)
	}
	if d.Role != "Companion" || d.RoleKey != "leader" {
		t.Errorf("role/roleKey = %q/%q, want Companion/leader", d.Role, d.RoleKey)
	}
	if d.EV != "" {
		t.Errorf("ev = %q, want empty", d.EV)
	}
	if len(d.Defenses) != 5 || d.Defenses[0].V != "1M" || d.Defenses[2].V != "= yours" {
		t.Fatalf("defenses = %+v", d.Defenses)
	}
	if d.Meta.Movement != "Climb" || d.Meta.Captain.Label != "Skills" {
		t.Errorf("meta = %+v", d.Meta)
	}
	if !strings.Contains(d.Meta.Captain.Value, "[Sneak](") {
		t.Errorf("skills value = %q", d.Meta.Captain.Value)
	}
	wantChars := map[string]string{"Might": "+2", "Reason": "−1", "Presence": "+1"}
	for _, c := range d.Characteristics {
		if w, ok := wantChars[c.L]; ok && c.V != w {
			t.Errorf("char %s = %q, want %q", c.L, c.V, w)
		}
	}
	if len(d.Features) != 2 {
		t.Fatalf("features = %d, want 2", len(d.Features))
	}
	if d.Features[0].Name != "Pounce" || d.Features[1].Name != "Mighty Spring" {
		t.Errorf("feature names = %q / %q", d.Features[0].Name, d.Features[1].Name)
	}
}

func TestBuildCompanionStatblockPage_Panther(t *testing.T) {
	page := `---
companion: panther
level: "1"
name: Panther
scc: mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther
type: feature-group
---

# Panther

---

` + pantherCompanionBody + `

## Panther Advancement Features {data-scc="mcdm.beastheart.v1/monster.companion.beastheart.advancement-features/panther"}

### Cat and Mouse {data-scc="mcdm.beastheart.v1/feature.companion.beastheart.panther.level-3/cat-and-mouse"}

Whenever the panther makes a strike while rampaging, the panther can knock the target prone.`

	statblockFeatureCache = map[string][]sbFeature{}
	companionStatblockCache = map[string]sbIsland{}

	out, ok := buildCompanionStatblockPage([]byte(page))
	if !ok {
		t.Fatal("buildCompanionStatblockPage returned ok=false for a companion page")
	}
	s := string(out)
	if !strings.Contains(s, `class="sb-wrap"`) || strings.Contains(s, "<br>Size") {
		t.Errorf("expected .sb-wrap card and no raw stat table; got:\n%s", s)
	}
	if !strings.HasPrefix(s, "---\n") || !strings.Contains(s, "type: feature-group") {
		t.Error("frontmatter not preserved verbatim")
	}
	if !strings.Contains(s, "## Panther Advancement Features") || !strings.Contains(s, "### Cat and Mouse") {
		t.Error("advancement-features section was dropped")
	}
	if _, hit := companionStatblockCache["mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther"]; !hit {
		t.Error("companion island not cached by scc")
	}
}

func TestBuildCompanionStatblockPage_NonCompanion(t *testing.T) {
	page := "---\ntype: statblock\nscc: mcdm.monsters.v1/monster.devils.statblock/x\n---\n\nbody"
	if _, ok := buildCompanionStatblockPage([]byte(page)); ok {
		t.Error("non-companion page must return ok=false")
	}
}

func TestParseCompanionGrid_Panther(t *testing.T) {
	g := parseCompanionGrid(pantherCompanionBody)
	if g.keywords != "Animal, Companion" {
		t.Errorf("keywords = %q", g.keywords)
	}
	if g.level != "1" {
		t.Errorf("level = %q, want 1", g.level)
	}
	want := map[string]string{
		"Size": "1M", "Speed": "7", "Stamina": "= yours", "Stability": "1",
		"Free Strike": "1 + M", "Immunity": "—", "Movement": "Climb",
		"Might": "+2", "Agility": "+2", "Reason": "−1", "Intuition": "+2", "Presence": "+1",
	}
	for k, v := range want {
		if g.cells[k] != v {
			t.Errorf("cell[%q] = %q, want %q", k, g.cells[k], v)
		}
	}
	// Skills keeps its markdown link (resolved later by resolveSbLinks).
	if !strings.Contains(g.cells["Skills"], "[Sneak](") {
		t.Errorf("skills cell = %q, want a Sneak link", g.cells["Skills"])
	}
}
