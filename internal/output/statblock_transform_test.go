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
