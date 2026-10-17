package output

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

func sampleStatblock() *content.ParsedContent {
	body := "" +
		"| Goblin, Humanoid | - | Level 1 | Horde Hexer | EV 3 |\n" +
		"|:--:|:--:|:--:|:--:|:--:|\n" +
		"| **1S**<br>Size | **5**<br>Speed | **10**<br>Stamina | **0**<br>Stability | **1**<br>Free Strike |\n" +
		"| **-**<br>Immunity | **Climb**<br>Movement | - | **-**<br>With Captain | **-**<br>Weakness |\n" +
		"| **-2**<br>Might | **+1**<br>Agility | **0**<br>Reason | **+2**<br>Intuition | **0**<br>Presence |\n\n" +
		"> ⭐️ **Crafty**\n>\n> Doesn't provoke opportunity attacks by moving.\n"

	return &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Goblin Cursespitter", "type": "statblock", "level": 1,
			"role": "Hexer", "organization": "Horde", "ev": "3",
			"keywords": []string{"Goblin", "Humanoid"}, "stamina": "10",
			"speed": 5, "size": "1S", "stability": 0, "free_strike": 1,
			"might": -2, "agility": 1, "reason": 0, "intuition": 2, "presence": 0,
			"movement": "Climb",
		},
		Body: body,
	}
}

func TestTransformStatblock_FixtureFields(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "The Boil", "type": "statblock",
			"role": "Support", "stamina": "20 + your level", "size": "2",
			"statblock_kind": "fixture", "terrain_type": "Hazard",
		},
		Body: "> ⭐️ **Hunger Thrush**\n>\n> Text.\n",
	}
	out := TransformToSDKFormat("mcdm.summoner.v1/fixture.demon.statblock/the-boil", parsed)
	if out["statblock_kind"] != "fixture" {
		t.Errorf("statblock_kind = %v, want fixture", out["statblock_kind"])
	}
	if out["terrain_type"] != "Hazard" {
		t.Errorf("terrain_type = %v, want Hazard", out["terrain_type"])
	}
	// A normal (non-fixture) statblock must NOT gain these keys.
	normal := &content.ParsedContent{
		Frontmatter: map[string]any{"name": "Goblin", "type": "statblock"},
		Body:        "",
	}
	nout := TransformToSDKFormat("mcdm.monsters.v1/monster.goblins.statblock/goblin", normal)
	if _, ok := nout["statblock_kind"]; ok {
		t.Error("non-fixture statblock should not have statblock_kind")
	}
	if _, ok := nout["terrain_type"]; ok {
		t.Error("non-fixture statblock should not have terrain_type")
	}
}

func TestTransformStatblock(t *testing.T) {
	out := TransformToSDKFormat("mcdm.monsters.v1/monster.goblins.statblock/goblin-cursespitter", sampleStatblock())

	if out["type"] != "statblock" || out["name"] != "Goblin Cursespitter" {
		t.Fatalf("base fields wrong: %+v", out)
	}
	if out["role"] != "Hexer" || out["organization"] != "Horde" {
		t.Errorf("role/org: %v / %v", out["role"], out["organization"])
	}
	if out["level"] != 1 {
		t.Errorf("level: %v", out["level"])
	}
	feats, ok := out["features"].([]map[string]any)
	if !ok || len(feats) != 1 {
		t.Fatalf("features: got %v", out["features"])
	}
	if feats[0]["name"] != "Crafty" || feats[0]["feature_type"] != "trait" {
		t.Errorf("feature: %+v", feats[0])
	}
}

// companionFixture is a beastheart companion statblock with its attached coded
// Level-1 members (an ability then a feature, in book order), as the pipeline
// builds it. The advancement-features featureblock is deliberately NOT a member.
func companionFixture() *content.ParsedContent {
	pounce := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Pounce", "type": "ability", "subtype": "signature", "action_type": "Maneuver",
			"flavor": "The panther bunches up.", "distance": "Melee 1", "target": "One enemy",
			"keywords": []string{"Companion", "Melee"}, "level": "1",
			"effects": []map[string]any{{"name": "Effect", "effect": "The target takes damage."}},
			"scc":     "mcdm.beastheart.v1/feature.ability.companion.beastheart.panther.level-1/pounce",
		},
		TypePath: []string{"feature", "ability", "companion", "beastheart", "panther", "level-1"},
		ItemID:   "pounce",
	}
	spring := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Mighty Spring", "type": "feature", "level": "1",
			"scc": "mcdm.beastheart.v1/feature.companion.beastheart.panther.level-1/mighty-spring",
		},
		Body:     "They can jump.",
		TypePath: []string{"feature", "companion", "beastheart", "panther", "level-1"},
		ItemID:   "mighty-spring",
	}
	return &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Panther", "type": "statblock", "level": 1, "organization": "Companion",
			"keywords": []string{"Animal", "Companion"}, "size": "1M", "speed": 7,
			"stamina": "= yours", "stability": 1, "free_strike": "1 + M", "movement": "Climb",
			"skills": []string{"Sneak"}, "might": 2, "agility": 2, "reason": -1, "intuition": 2,
			"presence": 1, "companion": "panther",
		},
		Body:     "| Animal, Companion | - | Level 1 | - | - |\n|:-:|:-:|:-:|:-:|:-:|\n",
		TypePath: []string{"monster", "companion", "beastheart", "statblock"},
		ItemID:   "panther",
		Members:  []*content.ParsedContent{pounce, spring},
	}
}

// TestTransformStatblock_CompanionMembers: a companion's attached members become
// its features[] in book order, as the members' own SDK feature objects, with the
// ability's "Signature" worded "Signature Ability" like every statblock feature;
// text free_strike / stamina and the skills list pass through (SC-390).
func TestTransformStatblock_CompanionMembers(t *testing.T) {
	out := TransformToSDKFormat("mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther", companionFixture())

	if out["free_strike"] != "1 + M" || out["stamina"] != "= yours" || out["organization"] != "Companion" {
		t.Errorf("text stats not passed through: free_strike=%v stamina=%v org=%v", out["free_strike"], out["stamina"], out["organization"])
	}
	if sk, _ := out["skills"].([]string); len(sk) != 1 || sk[0] != "Sneak" {
		t.Errorf("skills = %v, want [Sneak]", out["skills"])
	}
	if out["role"] != "" || out["ev"] != "" {
		t.Errorf("role/ev must default to empty, got %q/%q", out["role"], out["ev"])
	}
	feats, ok := out["features"].([]map[string]any)
	if !ok || len(feats) != 2 {
		t.Fatalf("features = %#v, want 2 members", out["features"])
	}
	if feats[0]["name"] != "Pounce" || feats[1]["name"] != "Mighty Spring" {
		t.Errorf("member order = %v, %v; want Pounce, Mighty Spring", feats[0]["name"], feats[1]["name"])
	}
	if feats[0]["feature_type"] != "ability" || feats[0]["ability_type"] != "Signature Ability" {
		t.Errorf("pounce feature_type/ability_type = %v/%v", feats[0]["feature_type"], feats[0]["ability_type"])
	}
	if feats[1]["feature_type"] != "feature" {
		t.Errorf("mighty spring feature_type = %v, want feature", feats[1]["feature_type"])
	}
	meta, _ := feats[0]["metadata"].(map[string]any)
	if len(meta) != 1 {
		t.Errorf("member metadata must be scc-only, got %v", meta)
	}
	if scc, _ := meta["scc"].([]string); len(scc) != 1 || scc[0] != "mcdm.beastheart.v1/feature.ability.companion.beastheart.panther.level-1/pounce" {
		t.Errorf("member metadata.scc = %v", meta["scc"])
	}
}
