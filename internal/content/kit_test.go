package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestKitParser(t *testing.T) {
	p := &KitParser{}

	if p.Type() != "kit" {
		t.Errorf("Type() = %q, want %q", p.Type(), "kit")
	}

	section := &parser.Section{
		Heading:      "Shining Armor",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "kit", "id": "shining-armor"},
		BodySource: `The shining armor kit is for heroes who stand at the front of battle.

| **Stamina** | **Speed** | **Melee Damage** | **Ranged Damage** |
| --- | --- | --- | --- |
| **+9** | **-1** | **+2/+2/+2** | **—** |

**Equipment:** Heavy armor, a melee weapon`,
	}

	ctx := context.NewContextStack(nil)
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Shining Armor" {
		t.Errorf("name = %v, want Shining Armor", result.Frontmatter["name"])
	}
	if result.Frontmatter["type"] != "kit" {
		t.Errorf("type = %v, want kit", result.Frontmatter["type"])
	}
	if result.ItemID != "shining-armor" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "shining-armor")
	}

	bonuses, ok := result.Frontmatter["stat_bonuses"].(map[string]string)
	if !ok {
		t.Fatal("expected stat_bonuses map")
	}
	if bonuses["stamina"] != "+9" {
		t.Errorf("stamina bonus = %q, want +9", bonuses["stamina"])
	}
	if bonuses["speed"] != "-1" {
		t.Errorf("speed bonus = %q, want -1", bonuses["speed"])
	}
	if bonuses["melee-damage"] != "+2/+2/+2" {
		t.Errorf("melee-damage bonus = %q, want +2/+2/+2", bonuses["melee-damage"])
	}
	// Ranged Damage is "—" (em dash) so should be excluded
	if _, exists := bonuses["ranged-damage"]; exists {
		t.Error("expected ranged-damage to be excluded (em dash value)")
	}
}

func TestAncestryParser(t *testing.T) {
	p := &AncestryParser{}

	if p.Type() != "ancestry" {
		t.Errorf("Type() = %q, want %q", p.Type(), "ancestry")
	}

	section := &parser.Section{
		Heading:      "Dwarf",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "ancestry"},
		BodySource: `Dwarves are stout folk known for their craftsmanship.

**Signature Trait:** Sturdy`,
	}

	ctx := context.NewContextStack(nil)
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Dwarf" {
		t.Errorf("name = %v, want Dwarf", result.Frontmatter["name"])
	}
	if result.Frontmatter["signature_trait"] != "Sturdy" {
		t.Errorf("signature_trait = %v, want Sturdy", result.Frontmatter["signature_trait"])
	}
	if result.ItemID != "dwarf" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "dwarf")
	}
	if len(result.TypePath) != 1 || result.TypePath[0] != "ancestry" {
		t.Errorf("TypePath = %v, want [ancestry]", result.TypePath)
	}
}

func TestTitleParser(t *testing.T) {
	p := &TitleParser{}

	section := &parser.Section{
		Heading:      "Mentor",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "title", "echelon": "1"},
		BodySource:   "You share your expertise with others.\n\n**Benefits:**\n- Gain a follower\n- +1 to Presence tests",
	}

	ctx := context.NewContextStack(nil)
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["echelon"] != "1" {
		t.Errorf("echelon = %v, want 1", result.Frontmatter["echelon"])
	}

	benefits, ok := result.Frontmatter["benefits"].([]string)
	if !ok {
		t.Fatal("expected benefits to be []string")
	}
	if len(benefits) != 2 {
		t.Errorf("expected 2 benefits, got %d", len(benefits))
	}
}

func TestTreasureParser(t *testing.T) {
	p := &TreasureParser{}

	section := &parser.Section{
		Heading:      "Healing Potion",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure", "treasure-type": "consumable"},
		BodySource:   "A vial of crimson liquid.\n\n**Level:** 1",
	}

	ctx := context.NewContextStack(nil)
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["treasure_type"] != "consumable" {
		t.Errorf("treasure_type = %v, want consumable", result.Frontmatter["treasure_type"])
	}
	if result.Frontmatter["level"] != "1" {
		t.Errorf("level = %v, want 1", result.Frontmatter["level"])
	}
}
