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

##### Equipment

Heavy armor, a melee weapon

##### Kit Bonuses

**Stamina Bonus:** +9

**Speed Bonus:** -1

**Melee Damage Bonus:** +2/+2/+2`,
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

	// Individual bonus fields extracted from **Field Bonus:** value lines
	if result.Frontmatter["stamina_bonus"] != "+9" {
		t.Errorf("stamina_bonus = %q, want +9", result.Frontmatter["stamina_bonus"])
	}
	if result.Frontmatter["speed_bonus"] != "-1" {
		t.Errorf("speed_bonus = %q, want -1", result.Frontmatter["speed_bonus"])
	}
	if result.Frontmatter["melee_damage_bonus"] != "+2/+2/+2" {
		t.Errorf("melee_damage_bonus = %q, want +2/+2/+2", result.Frontmatter["melee_damage_bonus"])
	}
	// Ranged Damage is not present, so should be excluded
	if _, exists := result.Frontmatter["ranged_damage_bonus"]; exists {
		t.Error("expected ranged_damage_bonus to be excluded (not in body)")
	}

	// Equipment text extracted from paragraph after ##### Equipment heading
	if result.Frontmatter["equipment_text"] != "Heavy armor, a melee weapon" {
		t.Errorf("equipment_text = %v, want 'Heavy armor, a melee weapon'", result.Frontmatter["equipment_text"])
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
	if result.Frontmatter["signature_trait_name"] != "Sturdy" {
		t.Errorf("signature_trait_name = %v, want Sturdy", result.Frontmatter["signature_trait_name"])
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
