package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

func TestTransformAbility_BasicPowerRoll(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":                     "Brutal Slam",
			"type":                     "ability",
			"subtype":                  "signature",
			"action_type":              "Main action",
			"keywords":                 []string{"Melee", "Strike", "Weapon"},
			"distance":                 "Melee 1",
			"target":                   "One creature or object",
			"flavor":                   "The heavy impact of your weapon attacks drives your foes ever back.",
			"power_roll_characteristic": "Might",
			"tier1":                    "3 + M damage; push 1",
			"tier2":                    "6 + M damage; push 2",
			"tier3":                    "9 + M damage; push 4",
			"class":                    "fury",
			"level":                    "1",
		},
		Body:     "Full markdown body here.",
		TypePath: []string{"feature", "ability", "fury", "level-1"},
		ItemID:   "brutal-slam",
	}

	scc := "mcdm.heroes.v1/feature.ability.fury.level-1/brutal-slam"
	out := TransformToSDKFormat(scc, parsed)

	// Required schema fields
	assertEqual(t, out["type"], "feature")
	assertEqual(t, out["feature_type"], "ability")

	// Top-level fields
	assertEqual(t, out["name"], "Brutal Slam")
	assertEqual(t, out["usage"], "Main action")
	assertEqual(t, out["ability_type"], "Signature")
	assertEqual(t, out["distance"], "Melee 1")
	assertEqual(t, out["target"], "One creature or object")
	assertEqual(t, out["flavor"], "The heavy impact of your weapon attacks drives your foes ever back.")

	// Keywords
	kw, ok := out["keywords"].([]string)
	if !ok {
		t.Fatal("expected keywords to be []string")
	}
	if len(kw) != 3 || kw[0] != "Melee" {
		t.Errorf("unexpected keywords: %v", kw)
	}

	// Effects array
	effects, ok := out["effects"].([]map[string]any)
	if !ok {
		t.Fatal("expected effects to be []map[string]any")
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	assertEqual(t, effects[0]["roll"], "Power Roll + Might")
	assertEqual(t, effects[0]["tier1"], "3 + M damage; push 1")
	assertEqual(t, effects[0]["tier2"], "6 + M damage; push 2")
	assertEqual(t, effects[0]["tier3"], "9 + M damage; push 4")

	// Metadata
	meta, ok := out["metadata"].(map[string]any)
	if !ok {
		t.Fatal("expected metadata to be map[string]any")
	}
	assertEqual(t, meta["class"], "fury")
	assertEqual(t, meta["level"], 1) // should be int
	assertEqual(t, meta["item_id"], "brutal-slam")
	assertEqual(t, meta["ability_type"], "Signature")
	assertEqual(t, meta["content"], "Full markdown body here.")

	sccArr, ok := meta["scc"].([]string)
	if !ok || len(sccArr) != 1 || sccArr[0] != scc {
		t.Errorf("expected scc = [%q], got %v", scc, meta["scc"])
	}

	// Must NOT have steel-etl internal fields at top level
	if _, ok := out["power_roll_characteristic"]; ok {
		t.Error("power_roll_characteristic should not be at top level")
	}
	if _, ok := out["tier1"]; ok {
		t.Error("tier1 should not be at top level")
	}
	if _, ok := out["action_type"]; ok {
		t.Error("action_type should not be at top level (renamed to usage)")
	}
	if _, ok := out["class"]; ok {
		t.Error("class should not be at top level (moved to metadata)")
	}
	if _, ok := out["level"]; ok {
		t.Error("level should not be at top level (moved to metadata)")
	}
	if _, ok := out["subtype"]; ok {
		t.Error("subtype should not be at top level (renamed to ability_type)")
	}
}

func TestTransformAbility_WithEffectAndSpend(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":                     "Hit and Run",
			"type":                     "ability",
			"action_type":              "Main action",
			"power_roll_characteristic": "Might",
			"tier1":                    "2 + M damage",
			"tier2":                    "5 + M damage",
			"tier3":                    "7 + M damage",
			"effect":                   "You can shift 1 square.",
		},
		Body:     "body",
		TypePath: []string{"feature", "ability", "fury", "level-1"},
		ItemID:   "hit-and-run",
	}

	out := TransformToSDKFormat("", parsed)
	effects := out["effects"].([]map[string]any)

	if len(effects) != 2 {
		t.Fatalf("expected 2 effects (roll + effect), got %d", len(effects))
	}

	// First: power roll
	assertEqual(t, effects[0]["roll"], "Power Roll + Might")

	// Second: effect
	assertEqual(t, effects[1]["name"], "Effect")
	assertEqual(t, effects[1]["effect"], "You can shift 1 square.")
}

func TestTransformAbility_WithSpend(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":  "My Life for Yours",
			"type":  "ability",
			"effect": "You spend a Recovery and the target regains Stamina.",
			"spend": "1 Wrath: You can end one effect on the target.",
		},
		Body:   "body",
		ItemID: "my-life-for-yours",
	}

	out := TransformToSDKFormat("", parsed)
	effects := out["effects"].([]map[string]any)

	if len(effects) != 2 {
		t.Fatalf("expected 2 effects (effect + spend), got %d", len(effects))
	}

	// effect entry
	assertEqual(t, effects[0]["effect"], "You spend a Recovery and the target regains Stamina.")

	// spend entry
	assertEqual(t, effects[1]["cost"], "Spend 1 Wrath")
	assertEqual(t, effects[1]["effect"], "You can end one effect on the target.")
}

func TestTransformAbility_NoEffects_FallbackEmpty(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Bare Ability",
			"type": "ability",
		},
		Body: "some body",
	}

	out := TransformToSDKFormat("", parsed)
	effects := out["effects"].([]map[string]any)

	// Schema requires minItems: 1, so there should be a fallback
	if len(effects) < 1 {
		t.Fatal("expected at least 1 effect (fallback)")
	}
}

func TestTransformAbility_Trigger(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":        "Reactive Strike",
			"type":        "ability",
			"action_type": "Triggered",
			"trigger":     "The target starts their turn.",
			"effect":      "You deal 5 damage.",
		},
		Body: "body",
	}

	out := TransformToSDKFormat("", parsed)

	assertEqual(t, out["usage"], "Triggered")
	assertEqual(t, out["trigger"], "The target starts their turn.")
}

func TestTransformTrait_Basic(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":  "Growing Ferocity",
			"type":  "trait",
			"class": "fury",
			"level": "1",
		},
		Body:     "You gain certain benefits in combat based on ferocity.\n\n###### Berserker Table\n| col | col |",
		TypePath: []string{"feature", "trait", "fury", "level-1"},
		ItemID:   "growing-ferocity",
	}

	scc := "mcdm.heroes.v1/feature.trait.fury.level-1/growing-ferocity"
	out := TransformToSDKFormat(scc, parsed)

	assertEqual(t, out["type"], "feature")
	assertEqual(t, out["feature_type"], "trait")
	assertEqual(t, out["name"], "Growing Ferocity")

	effects := out["effects"].([]map[string]any)
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	if !strings.Contains(effects[0]["effect"].(string), "Berserker Table") {
		t.Error("trait effect should contain the full body including sub-headings")
	}

	meta := out["metadata"].(map[string]any)
	assertEqual(t, meta["class"], "fury")
	assertEqual(t, meta["level"], 1)
	assertEqual(t, meta["action_type"], "feature")

	// Traits should NOT have these at top level
	if _, ok := out["class"]; ok {
		t.Error("class should be in metadata, not top level")
	}
	if _, ok := out["level"]; ok {
		t.Error("level should be in metadata, not top level")
	}
}

func TestTransformTrait_EmptyBody(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Empty Trait",
			"type": "trait",
		},
		Body: "",
	}

	out := TransformToSDKFormat("", parsed)
	effects := out["effects"].([]map[string]any)

	// Schema requires minItems: 1
	if len(effects) < 1 {
		t.Fatal("expected at least 1 effect for empty trait")
	}
}

func TestTransformPassthrough_Class(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":            "Fury",
			"type":            "class",
			"heroic_resource": "Ferocity",
		},
		Body: "The fury is a primal warrior.",
	}

	out := TransformToSDKFormat("", parsed)

	assertEqual(t, out["type"], "class")
	assertEqual(t, out["name"], "Fury")
	assertEqual(t, out["heroic_resource"], "Ferocity")
	assertEqual(t, out["content"], "The fury is a primal warrior.")

	// Should NOT have feature schema fields
	if _, ok := out["feature_type"]; ok {
		t.Error("passthrough should not have feature_type")
	}
	if _, ok := out["effects"]; ok {
		t.Error("passthrough should not have effects")
	}
}

func TestTransformPassthrough_Kit(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Panther",
			"type": "kit",
			"stat_bonuses": map[string]string{
				"stamina": "+3",
				"speed":   "+2",
			},
		},
		Body: "A swift melee kit.",
	}

	out := TransformToSDKFormat("", parsed)

	assertEqual(t, out["type"], "kit")
	assertEqual(t, out["content"], "A swift melee kit.")
}

func TestTransformAbility_JSONSchemaCompliant(t *testing.T) {
	// Verify the output can be serialized and contains required schema fields
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":                     "Brutal Slam",
			"type":                     "ability",
			"action_type":              "Main action",
			"power_roll_characteristic": "Might",
			"tier1":                    "3 + M damage",
			"tier2":                    "6 + M damage",
			"tier3":                    "9 + M damage",
		},
		Body:   "body",
		ItemID: "brutal-slam",
	}

	out := TransformToSDKFormat("", parsed)

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Parse it back and check required fields
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result["type"] != "feature" {
		t.Errorf("type = %v, want feature", result["type"])
	}
	if result["feature_type"] != "ability" {
		t.Errorf("feature_type = %v, want ability", result["feature_type"])
	}
	if result["effects"] == nil {
		t.Error("effects should not be nil")
	}

	effectsArr, ok := result["effects"].([]any)
	if !ok || len(effectsArr) < 1 {
		t.Error("effects should have at least 1 item")
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"signature", "Signature"},
		{"heroic", "Heroic"},
		{"", ""},
		{"A", "A"},
		{"already Capitalized", "Already Capitalized"},
	}
	for _, tt := range tests {
		got := capitalizeFirst(tt.in)
		if got != tt.want {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseSpendField(t *testing.T) {
	tests := []struct {
		input    string
		wantCost string
		wantEff  string
	}{
		{
			"1 Wrath: You can end one effect.",
			"Spend 1 Wrath",
			"You can end one effect.",
		},
		{
			"5 Essence: The target gains 10 temporary Stamina.",
			"Spend 5 Essence",
			"The target gains 10 temporary Stamina.",
		},
		{
			"no colon here",
			"",
			"no colon here",
		},
	}

	for _, tt := range tests {
		result := parseSpendField(tt.input)
		if tt.wantCost != "" {
			if result["cost"] != tt.wantCost {
				t.Errorf("parseSpendField(%q).cost = %q, want %q", tt.input, result["cost"], tt.wantCost)
			}
		}
		if result["effect"] != tt.wantEff {
			t.Errorf("parseSpendField(%q).effect = %q, want %q", tt.input, result["effect"], tt.wantEff)
		}
	}
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}
