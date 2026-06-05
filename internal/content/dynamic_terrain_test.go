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
	if got.Frontmatter["ev"] != "2" || got.Frontmatter["stamina"] != "3" || got.Frontmatter["size"] != "1S" {
		t.Errorf("stats: got ev=%v stamina=%v size=%v",
			got.Frontmatter["ev"], got.Frontmatter["stamina"], got.Frontmatter["size"])
	}
	if got.Frontmatter["level"] != "2" {
		t.Errorf("level: got %v", got.Frontmatter["level"])
	}
}
