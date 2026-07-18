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

func TestTreasureParser_ItemPrerequisiteAndProjectSource(t *testing.T) {
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)

	section := &parser.Section{
		Heading:      "Ruby Ring of Recall",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource: "**Keywords:** Magic, Ring\n\n" +
			"**Item Prerequisite:** A ruby retrieved from an ancient sky elf ruin\n\n" +
			"**Project Source:** Texts or lore in Hyrallic\n\n" +
			"**Project Roll Characteristic:** Reason, Intuition, or Presence\n\n" +
			"**Project Goal:** 150\n\n" +
			"**Effect:** While wearing this ring, you can pull a willing creature.",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if got := result.Frontmatter["item_prerequisite"]; got != "A ruby retrieved from an ancient sky elf ruin" {
		t.Errorf("item_prerequisite = %v, want %q", got, "A ruby retrieved from an ancient sky elf ruin")
	}
	if got := result.Frontmatter["project_source"]; got != "Texts or lore in Hyrallic" {
		t.Errorf("project_source = %v, want %q", got, "Texts or lore in Hyrallic")
	}
	if got := result.Frontmatter["project_roll_characteristic"]; got != "Reason, Intuition, or Presence" {
		t.Errorf("project_roll_characteristic = %v, want %q", got, "Reason, Intuition, or Presence")
	}
}

func TestTreasureParser_ItemPrerequisiteAndProjectSource_SameLine(t *testing.T) {
	// Beastheart's "Precious Collar": Item Prerequisite, Project Source, and
	// Project Roll Characteristic all share one source line.
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)

	section := &parser.Section{
		Heading:      "Precious Collar",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource: "**Keywords:** Magic, Neck\n\n" +
			"**Item Prerequisite:** One collar worn by a royal pet **Project Source:** Texts or lore in Vaslorian **Project Roll Characteristic:** Reason or Intuition\n\n" +
			"**Project Goal:** 150",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if got := result.Frontmatter["item_prerequisite"]; got != "One collar worn by a royal pet" {
		t.Errorf("item_prerequisite = %v, want %q", got, "One collar worn by a royal pet")
	}
	if got := result.Frontmatter["project_source"]; got != "Texts or lore in Vaslorian" {
		t.Errorf("project_source = %v, want %q", got, "Texts or lore in Vaslorian")
	}
	if got := result.Frontmatter["project_roll_characteristic"]; got != "Reason or Intuition" {
		t.Errorf("project_roll_characteristic = %v, want %q", got, "Reason or Intuition")
	}
}

func TestTreasureParser_LevelEffects(t *testing.T) {
	// "Rampant Shield": leveled treasure with 1st/5th/9th Level effect bands.
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)

	section := &parser.Section{
		Heading:      "Rampant Shield",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource: "**Keywords:** Magic, Shield\n\n" +
			"**Item Prerequisite:** Strands from the manes of nine lions\n\n" +
			"**Project Source:** Texts or lore in Vaslorian **Project Roll Characteristic:** Might or Intuition\n\n" +
			"**Project Goal:** 450\n\n" +
			"**1st Level:** Only a beastheart can wield or carry this shield. You gain a +3 bonus to Stamina.\n\n" +
			"**5th Level:** The shield's bonus to Stamina increases to +6.\n\n" +
			"**9th Level:** The shield's bonus to Stamina increases to +9.",
	}

	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	le, ok := result.Frontmatter["level_effects"].(map[string]string)
	if !ok {
		t.Fatalf("level_effects = %v (%T), want map[string]string", result.Frontmatter["level_effects"], result.Frontmatter["level_effects"])
	}
	want := map[string]string{
		"1st": "Only a beastheart can wield or carry this shield. You gain a +3 bonus to Stamina.",
		"5th": "The shield's bonus to Stamina increases to +6.",
		"9th": "The shield's bonus to Stamina increases to +9.",
	}
	if !reflect.DeepEqual(le, want) {
		t.Errorf("level_effects = %v, want %v", le, want)
	}
}

func TestTreasureParser_NoLevelEffects_FieldAbsent(t *testing.T) {
	p := &TreasureParser{}
	ctx := context.NewContextStack(nil)
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
	if _, ok := result.Frontmatter["level_effects"]; ok {
		t.Errorf("level_effects should be absent, got %v", result.Frontmatter["level_effects"])
	}
	if _, ok := result.Frontmatter["level"]; ok {
		t.Errorf("level should be absent (no source data carries it), got %v", result.Frontmatter["level"])
	}
	if _, ok := result.Frontmatter["rarity"]; ok {
		t.Errorf("rarity should be absent (no source data carries it), got %v", result.Frontmatter["rarity"])
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
