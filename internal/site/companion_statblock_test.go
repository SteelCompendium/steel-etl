package site

import (
	"os"
	"path/filepath"
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
	// The italic flavor line ("*The panther bunches up…*") precedes the spec
	// table as bare prose, so it becomes its own effects[0] entry (a
	// pre-existing quirk — the text wasn't lost before either, just silently
	// joined into Trailing; SC-308 round 3b's document-order model makes it a
	// visible, separate entry instead).
	if len(pounce.Effects) != 3 {
		t.Fatalf("pounce effects = %+v, want 3 entries (flavor, Effect, Spend enhancement)", pounce.Effects)
	}
	if pounce.Effects[1].Name != "Effect" {
		t.Errorf("pounce effects[1] = %+v, want the Effect entry", pounce.Effects[1])
	}
	if !strings.Contains(pounce.Effects[2].Cost, "Spend 1 Ferocity") {
		t.Errorf("pounce effects[2] = %+v, want the Spend cost entry", pounce.Effects[2])
	}
	spring := feats[1]
	if spring.Name != "Mighty Spring" || spring.Kind != "passive" || spring.Body == "" {
		t.Errorf("mighty spring = %+v", spring)
	}
}

func TestBuildCompanionStatblockIsland_Panther(t *testing.T) {
	fm := "name: Panther\nlevel: 1\ncompanion: panther\ntype: statblock\nscc: mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther"
	d := buildCompanionStatblockIsland(fm, pantherCompanionBody)
	if d.Name != "Panther" || d.ID != "panther" {
		t.Errorf("name/id = %q/%q", d.Name, d.ID)
	}
	if d.Eyebrow != "Animal, Companion" || d.Level != "1" {
		t.Errorf("eyebrow/level = %q/%q", d.Eyebrow, d.Level)
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
level: 1
name: Panther
scc: mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther
type: statblock
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
	if !strings.HasPrefix(s, "---\n") || !strings.Contains(s, "type: statblock") {
		t.Error("frontmatter not preserved verbatim")
	}
	if !strings.Contains(s, "## Panther Advancement Features") || !strings.Contains(s, "### Cat and Mouse") {
		t.Error("advancement-features section was dropped")
	}
	if _, hit := companionStatblockCache["mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther"]; !hit {
		t.Error("companion island not cached by scc")
	}
}

// Since SC-390 a companion page is `type: statblock` and carries the stats in
// frontmatter, which is exactly what the generic statblock page would claim. The
// companion adapter (SCC-gated, body-table driven) must win in buildSection so the
// card keeps its Skills slot and ability sections; the generic path must NOT run
// (it would cache the page's features and render a features-less card).
func TestBuild_CompanionStatblockTakesCompanionPath(t *testing.T) {
	const code = "mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther"
	srcDir := t.TempDir()
	docsDir := filepath.Join(t.TempDir(), "docs")
	page := "---\nname: Panther\nscc: " + code + "\ntype: statblock\nlevel: 1\ncompanion: panther\n" +
		"organization: Companion\nsize: 1M\nspeed: 7\nstamina: = yours\nstability: 1\nfree_strike: 1 + M\n" +
		"movement: Climb\nskills:\n    - Sneak\nmight: 2\nagility: 2\nreason: -1\nintuition: 2\npresence: 1\n---\n\n" +
		pantherCompanionBody + "\n\n## Panther Advancement Features {data-scc=\"x/monster.companion.beastheart.advancement-features/panther\"}\n\nAdvancement body."
	path := filepath.Join(srcDir, "monster/companion/beastheart/statblock/panther.md")
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(page), 0644)

	result, err := Build(&Config{
		SourceDir: srcDir, DocsDir: docsDir,
		Sections: []SectionConfig{{Name: "Browse", Include: []string{"monster/"}}},
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	for _, e := range result.Errors {
		t.Errorf("Error: %s", e)
	}
	data, err := os.ReadFile(filepath.Join(docsDir, "Browse/monster/companion/beastheart/panther.md"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if !strings.Contains(out, `class="sb-wrap"`) || !strings.Contains(out, "Skills") || !strings.Contains(out, "Pounce") || !strings.Contains(out, "Mighty Spring") {
		t.Errorf("expected the companion card with Skills + both abilities:\n%s", out)
	}
	if !strings.Contains(out, "type: statblock") || !strings.Contains(out, "## Panther Advancement Features") {
		t.Errorf("frontmatter type / advancement section not preserved:\n%s", out)
	}
	if _, hit := companionStatblockCache[code]; !hit {
		t.Error("companion adapter did not run (island not cached)")
	}
	if _, hit := statblockFeatureCache[code]; hit {
		t.Error("generic statblock page ran for a companion (feature cache populated)")
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
	// Skills keeps its markdown link (resolved later by richSb at render).
	if !strings.Contains(g.cells["Skills"], "[Sneak](") {
		t.Errorf("skills cell = %q, want a Sneak link", g.cells["Skills"])
	}
}
