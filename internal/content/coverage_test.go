package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// CultureParser: annotation overrides and plural field names (Skills, Languages)
func TestCultureParser_AnnotationOverrides(t *testing.T) {
	section := &parser.Section{
		Heading:      "Urban",
		HeadingLevel: 3,
		Annotation: map[string]string{
			"type":         "culture",
			"environment":  "City",
			"organization": "Bureaucratic",
			"upbringing":   "Academic",
			"skill":        "Persuasion",
			"language":     "Common",
		},
		BodySource: `An urban culture of sprawling cities.

**Environment:** Rural
**Organization:** Feudal`,
	}

	ctx := context.NewContextStack(nil)
	result, err := (&CultureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Annotation overrides should take precedence over body extraction
	if result.Frontmatter["environment"] != "City" {
		t.Errorf("environment = %v, want City (annotation override)", result.Frontmatter["environment"])
	}
	// skill annotation → skill_options array
	skillOpts, ok := result.Frontmatter["skill_options"].([]string)
	if !ok {
		t.Fatal("expected skill_options to be []string")
	}
	if len(skillOpts) != 1 || skillOpts[0] != "Persuasion" {
		t.Errorf("skill_options = %v, want [Persuasion]", skillOpts)
	}
	if result.Frontmatter["language"] != "Common" {
		t.Errorf("language = %v, want Common", result.Frontmatter["language"])
	}
}

func TestCultureParser_PluralFieldNames(t *testing.T) {
	section := &parser.Section{
		Heading:      "Cosmopolitan",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "culture"},
		BodySource: `A cosmopolitan culture.

**Skills:** Diplomacy
**Languages:** Common, Elvish`,
	}

	ctx := context.NewContextStack(nil)
	result, err := (&CultureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Skills → skill_options array
	skillOpts, ok := result.Frontmatter["skill_options"].([]string)
	if !ok {
		t.Fatal("expected skill_options to be []string")
	}
	if len(skillOpts) != 1 || skillOpts[0] != "Diplomacy" {
		t.Errorf("skill_options = %v, want [Diplomacy] (from Skills plural)", skillOpts)
	}
	if result.Frontmatter["language"] != "Common, Elvish" {
		t.Errorf("language = %v, want 'Common, Elvish' (from Languages plural)", result.Frontmatter["language"])
	}
}

func TestCultureParser_NoAnnotation(t *testing.T) {
	section := &parser.Section{
		Heading:      "Tribal",
		HeadingLevel: 3,
		BodySource:   "A tribal culture.\n\n**Environment:** Wilderness",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&CultureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.ItemID != "tribal" {
		t.Errorf("ItemID = %q, want tribal (slugified from heading)", result.ItemID)
	}
	if result.Frontmatter["environment"] != "Wilderness" {
		t.Errorf("environment = %v, want Wilderness", result.Frontmatter["environment"])
	}
}

// TreasureParser: treasure_type from annotation (underscore variant), context lookup, rarity
func TestTreasureParser_UnderscoreAnnotation(t *testing.T) {
	section := &parser.Section{
		Heading:      "Flame Sword",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure", "treasure_type": "artifact"},
		BodySource:   "A sword wreathed in flame.\n\n**Level:** 5\n**Rarity:** Rare",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&TreasureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["treasure_type"] != "artifact" {
		t.Errorf("treasure_type = %v, want artifact (from treasure_type annotation)", result.Frontmatter["treasure_type"])
	}
	if result.Frontmatter["rarity"] != "Rare" {
		t.Errorf("rarity = %v, want Rare", result.Frontmatter["rarity"])
	}
}

func TestTreasureParser_ContextLookup(t *testing.T) {
	section := &parser.Section{
		Heading:      "Minor Trinket",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource:   "A small trinket.",
	}

	// Set up context with treasure-type from parent
	ctx := context.NewContextStack(nil)
	ctx.Push(3, context.Metadata{"treasure-type": "trinket"})

	result, err := (&TreasureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["treasure_type"] != "trinket" {
		t.Errorf("treasure_type = %v, want trinket (from context)", result.Frontmatter["treasure_type"])
	}
}

func TestTreasureParser_NoID(t *testing.T) {
	section := &parser.Section{
		Heading:      "Crystal Ball",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure"},
		BodySource:   "A fortune-telling device.",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&TreasureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.ItemID != "crystal-ball" {
		t.Errorf("ItemID = %q, want crystal-ball (slugified)", result.ItemID)
	}
}

func TestTreasureParser_Keywords(t *testing.T) {
	section := &parser.Section{
		Heading:      "Ring of Protection",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "treasure", "treasure-type": "Leveled"},
		BodySource:   "A magical ring.\n\n**Keywords:** Magic, Ring\n**Effect:** +1 to defense",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&TreasureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	kw, ok := result.Frontmatter["keywords"].([]string)
	if !ok {
		t.Fatal("expected keywords to be []string")
	}
	if len(kw) != 2 || kw[0] != "Magic" || kw[1] != "Ring" {
		t.Errorf("keywords = %v, want [Magic, Ring]", kw)
	}
	if result.Frontmatter["effect"] != "+1 to defense" {
		t.Errorf("effect = %v, want '+1 to defense'", result.Frontmatter["effect"])
	}
}

// TitleParser: echelon from body, echelon from context, benefits with "Benefit" singular
func TestTitleParser_EchelonFromBody(t *testing.T) {
	section := &parser.Section{
		Heading:      "Champion",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "title"},
		BodySource:   "A great champion.\n\n**Echelon:** 2\n\n**Benefit:**\n- +1 to attacks\n- Aura of courage",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&TitleParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["echelon"] != "2" {
		t.Errorf("echelon = %v, want 2 (from body)", result.Frontmatter["echelon"])
	}

	benefits, ok := result.Frontmatter["benefits"].([]string)
	if !ok {
		t.Fatal("expected benefits to be []string")
	}
	if len(benefits) != 2 {
		t.Errorf("expected 2 benefits, got %d: %v", len(benefits), benefits)
	}
}

func TestTitleParser_EchelonFromContext(t *testing.T) {
	section := &parser.Section{
		Heading:      "Guardian",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "title"},
		BodySource:   "A steadfast guardian.",
	}

	ctx := context.NewContextStack(nil)
	ctx.Push(3, context.Metadata{"echelon": "3"})

	result, err := (&TitleParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["echelon"] != "3" {
		t.Errorf("echelon = %v, want 3 (from context)", result.Frontmatter["echelon"])
	}
}

func TestTitleParser_NoEchelon(t *testing.T) {
	section := &parser.Section{
		Heading:      "Wanderer",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "title"},
		BodySource:   "A wandering soul.",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&TitleParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if _, ok := result.Frontmatter["echelon"]; ok {
		t.Error("expected no echelon when not set anywhere")
	}
}

func TestTitleParser_PrerequisiteAndEffect(t *testing.T) {
	section := &parser.Section{
		Heading:      "Archmage",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "title", "echelon": "4"},
		BodySource:   "The pinnacle of magical mastery.\n\n**Prerequisite:** Must be a caster class\n**Effect:** You gain access to 10th-level spells.",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&TitleParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["prerequisite"] != "Must be a caster class" {
		t.Errorf("prerequisite = %v, want 'Must be a caster class'", result.Frontmatter["prerequisite"])
	}
	if result.Frontmatter["effect"] != "You gain access to 10th-level spells." {
		t.Errorf("effect = %v, want 'You gain access to 10th-level spells.'", result.Frontmatter["effect"])
	}
}

// CareerParser: annotation overrides, plural "Skills"/"Languages"
func TestCareerParser_AnnotationOverrides(t *testing.T) {
	section := &parser.Section{
		Heading:      "Scholar",
		HeadingLevel: 3,
		Annotation: map[string]string{
			"type":     "career",
			"skill":    "Lore",
			"language": "Ancient",
			"renown":   "2",
			"wealth":   "3",
			"perk":     "Bookworm",
		},
		BodySource: "A learned scholar.\n\n**Skill:** History\n**Language:** Common",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&CareerParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Annotation overrides should win - skill → skills array
	skills, ok := result.Frontmatter["skills"].([]string)
	if !ok {
		t.Fatal("expected skills to be []string")
	}
	if len(skills) != 1 || skills[0] != "Lore" {
		t.Errorf("skills = %v, want [Lore] (annotation override)", skills)
	}
	if result.Frontmatter["language"] != "Ancient" {
		t.Errorf("language = %v, want Ancient (annotation override)", result.Frontmatter["language"])
	}
	if result.Frontmatter["perk"] != "Bookworm" {
		t.Errorf("perk = %v, want Bookworm (annotation override)", result.Frontmatter["perk"])
	}
}

func TestCareerParser_PluralFields(t *testing.T) {
	section := &parser.Section{
		Heading:      "Diplomat",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "career"},
		BodySource:   "A skilled negotiator.\n\n**Skills:** Persuasion\n**Languages:** Common, Elvish",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&CareerParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Skills → skills array
	skills, ok := result.Frontmatter["skills"].([]string)
	if !ok {
		t.Fatal("expected skills to be []string")
	}
	if len(skills) != 1 || skills[0] != "Persuasion" {
		t.Errorf("skills = %v, want [Persuasion] (from Skills plural)", skills)
	}
	if result.Frontmatter["language"] != "Common, Elvish" {
		t.Errorf("language = %v, want 'Common, Elvish' (from Languages plural)", result.Frontmatter["language"])
	}
}

// ClassParser: with annotation ID and without heroic resource
func TestClassParser_WithAnnotationID(t *testing.T) {
	section := &parser.Section{
		Heading:      "Tactician",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "tactician"},
		BodySource:   "The tactician directs allies with strategic precision.",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&ClassParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.ItemID != "tactician" {
		t.Errorf("ItemID = %q, want tactician (from annotation id)", result.ItemID)
	}
	if _, ok := result.Frontmatter["heroic_resource"]; ok {
		t.Error("expected no heroic_resource when not in body")
	}
}

func TestClassParser_NoAnnotation(t *testing.T) {
	section := &parser.Section{
		Heading:      "Shadow Warrior",
		HeadingLevel: 2,
		BodySource:   "The shadow warrior is a deadly assassin.\n\n**Heroic Resource: Shadow Points**",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&ClassParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.ItemID != "shadow-warrior" {
		t.Errorf("ItemID = %q, want shadow-warrior (slugified)", result.ItemID)
	}
	if result.Frontmatter["heroic_resource"] != "Shadow Points" {
		t.Errorf("heroic_resource = %v, want Shadow Points", result.Frontmatter["heroic_resource"])
	}
}

func TestClassParser_PrimaryCharacteristics(t *testing.T) {
	section := &parser.Section{
		Heading:      "Fury",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "fury"},
		BodySource:   "The fury charges into battle.\n\n**Primary Characteristics:** Might, Agility\n**Heroic Resource: Rage**",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&ClassParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	chars, ok := result.Frontmatter["primary_characteristics"].([]string)
	if !ok {
		t.Fatal("expected primary_characteristics to be []string")
	}
	if len(chars) != 2 || chars[0] != "Might" || chars[1] != "Agility" {
		t.Errorf("primary_characteristics = %v, want [Might, Agility]", chars)
	}
}

// KitParser: with annotation kit-type
func TestKitParser_WithKitType(t *testing.T) {
	section := &parser.Section{
		Heading:      "Ranger Kit",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "kit", "kit-type": "martial"},
		BodySource:   "A kit for rangers.\n\n| **Stamina** | **Speed** |\n| --- | --- |\n| **+3** | **+1** |",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&KitParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["kit_type"] != "martial" {
		t.Errorf("kit_type = %v, want martial", result.Frontmatter["kit_type"])
	}

	if result.Frontmatter["stamina_bonus"] != "+3" {
		t.Errorf("stamina_bonus = %v, want +3", result.Frontmatter["stamina_bonus"])
	}
}

func TestKitParser_NoTable(t *testing.T) {
	section := &parser.Section{
		Heading:      "Simple Kit",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "kit"},
		BodySource:   "A simple kit with no stat table.",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&KitParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// No bonus fields should be present
	for _, field := range []string{"stamina_bonus", "speed_bonus", "melee_damage_bonus"} {
		if _, ok := result.Frontmatter[field]; ok {
			t.Errorf("expected no %s when no table present", field)
		}
	}
}

func TestKitParser_WithEquipmentText(t *testing.T) {
	section := &parser.Section{
		Heading:      "Heavy Kit",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "kit"},
		BodySource:   "A heavy kit.\n\n**Equipment:** Heavy armor, shield, and a longsword",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&KitParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["equipment_text"] != "Heavy armor, shield, and a longsword" {
		t.Errorf("equipment_text = %v, want 'Heavy armor, shield, and a longsword'", result.Frontmatter["equipment_text"])
	}
}

// ComplicationParser: benefit and drawback extraction
func TestComplicationParser_BenefitDrawback(t *testing.T) {
	section := &parser.Section{
		Heading:      "Criminal Past",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "complication"},
		BodySource:   "You have a criminal record.\n\n**Benefit:** You know the underworld\n**Drawback:** The law is watching you",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&ComplicationParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["benefit"] != "You know the underworld" {
		t.Errorf("benefit = %v, want 'You know the underworld'", result.Frontmatter["benefit"])
	}
	if result.Frontmatter["drawback"] != "The law is watching you" {
		t.Errorf("drawback = %v, want 'The law is watching you'", result.Frontmatter["drawback"])
	}
}

// PerkParser: perk_group extraction
func TestPerkParser_PerkGroup(t *testing.T) {
	section := &parser.Section{
		Heading:      "Alert",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "perk", "perk-group": "Exploration"},
		BodySource:   "You are always on guard.",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&PerkParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["perk_group"] != "Exploration" {
		t.Errorf("perk_group = %v, want Exploration", result.Frontmatter["perk_group"])
	}
}

func TestPerkParser_PerkGroupFromContext(t *testing.T) {
	section := &parser.Section{
		Heading:      "Craft Item",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "perk"},
		BodySource:   "You can craft items during downtime.",
	}

	ctx := context.NewContextStack(nil)
	ctx.Push(3, context.Metadata{"perk-group": "Crafting"})

	result, err := (&PerkParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["perk_group"] != "Crafting" {
		t.Errorf("perk_group = %v, want Crafting (from context)", result.Frontmatter["perk_group"])
	}
}

// extractListField edge cases
func TestExtractListField_InlineValue(t *testing.T) {
	body := "Some text.\n\n**Equipment:** Heavy armor, a shield"
	result := extractListField(body, "Equipment")
	if len(result) != 1 || result[0] != "Heavy armor, a shield" {
		t.Errorf("expected inline value, got %v", result)
	}
}

func TestExtractListField_ListItems(t *testing.T) {
	body := "Some text.\n\n**Equipment:**\n- Heavy armor\n- Shield\n- Longsword\n\nMore text."
	result := extractListField(body, "Equipment")
	if len(result) != 3 {
		t.Errorf("expected 3 list items, got %d: %v", len(result), result)
	}
}

func TestExtractListField_NoMatch(t *testing.T) {
	body := "Some text without the target field."
	result := extractListField(body, "Equipment")
	if len(result) != 0 {
		t.Errorf("expected 0 items, got %d: %v", len(result), result)
	}
}

func TestExtractListField_ListWithBlankLines(t *testing.T) {
	body := "**Benefits:**\n\n- First benefit\n- Second benefit"
	result := extractListField(body, "Benefits")
	if len(result) != 2 {
		t.Errorf("expected 2 items (blank lines between header and items), got %d: %v", len(result), result)
	}
}
