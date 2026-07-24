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
			"name":                      "Brutal Slam",
			"type":                      "ability",
			"subtype":                   "signature",
			"action_type":               "Main action",
			"keywords":                  []string{"Melee", "Strike", "Weapon"},
			"distance":                  "Melee 1",
			"target":                    "One creature or object",
			"flavor":                    "The heavy impact of your weapon attacks drives your foes ever back.",
			"power_roll_characteristic": "Might",
			"tier1":                     "3 + M damage; push 1",
			"tier2":                     "6 + M damage; push 2",
			"tier3":                     "9 + M damage; push 4",
			"class":                     "fury",
			"level":                     "1",
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
			"name":                      "Hit and Run",
			"type":                      "ability",
			"action_type":               "Main action",
			"power_roll_characteristic": "Might",
			"tier1":                     "2 + M damage",
			"tier2":                     "5 + M damage",
			"tier3":                     "7 + M damage",
			"effect":                    "You can shift 1 square.",
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
			"name":   "My Life for Yours",
			"type":   "ability",
			"effect": "You spend a Recovery and the target regains Stamina.",
			"spend":  "1 Wrath: You can end one effect on the target.",
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

// The parser emits a complete, document-ordered fm["effects"] list (roll entry
// included at its position). The transform must pass it through verbatim.
func TestTransformAbility_EffectsListVerbatim(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Minor Telekinesis",
			"type": "ability",
			// Top-level power-roll fields still exist (cards read them); the roll
			// is ALSO in the list at its position and must NOT be re-added.
			"power_roll_characteristic": "Reason",
			"tier1":                     "a",
			"effects": []map[string]any{
				{"roll": "Power Roll + Reason", "tier1": "a"},
				{"name": "Effect", "effect": "You slide the target."},
				{"cost": "Spend 2+ Clarity", "effect": "Size increases."},
				{"cost": "Spend 3 Clarity", "effect": "You can vertical slide the target."},
			},
		},
		Body:   "body",
		ItemID: "minor-telekinesis",
	}

	out := TransformToSDKFormat("", parsed)
	effects := out["effects"].([]map[string]any)

	if len(effects) != 4 {
		t.Fatalf("expected 4 effects verbatim (roll not duplicated), got %d: %v", len(effects), effects)
	}
	assertEqual(t, effects[0]["roll"], "Power Roll + Reason")
	assertEqual(t, effects[1]["name"], "Effect")
	assertEqual(t, effects[2]["cost"], "Spend 2+ Clarity")
	assertEqual(t, effects[3]["cost"], "Spend 3 Clarity")
}

// When an Effect precedes the power roll in the source, the list carries that
// order and the transform must preserve it (roll second, not forced first).
func TestTransformAbility_EffectBeforeRollPreserved(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":                      "Instantaneous Excavation",
			"type":                      "ability",
			"power_roll_characteristic": "Reason",
			"tier1":                     "x",
			"effects": []map[string]any{
				{"name": "Effect", "effect": "You open two holes."},
				{"roll": "Power Roll + Reason", "tier1": "x"},
			},
		},
		Body:   "body",
		ItemID: "instantaneous-excavation",
	}

	out := TransformToSDKFormat("", parsed)
	effects := out["effects"].([]map[string]any)
	if len(effects) != 2 {
		t.Fatalf("expected 2 effects (Effect then roll, roll not duplicated), got %d: %v", len(effects), effects)
	}
	assertEqual(t, effects[0]["name"], "Effect")
	assertEqual(t, effects[1]["roll"], "Power Roll + Reason")
}

func TestTransformFeature_PlainFeature(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "A Beyonding of Vision",
			"type": "feature",
		},
		Body:     "Your void sense reaches further.",
		TypePath: []string{"feature", "elementalist", "level-1"},
		ItemID:   "a-beyonding-of-vision",
	}
	out := TransformToSDKFormat("mcdm.heroes.v1/feature.elementalist.level-1/a-beyonding-of-vision", parsed)
	assertEqual(t, out["type"], "feature")
	assertEqual(t, out["feature_type"], "feature")
	if _, ok := out["effects"]; !ok {
		t.Error("expected effects[] on a plain feature")
	}
	meta := out["metadata"].(map[string]any)
	assertEqual(t, meta["feature_type"], "feature")
	assertEqual(t, meta["action_type"], "feature")
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

func TestTransformAbility_SubclassInMetadata(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":     "Sic 'Em!",
			"type":     "ability",
			"class":    "beastheart",
			"level":    "6",
			"subclass": "guardian",
		},
		Body:     "body",
		TypePath: []string{"feature", "ability", "beastheart", "level-6"},
		ItemID:   "sic-em",
	}
	out := TransformToSDKFormat("mcdm.beastheart.v1/feature.ability.beastheart.level-6/sic-em", parsed)
	meta := out["metadata"].(map[string]any)
	assertEqual(t, meta["subclass"], "guardian")
}

func TestTransformTrait_SubclassInMetadata(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":     "Stormheart",
			"type":     "trait",
			"class":    "beastheart",
			"level":    "2",
			"subclass": "spark",
		},
		Body:     "body",
		TypePath: []string{"feature", "trait", "beastheart", "level-2"},
		ItemID:   "stormheart",
	}
	out := TransformToSDKFormat("mcdm.beastheart.v1/feature.trait.beastheart.level-2/stormheart", parsed)
	meta := out["metadata"].(map[string]any)
	assertEqual(t, meta["subclass"], "spark")
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

// TestTransformTrait_WithAbility_PreservesBodySubHeadings guards against the
// data-loss risk described in FOLLOWUPS.md: a single-ability trait embeds its
// ability as a structured `ability` field, and its Body (FullBodySource) already
// excludes that annotated ability child. The Body may still contain unannotated
// sub-headings (tables, notes) that must NOT be dropped from the SDK effect.
func TestTransformTrait_WithAbility_PreservesBodySubHeadings(t *testing.T) {
	abilityChild := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":        "Faithful Strike",
			"type":        "ability",
			"action_type": "Main action",
		},
	}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":  "Faithful Friend",
			"type":  "trait",
			"class": "censor",
			"level": "1",
		},
		// FullBodySource excludes the annotated ability child but folds in any
		// unannotated sub-headings (e.g. a table). The embedded ability markdown
		// is NOT present here — only the intro text and an unannotated note.
		Body:     "You have the following ability.\n\n###### Bonding Note\n\nThe bond persists until death.",
		TypePath: []string{"feature", "trait", "censor", "level-1"},
		ItemID:   "faithful-friend",
		Children: map[string]*content.ParsedContent{
			"ability": abilityChild,
		},
	}

	out := TransformToSDKFormat("", parsed)

	// The embedded ability is surfaced as a structured field.
	if _, ok := out["ability"]; !ok {
		t.Fatal("expected embedded ability field on single-ability trait")
	}

	effects := out["effects"].([]map[string]any)
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	effect := effects[0]["effect"].(string)
	if !strings.Contains(effect, "You have the following ability.") {
		t.Errorf("trait effect should retain the intro text, got: %q", effect)
	}
	if !strings.Contains(effect, "Bonding Note") || !strings.Contains(effect, "The bond persists until death.") {
		t.Errorf("trait effect dropped an unannotated body sub-heading (data loss), got: %q", effect)
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

func TestTransformKit_NoSignatureAbility(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":          "Panther",
			"type":          "kit",
			"stamina_bonus": "+3",
			"speed_bonus":   "+2",
		},
		Body: "A swift melee kit.",
	}

	out := TransformToSDKFormat("", parsed)

	assertEqual(t, out["type"], "kit")
	assertEqual(t, out["content"], "A swift melee kit.")
	assertEqual(t, out["stamina_bonus"], "+3")
	assertEqual(t, out["speed_bonus"], "+2")

	// No signature ability should be present
	if _, ok := out["signature_ability"]; ok {
		t.Error("expected no signature_ability when Children is nil")
	}
}

func TestTransformKit_WithSignatureAbility(t *testing.T) {
	sigAbilityParsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":                      "Fade",
			"type":                      "ability",
			"subtype":                   "signature",
			"action_type":               "Main action",
			"keywords":                  []string{"Melee", "Ranged", "Strike", "Weapon"},
			"distance":                  "Melee 1 or ranged 10",
			"target":                    "One creature",
			"flavor":                    "A stab, and a few quick, careful steps back.",
			"power_roll_characteristic": "Might or Agility",
			"tier1":                     "3 + M or A damage; you can shift 1 square",
			"tier2":                     "6 + M or A damage; you can shift up to 2 squares",
			"tier3":                     "8 + M or A damage; you can shift up to 3 squares",
		},
		Body:     "ability body",
		TypePath: []string{"feature", "ability", "common"},
		ItemID:   "fade",
	}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":                  "Cloak and Dagger",
			"type":                  "kit",
			"stamina_bonus":         "+3 per echelon",
			"speed_bonus":           "+2",
			"melee_damage_bonus":    "+1/+1/+1",
			"ranged_damage_bonus":   "+1/+1/+1",
			"ranged_distance_bonus": "+5",
			"disengage_bonus":       "+1",
		},
		Body:   "Kit body text.",
		ItemID: "cloak-and-dagger",
		Children: map[string]*content.ParsedContent{
			"signature_ability": sigAbilityParsed,
		},
	}

	scc := "mcdm.heroes.v1/kit/cloak-and-dagger"
	out := TransformToSDKFormat(scc, parsed)

	// Kit fields
	assertEqual(t, out["type"], "kit")
	assertEqual(t, out["name"], "Cloak and Dagger")
	assertEqual(t, out["stamina_bonus"], "+3 per echelon")
	assertEqual(t, out["content"], "Kit body text.")

	// Signature ability should be a nested feature object
	sig, ok := out["signature_ability"].(map[string]any)
	if !ok {
		t.Fatal("expected signature_ability to be map[string]any")
	}

	assertEqual(t, sig["type"], "feature")
	assertEqual(t, sig["feature_type"], "ability")
	assertEqual(t, sig["name"], "Fade")
	assertEqual(t, sig["usage"], "Main action")
	assertEqual(t, sig["ability_type"], "Signature")
	assertEqual(t, sig["distance"], "Melee 1 or ranged 10")
	assertEqual(t, sig["target"], "One creature")
	assertEqual(t, sig["flavor"], "A stab, and a few quick, careful steps back.")

	// Effects should have power roll tiers
	effects, ok := sig["effects"].([]map[string]any)
	if !ok {
		t.Fatal("expected signature_ability effects to be []map[string]any")
	}
	if len(effects) < 1 {
		t.Fatal("expected at least 1 effect in signature ability")
	}
	assertEqual(t, effects[0]["roll"], "Power Roll + Might or Agility")
	assertEqual(t, effects[0]["tier1"], "3 + M or A damage; you can shift 1 square")

	// Keywords
	kw, ok := sig["keywords"].([]string)
	if !ok {
		t.Fatal("expected keywords to be []string")
	}
	if len(kw) != 4 || kw[0] != "Melee" {
		t.Errorf("unexpected keywords: %v", kw)
	}

	// Verify it serializes to valid JSON
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal kit with signature ability: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON output")
	}
}

func TestTransformAbility_JSONSchemaCompliant(t *testing.T) {
	// Verify the output can be serialized and contains required schema fields
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":                      "Brutal Slam",
			"type":                      "ability",
			"action_type":               "Main action",
			"power_roll_characteristic": "Might",
			"tier1":                     "3 + M damage",
			"tier2":                     "6 + M damage",
			"tier3":                     "9 + M damage",
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
