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
	if result.Frontmatter["skill"] != "Persuasion" {
		t.Errorf("skill = %v, want Persuasion", result.Frontmatter["skill"])
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

	if result.Frontmatter["skill"] != "Diplomacy" {
		t.Errorf("skill = %v, want Diplomacy (from Skills plural)", result.Frontmatter["skill"])
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

	// Annotation overrides should win
	if result.Frontmatter["skill"] != "Lore" {
		t.Errorf("skill = %v, want Lore (annotation override)", result.Frontmatter["skill"])
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

	if result.Frontmatter["skill"] != "Persuasion" {
		t.Errorf("skill = %v, want Persuasion (from Skills plural)", result.Frontmatter["skill"])
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

	bonuses, ok := result.Frontmatter["stat_bonuses"].(map[string]string)
	if !ok {
		t.Fatal("expected stat_bonuses")
	}
	if bonuses["stamina"] != "+3" {
		t.Errorf("stamina = %v, want +3", bonuses["stamina"])
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

	if _, ok := result.Frontmatter["stat_bonuses"]; ok {
		t.Error("expected no stat_bonuses when no table present")
	}
}

func TestKitParser_WithEquipment(t *testing.T) {
	section := &parser.Section{
		Heading:      "Heavy Kit",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "kit"},
		BodySource:   "A heavy kit.\n\n**Equipment:**\n- Heavy armor\n- Shield\n- Longsword",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&KitParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	equipment, ok := result.Frontmatter["equipment"].([]string)
	if !ok {
		t.Fatal("expected equipment to be []string")
	}
	if len(equipment) != 3 {
		t.Errorf("expected 3 equipment items, got %d: %v", len(equipment), equipment)
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
