package output

import (
	"encoding/json"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// Schema validation tests for new content types (Phase 3).
// These verify that TransformToSDKFormat passthrough output conforms to
// the schemas defined in steel-etl/schemas/.

// --- Schema property allowlists (from schemas/*.schema.json v2.0.0) ---

var schemaAllowedFields = map[string]map[string]bool{
	"class": {
		"name": true, "type": true, "heroic_resource": true,
		"flavor": true, "primary_characteristics": true,
		"weak_potency": true, "average_potency": true, "strong_potency": true,
		"starting_stamina": true, "stamina_per_level": true, "recoveries": true,
		"skills": true, "skill_group": true,
		"content": true, "metadata": true,
	},
	"kit": {
		"name": true, "type": true, "kit_type": true,
		"flavor": true, "armor": true, "weapon": true, "equipment_text": true,
		"stamina_bonus": true, "speed_bonus": true, "stability_bonus": true,
		"melee_damage_bonus": true, "ranged_damage_bonus": true,
		"melee_distance_bonus": true, "ranged_distance_bonus": true,
		"disengage_bonus": true, "signature_ability": true,
		"content": true, "metadata": true,
	},
	"perk": {
		"name": true, "type": true, "prerequisites": true, "perk_group": true,
		"content": true, "metadata": true,
	},
	"career": {
		"name": true, "type": true, "skills": true, "skill_group": true,
		"language": true, "renown": true, "wealth": true,
		"project_points": true, "perk": true, "flavor": true,
		"inciting_incidents": true,
		"content": true, "metadata": true,
	},
	"ancestry": {
		"name": true, "type": true,
		"flavor": true, "signature_trait_name": true, "signature_trait_description": true,
		"ancestry_points": true, "purchased_traits": true,
		"content": true, "metadata": true,
	},
	"culture": {
		"name": true, "type": true, "environment": true,
		"organization": true, "upbringing": true,
		"culture_benefit_type": true, "skill_options": true, "quick_build_skill": true,
		"language": true,
		"content": true, "metadata": true,
	},
	"title": {
		"name": true, "type": true, "echelon": true, "benefits": true,
		"flavor": true, "prerequisite": true, "effect": true,
		"content": true, "metadata": true,
	},
	"treasure": {
		"name": true, "type": true, "treasure_type": true,
		"level": true, "rarity": true,
		"flavor": true, "keywords": true, "item_prerequisite": true,
		"project_source": true, "project_roll_characteristic": true,
		"project_goal": true, "effect": true, "level_effects": true,
		"content": true, "metadata": true,
	},
	"condition": {
		"name": true, "type": true,
		"content": true, "metadata": true,
	},
	"complication": {
		"name": true, "type": true,
		"flavor": true, "benefit": true, "drawback": true,
		"content": true, "metadata": true,
	},
}

// --- Required field tests ---

func TestSchema_RequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		fm          map[string]any
		body        string
	}{
		{"class", "class", map[string]any{"name": "Fury", "type": "class"}, "Class body."},
		{"kit", "kit", map[string]any{"name": "Cloak and Dagger", "type": "kit"}, "Kit body."},
		{"perk", "perk", map[string]any{"name": "Coward", "type": "perk"}, "Perk body."},
		{"career", "career", map[string]any{"name": "Artisan", "type": "career"}, "Career body."},
		{"ancestry", "ancestry", map[string]any{"name": "Human", "type": "ancestry"}, "Ancestry body."},
		{"culture", "culture", map[string]any{"name": "Nomadic", "type": "culture"}, "Culture body."},
		{"title", "title", map[string]any{"name": "Demonslayer", "type": "title"}, "Title body."},
		{"treasure", "treasure", map[string]any{"name": "Bag of Holding", "type": "treasure"}, "Treasure body."},
		{"condition", "condition", map[string]any{"name": "Dazed", "type": "condition"}, "Condition body."},
		{"complication", "complication", map[string]any{"name": "Criminal Past", "type": "complication"}, "Complication body."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := &content.ParsedContent{Frontmatter: tt.fm, Body: tt.body}
			out := TransformToSDKFormat("", parsed)

			// Required: name
			if out["name"] == nil {
				t.Error("missing required field: name")
			}
			// Required: type must match the const
			if out["type"] != tt.contentType {
				t.Errorf("type = %v, want %q", out["type"], tt.contentType)
			}
			// Body should appear as content
			if out["content"] != tt.body {
				t.Errorf("content = %v, want %q", out["content"], tt.body)
			}
		})
	}
}

// --- No unevaluated properties tests (schemas use unevaluatedProperties: false per data-sdk-npm#13) ---

func TestSchema_NoUnevaluatedProperties(t *testing.T) {
	tests := []struct {
		name string
		fm   map[string]any
		body string
	}{
		{
			"class with heroic_resource",
			map[string]any{"name": "Fury", "type": "class", "heroic_resource": "Rage"},
			"Body text.",
		},
		{
			"kit with all fields",
			map[string]any{
				"name": "Cloak and Dagger", "type": "kit", "kit_type": "Martial",
				"stamina_bonus":      "+3",
				"speed_bonus":        "+1",
				"melee_damage_bonus": "+2/+2/+2",
				"equipment_text":     "Light armor and two light weapons",
			},
			"Kit description.",
		},
		{
			"perk with prerequisites and perk_group",
			map[string]any{"name": "Coward", "type": "perk", "prerequisites": "None", "perk_group": "Exploration"},
			"You know when to run.",
		},
		{
			"career with all fields",
			map[string]any{
				"name": "Artisan", "type": "career",
				"skills": []string{"Crafting"}, "language": "Common",
				"renown": "1", "wealth": "2",
				"project_points": "10", "perk": "Maker",
			},
			"Career description.",
		},
		{
			"ancestry with signature_trait_name",
			map[string]any{"name": "Hakaan", "type": "ancestry", "signature_trait_name": "Mighty"},
			"Ancestry body.",
		},
		{
			"culture with all fields",
			map[string]any{
				"name": "Nomadic", "type": "culture",
				"environment": "Wilderness", "organization": "Communal",
				"upbringing": "Martial", "skill_options": []string{"Nature"}, "language": "Caelian",
			},
			"Culture body.",
		},
		{
			"title with echelon, benefits, prerequisite, and effect",
			map[string]any{
				"name": "Demonslayer", "type": "title",
				"echelon": "1st", "benefits": []string{"Demon sense", "+1 damage vs fiends"},
				"prerequisite": "Must have slain a demon", "effect": "You can sense demons within 30 feet.",
			},
			"Title body.",
		},
		{
			"treasure with all fields",
			map[string]any{
				"name": "Bag of Holding", "type": "treasure",
				"treasure_type": "Leveled", "level": "3", "rarity": "Rare",
				"keywords": []string{"Magic", "Container"},
				"effect":   "This bag can hold 500 pounds.",
			},
			"Treasure body.",
		},
		{
			"condition minimal",
			map[string]any{"name": "Dazed", "type": "condition"},
			"You can't use maneuvers or triggered actions.",
		},
		{
			"complication with benefit and drawback",
			map[string]any{
				"name": "Criminal Past", "type": "complication",
				"benefit": "You know the underworld", "drawback": "The law watches you",
			},
			"You have a criminal past that may catch up with you.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := &content.ParsedContent{Frontmatter: tt.fm, Body: tt.body}
			out := TransformToSDKFormat("", parsed)

			contentType, _ := out["type"].(string)
			allowed, ok := schemaAllowedFields[contentType]
			if !ok {
				t.Fatalf("no schema allowlist for type %q", contentType)
			}

			for key := range out {
				if !allowed[key] {
					t.Errorf("unexpected field %q for type %q (not in schema)", key, contentType)
				}
			}
		})
	}
}

// --- Type const validation ---

func TestSchema_TypeConst(t *testing.T) {
	types := []string{"class", "kit", "perk", "career", "ancestry", "culture", "title", "treasure", "condition", "complication"}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			parsed := &content.ParsedContent{
				Frontmatter: map[string]any{"name": "Test", "type": typ},
				Body:        "Body.",
			}
			out := TransformToSDKFormat("", parsed)

			if out["type"] != typ {
				t.Errorf("type = %v, want %q", out["type"], typ)
			}
		})
	}
}

// --- Optional field type validation ---

func TestSchema_FieldTypes_Kit(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":               "Shining Armor",
			"type":               "kit",
			"kit_type":           "Martial",
			"stamina_bonus":      "+12",
			"stability_bonus":    "+1",
			"melee_damage_bonus": "+2/+2/+2",
			"equipment_text":     "Heavy armor, shield, and a weapon",
		},
		Body: "Kit body with stat table.",
	}
	out := TransformToSDKFormat("", parsed)

	// Individual bonus fields should be strings
	for _, field := range []string{"stamina_bonus", "stability_bonus", "melee_damage_bonus"} {
		if _, ok := out[field].(string); !ok {
			t.Errorf("%s should be string, got %T", field, out[field])
		}
	}

	// equipment_text should be string
	if _, ok := out["equipment_text"].(string); !ok {
		t.Errorf("equipment_text should be string, got %T", out["equipment_text"])
	}

	// kit_type should be string
	if _, ok := out["kit_type"].(string); !ok {
		t.Errorf("kit_type should be string, got %T", out["kit_type"])
	}
}

func TestSchema_FieldTypes_Title(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":     "Demonslayer",
			"type":     "title",
			"echelon":  "1st",
			"benefits": []string{"Demon sense", "Extra damage vs fiends"},
		},
		Body: "Title description.",
	}
	out := TransformToSDKFormat("", parsed)

	// echelon should be string
	if _, ok := out["echelon"].(string); !ok {
		t.Errorf("echelon should be string, got %T", out["echelon"])
	}

	// benefits should be array of strings
	if _, ok := out["benefits"].([]string); !ok {
		t.Errorf("benefits should be []string, got %T", out["benefits"])
	}
}

func TestSchema_FieldTypes_Treasure(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":          "Flame Tongue",
			"type":          "treasure",
			"treasure_type": "Leveled",
			"level":         "5",
			"rarity":        "Rare",
			"keywords":      []string{"Magic", "Weapon"},
			"effect":        "Deals extra fire damage",
		},
		Body: "A sword wreathed in flame.",
	}
	out := TransformToSDKFormat("", parsed)

	for _, field := range []string{"treasure_type", "level", "rarity", "effect"} {
		if _, ok := out[field].(string); !ok {
			t.Errorf("%s should be string, got %T", field, out[field])
		}
	}
	if _, ok := out["keywords"].([]string); !ok {
		t.Errorf("keywords should be []string, got %T", out["keywords"])
	}
}

func TestSchema_FieldTypes_Career(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":           "Criminal",
			"type":           "career",
			"skills":         []string{"Stealth"},
			"language":       "Thieves' Cant",
			"renown":         "1",
			"wealth":         "2",
			"project_points": "5",
			"perk":           "Streetwise",
		},
		Body: "Career description.",
	}
	out := TransformToSDKFormat("", parsed)

	// skills should be []string
	if _, ok := out["skills"].([]string); !ok {
		t.Errorf("skills should be []string, got %T", out["skills"])
	}
	for _, field := range []string{"language", "renown", "wealth", "project_points", "perk"} {
		if _, ok := out[field].(string); !ok {
			t.Errorf("%s should be string, got %T", field, out[field])
		}
	}
}

func TestSchema_FieldTypes_Culture(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":          "Nomadic",
			"type":          "culture",
			"environment":   "Wilderness",
			"organization":  "Communal",
			"upbringing":    "Martial",
			"skill_options": []string{"Nature"},
			"language":      "Caelian",
		},
		Body: "Culture description.",
	}
	out := TransformToSDKFormat("", parsed)

	for _, field := range []string{"environment", "organization", "upbringing", "language"} {
		if _, ok := out[field].(string); !ok {
			t.Errorf("%s should be string, got %T", field, out[field])
		}
	}
	if _, ok := out["skill_options"].([]string); !ok {
		t.Errorf("skill_options should be []string, got %T", out["skill_options"])
	}
}

// --- JSON roundtrip tests ---

func TestSchema_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		fm   map[string]any
		body string
	}{
		{"class", map[string]any{"name": "Fury", "type": "class", "heroic_resource": "Rage"}, "Body."},
		{"kit", map[string]any{"name": "Panther", "type": "kit", "kit_type": "Martial"}, "Body."},
		{"perk", map[string]any{"name": "Alert", "type": "perk"}, "Body."},
		{"career", map[string]any{"name": "Sage", "type": "career", "skills": []string{"Lore"}}, "Body."},
		{"ancestry", map[string]any{"name": "Elf", "type": "ancestry"}, "Body."},
		{"culture", map[string]any{"name": "Urban", "type": "culture"}, "Body."},
		{"title", map[string]any{"name": "Champion", "type": "title", "echelon": "1st"}, "Body."},
		{"treasure", map[string]any{"name": "Potion", "type": "treasure", "treasure_type": "Consumable"}, "Body."},
		{"condition", map[string]any{"name": "Prone", "type": "condition"}, "Body."},
		{"complication", map[string]any{"name": "Debt", "type": "complication"}, "Body."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := &content.ParsedContent{Frontmatter: tt.fm, Body: tt.body}
			out := TransformToSDKFormat("", parsed)

			data, err := json.Marshal(out)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var roundtrip map[string]any
			if err := json.Unmarshal(data, &roundtrip); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			// Verify name and type survive roundtrip
			if roundtrip["name"] != tt.fm["name"] {
				t.Errorf("name after roundtrip: got %v, want %v", roundtrip["name"], tt.fm["name"])
			}
			if roundtrip["type"] != tt.fm["type"] {
				t.Errorf("type after roundtrip: got %v, want %v", roundtrip["type"], tt.fm["type"])
			}
		})
	}
}

// --- Empty body tests ---

func TestSchema_EmptyBody(t *testing.T) {
	types := []string{"class", "kit", "perk", "career", "ancestry", "culture", "title", "treasure", "condition", "complication"}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			parsed := &content.ParsedContent{
				Frontmatter: map[string]any{"name": "Test", "type": typ},
			}
			out := TransformToSDKFormat("", parsed)

			if _, ok := out["content"]; ok {
				t.Error("content should not be present when body is empty")
			}
			if out["name"] != "Test" {
				t.Errorf("name = %v, want Test", out["name"])
			}
			if out["type"] != typ {
				t.Errorf("type = %v, want %q", out["type"], typ)
			}
		})
	}
}

// --- Passthrough does NOT apply feature schema ---

func TestSchema_PassthroughNoFeatureFields(t *testing.T) {
	// Verify that non-feature types don't get feature_type, effects, usage, etc.
	featureOnlyFields := []string{"feature_type", "effects", "usage", "ability_type", "trigger"}

	types := []string{"class", "kit", "perk", "career", "ancestry", "culture", "title", "treasure", "condition", "complication"}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			parsed := &content.ParsedContent{
				Frontmatter: map[string]any{"name": "Test", "type": typ},
				Body:        "Body text.",
			}
			out := TransformToSDKFormat("", parsed)

			for _, field := range featureOnlyFields {
				if _, ok := out[field]; ok {
					t.Errorf("passthrough type %q should not have feature field %q", typ, field)
				}
			}
		})
	}
}
