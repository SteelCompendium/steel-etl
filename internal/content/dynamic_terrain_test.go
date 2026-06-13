package content

import (
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
)

func TestDynamicTerrainParser(t *testing.T) {
	body := "" +
		"This beehive is full of angry bees.\n\n" +
		"- **EV:** 2\n- **Stamina:** 3\n- **Size:** 1S\n\n" +
		"> 🌀 **Deactivate**\n>\n> The beehive can't be deactivated.\n"
	sec := newSection("Angry Beehive (Level 2 Hazard Hexer)", 9,
		map[string]string{"type": "dynamic-terrain"}, body)

	ctx := context.NewContextStack(nil)
	ctx.Push(3, map[string]string{"domain": "dynamic-terrain", "category": "environmental-hazards"})

	p := &DynamicTerrainParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ItemID != "angry-beehive" {
		t.Errorf("ItemID: got %q", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "dynamic-terrain/environmental-hazards" {
		t.Errorf("TypePath: got %v", got.TypePath)
	}
	if got.Frontmatter["level"] != 2 {
		t.Errorf("level: got %v (want int 2)", got.Frontmatter["level"])
	}
	stats, ok := got.Frontmatter["stats"].([]map[string]any)
	if !ok || len(stats) != 3 {
		t.Fatalf("stats: got %+v, want 3 pairs", got.Frontmatter["stats"])
	}
	if stats[0]["name"] != "EV" || stats[0]["value"] != "2" {
		t.Errorf("stats[0]: got %+v", stats[0])
	}
}

func TestDynamicTerrainParser_ClassifierStatsFeatures(t *testing.T) {
	body := "This beehive is full of angry bees who swarm and attack with little provocation.\n\n" +
		"- **EV:** 2\n" +
		"- **Stamina:** 3\n" +
		"- **Size:** 1S\n\n" +
		"> 🌀 **Deactivate**\n>\n> The beehive can't be deactivated.\n" +
		"\n" +
		"> ❕ **Activate**\n>\n> A creature enters the hive's space.\n>\n> **Effect:** The hive is removed from the encounter map.\n"

	sec := newSection("Angry Beehive (Level 2 Hazard Hexer)", 9,
		map[string]string{"type": "dynamic-terrain"}, body)
	ctx := context.NewContextStack(nil)
	ctx.Push(3, map[string]string{"domain": "dynamic-terrain", "category": "environmental-hazards"})

	p := &DynamicTerrainParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatal(err)
	}
	fm := got.Frontmatter
	if fm["level"] != 2 {
		t.Errorf("level = %v, want int 2", fm["level"])
	}
	if fm["terrain_type"] != "Hazard" {
		t.Errorf("terrain_type = %v, want Hazard", fm["terrain_type"])
	}
	if fm["role"] != "Hexer" {
		t.Errorf("role = %v, want Hexer", fm["role"])
	}
	flavor, _ := fm["flavor"].(string)
	if !strings.HasPrefix(flavor, "This beehive is full of angry bees") {
		t.Errorf("flavor = %q", flavor)
	}
	stats, ok := fm["stats"].([]map[string]any)
	if !ok || len(stats) != 3 {
		t.Fatalf("stats = %+v, want 3 ordered pairs", fm["stats"])
	}
	if stats[2]["name"] != "Size" || stats[2]["value"] != "1S" {
		t.Errorf("stats[2] = %+v", stats[2])
	}
	for _, gone := range []string{"ev", "stamina", "size"} {
		if _, ok := fm[gone]; ok {
			t.Errorf("scalar %q should be replaced by stats[]", gone)
		}
	}
	feats, ok := fm["features"].([]map[string]any)
	if !ok || len(feats) != 2 {
		t.Fatalf("features = %+v, want 2", fm["features"])
	}
	if feats[0]["name"] != "Deactivate" || feats[0]["icon"] != "🌀" {
		t.Errorf("features[0] = %+v", feats[0])
	}
	if secs, ok := feats[1]["sections"].([]map[string]any); !ok || len(secs) != 1 || secs[0]["label"] != "Effect" {
		t.Errorf("Activate sections = %+v", feats[1]["sections"])
	}
}
