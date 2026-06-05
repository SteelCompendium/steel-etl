package content

import (
	"reflect"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestTreasureParser_NestedTypePath_Echelon(t *testing.T) {
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)
	// Parent treasure-group at H4 supplies echelon + category.
	ctx.Push(4, context.Metadata{"type": "treasure-group", "echelon": "1", "treasure-type": "consumable"})

	section := &parser.Section{
		Heading:      "Black Ash Dart",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource:   "**Keywords:** Magic\n\nAs a maneuver, you make a ranged free strike.",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	want := []string{"treasure", "1st-echelon", "consumable"}
	if !reflect.DeepEqual(result.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", result.TypePath, want)
	}
	if result.ItemID != "black-ash-dart" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "black-ash-dart")
	}
	if result.Frontmatter["echelon"] != "1" {
		t.Errorf("echelon = %v, want 1", result.Frontmatter["echelon"])
	}
	if result.Frontmatter["treasure_type"] != "consumable" {
		t.Errorf("treasure_type = %v, want consumable", result.Frontmatter["treasure_type"])
	}
}

func TestTreasureParser_NestedTypePath_Leveled(t *testing.T) {
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)
	// Leveled treasures have no echelon → tier "leveled".
	ctx.Push(4, context.Metadata{"type": "treasure-group", "treasure-type": "weapon"})

	section := &parser.Section{
		Heading:      "Displacer",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource:   "A leveled weapon treasure.",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	want := []string{"treasure", "leveled", "weapon"}
	if !reflect.DeepEqual(result.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", result.TypePath, want)
	}
	if _, ok := result.Frontmatter["echelon"]; ok {
		t.Errorf("echelon should be unset for leveled treasures, got %v", result.Frontmatter["echelon"])
	}
}

func TestTreasureParser_ItemAnnotationOverridesContext(t *testing.T) {
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)
	ctx.Push(3, context.Metadata{"type": "treasure-group", "echelon": "2", "treasure-type": "trinket"})

	// Beastheart-style: item carries its own @echelon.
	section := &parser.Section{
		Heading:      "Werewolf Tooth Pendant",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure", "echelon": "2"},
		BodySource:   "A trinket.",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []string{"treasure", "2nd-echelon", "trinket"}
	if !reflect.DeepEqual(result.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", result.TypePath, want)
	}
}

func TestTreasureParser_TierOverride_Artifact(t *testing.T) {
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)
	// Artifacts have no echelon/category; the group supplies an explicit tier.
	ctx.Push(3, context.Metadata{"type": "treasure-group", "tier": "artifact"})

	section := &parser.Section{
		Heading:      "Blade of a Thousand Years",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource:   "A powerful treasure that can unbalance the game.",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []string{"treasure", "artifact"}
	if !reflect.DeepEqual(result.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", result.TypePath, want)
	}
}

func TestTreasureGroupParser_NoOutput(t *testing.T) {
	p := &TreasureGroupParser{}
	if p.Type() != "treasure-group" {
		t.Errorf("Type() = %q, want treasure-group", p.Type())
	}
	section := &parser.Section{
		Heading:      "1st-Echelon Consumables",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure-group", "echelon": "1", "treasure-type": "consumable"},
		BodySource:   "These are the most numerous treasures.",
	}
	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.TypePath != nil {
		t.Errorf("TypePath = %v, want nil (container emits no file)", result.TypePath)
	}
	if result.ItemID != "" {
		t.Errorf("ItemID = %q, want empty", result.ItemID)
	}
	if result.Frontmatter["treasure_type"] != "consumable" {
		t.Errorf("treasure_type = %v, want consumable", result.Frontmatter["treasure_type"])
	}
}
