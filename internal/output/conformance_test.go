package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
	ctx "github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// Conformance tests verify that TransformToSDKFormat produces output structurally
// compatible with the legacy data-rules-json format (data-sdk-npm feature schema).
//
// These tests parse the real Heroes input document, run sections through the
// content parsers + SDK transform, and compare key fields against the legacy
// JSON files in data/data-rules-json/.

const (
	heroesInputPath = "../../input/heroes/Draw Steel Heroes.md"
	legacyBasePath  = "../../../data/data-rules-json"
)

// --- Ability conformance tests ---

func TestConformance_BrutalSlam(t *testing.T) {
	legacy := loadLegacyJSON(t, "Abilities/Fury/1st-Level Features/Brutal Slam.json")
	etl := transformAbilityFromDoc(t, "Fury", "1st-Level Features", "Brutal Slam")

	assertRequiredSchemaFields(t, etl)
	assertTopLevelMatch(t, etl, legacy, "name")
	assertTopLevelMatch(t, etl, legacy, "type")
	assertTopLevelMatch(t, etl, legacy, "feature_type")
	assertTopLevelMatch(t, etl, legacy, "usage")
	assertTopLevelMatch(t, etl, legacy, "distance")
	assertTopLevelMatch(t, etl, legacy, "target")

	// Keywords
	assertKeywordsMatch(t, etl, legacy)

	// Flavor text
	assertTopLevelMatch(t, etl, legacy, "flavor")

	// Effects: should have power roll with tiers
	etlEffects := getEffects(t, etl)
	legacyEffects := getEffects(t, legacy)
	if len(etlEffects) < 1 || len(legacyEffects) < 1 {
		t.Fatal("both should have at least 1 effect")
	}

	// Power roll effect
	assertEffectFieldMatch(t, etlEffects[0], legacyEffects[0], "roll")
	assertEffectFieldMatch(t, etlEffects[0], legacyEffects[0], "tier1")
	assertEffectFieldMatch(t, etlEffects[0], legacyEffects[0], "tier2")
	assertEffectFieldMatch(t, etlEffects[0], legacyEffects[0], "tier3")

	// Metadata
	assertMetadataField(t, etl, "class", "fury")
	assertMetadataFieldInt(t, etl, "level", 1)
}

func TestConformance_HitAndRun(t *testing.T) {
	legacy := loadLegacyJSON(t, "Abilities/Fury/1st-Level Features/Hit and Run.json")
	etl := transformAbilityFromDoc(t, "Fury", "1st-Level Features", "Hit and Run")

	assertRequiredSchemaFields(t, etl)
	assertTopLevelMatch(t, etl, legacy, "name")
	assertTopLevelMatch(t, etl, legacy, "type")
	assertTopLevelMatch(t, etl, legacy, "feature_type")

	// Should have power roll + effect
	etlEffects := getEffects(t, etl)
	if len(etlEffects) < 2 {
		t.Fatalf("expected at least 2 effects (roll + effect), got %d", len(etlEffects))
	}

	// Effect entry
	legacyEffects := getEffects(t, legacy)
	if len(legacyEffects) >= 2 {
		assertEffectFieldMatch(t, etlEffects[1], legacyEffects[1], "effect")
	}
}

func TestConformance_ToTheDeath(t *testing.T) {
	legacy := loadLegacyJSON(t, "Abilities/Fury/1st-Level Features/To the Death.json")
	etl := transformAbilityFromDoc(t, "Fury", "1st-Level Features", "To the Death!")

	assertRequiredSchemaFields(t, etl)

	// Name may differ slightly (exclamation mark handling)
	if etl["name"] == nil {
		t.Error("name should be present")
	}

	assertTopLevelMatch(t, etl, legacy, "type")
	assertTopLevelMatch(t, etl, legacy, "feature_type")
}

// --- Trait conformance tests ---

func TestConformance_GrowingFerocity(t *testing.T) {
	legacy := loadLegacyJSON(t, "Features/Fury/1st-Level Features/Growing Ferocity.json")
	etl := transformTraitFromDoc(t, "Fury", "1st-Level Features", "Growing Ferocity")

	assertRequiredSchemaFields(t, etl)
	assertTopLevelMatch(t, etl, legacy, "name")
	assertTopLevelMatch(t, etl, legacy, "type")
	assertTopLevelMatch(t, etl, legacy, "feature_type")

	// Effects: trait body as single effect
	etlEffects := getEffects(t, etl)
	if len(etlEffects) < 1 {
		t.Fatal("expected at least 1 effect")
	}

	effectText, ok := etlEffects[0]["effect"].(string)
	if !ok || effectText == "" {
		t.Error("trait effect should contain body text")
	}

	// Body should include sub-heading content (Berserker/Reaver tables)
	if !strings.Contains(effectText, "Berserker") {
		t.Error("effect should include Berserker Growing Ferocity Table content")
	}
	if !strings.Contains(effectText, "Reaver") {
		t.Error("effect should include Reaver Growing Ferocity Table content")
	}

	assertMetadataField(t, etl, "class", "fury")
	assertMetadataFieldInt(t, etl, "level", 1)
}

// --- Feature schema field optionality (data-gen#94) ---

func TestConformance_UsageIsOptional(t *testing.T) {
	// Per data-gen#94: usage should be optional, many features don't have it.
	// Traits never have usage; verify they still pass schema validation.
	traits := findTraitsInDoc(t, "Fury", "1st-Level Features")

	for _, section := range traits {
		t.Run(section.Heading, func(t *testing.T) {
			parsed := parseTrait(t, section, "fury", "1")
			out := TransformToSDKFormat("", parsed)

			// usage must NOT be present on traits
			if _, ok := out["usage"]; ok {
				t.Error("traits should not have usage field")
			}

			// But should still have all required fields
			assertRequiredSchemaFields(t, out)
		})
	}
}

func TestConformance_NameAlwaysPresent(t *testing.T) {
	// Per data-gen#94: name should be required (every feature has it).
	// Verify all abilities and traits have name set.
	abilities := findAbilitiesInDoc(t, "Fury", "1st-Level Features")
	for _, section := range abilities {
		t.Run("ability/"+section.Heading, func(t *testing.T) {
			parsed := parseAbility(t, section, "fury", "1")
			out := TransformToSDKFormat("", parsed)
			if out["name"] == nil || out["name"] == "" {
				t.Error("name must always be present on abilities")
			}
		})
	}

	traits := findTraitsInDoc(t, "Fury", "1st-Level Features")
	for _, section := range traits {
		t.Run("trait/"+section.Heading, func(t *testing.T) {
			parsed := parseTrait(t, section, "fury", "1")
			out := TransformToSDKFormat("", parsed)
			if out["name"] == nil || out["name"] == "" {
				t.Error("name must always be present on traits")
			}
		})
	}
}

// --- Schema structure validation ---

func TestConformance_SchemaStructure_Abilities(t *testing.T) {
	// Validate that all Fury 1st-level abilities have the required schema structure
	abilities := findAbilitiesInDoc(t, "Fury", "1st-Level Features")

	for _, section := range abilities {
		t.Run(section.Heading, func(t *testing.T) {
			parsed := parseAbility(t, section, "fury", "1")
			scc := "mcdm.heroes.v1/feature.ability.fury.level-1/" + content.Slugify(content.CleanHeading(section.Heading))
			out := TransformToSDKFormat(scc, parsed)

			assertRequiredSchemaFields(t, out)

			// type must be "feature"
			if out["type"] != "feature" {
				t.Errorf("type = %v, want feature", out["type"])
			}

			// feature_type must be "ability"
			if out["feature_type"] != "ability" {
				t.Errorf("feature_type = %v, want ability", out["feature_type"])
			}

			// Must have effects array with at least 1 item
			effects := getEffects(t, out)
			if len(effects) < 1 {
				t.Error("effects must have at least 1 item (schema minItems: 1)")
			}

			// Each effect must satisfy one of the anyOf constraints
			for i, eff := range effects {
				if !isValidEffect(eff) {
					data, _ := json.MarshalIndent(eff, "", "  ")
					t.Errorf("effect[%d] does not satisfy any schema constraint:\n%s", i, data)
				}
			}

			// Must NOT have fields that aren't in the schema
			forbiddenTopLevel := []string{
				"power_roll_characteristic", "tier1", "tier2", "tier3",
				"action_type", "subtype", "class", "level", "scc", "source",
				"content",
			}
			for _, key := range forbiddenTopLevel {
				if _, ok := out[key]; ok {
					t.Errorf("top-level field %q should not be present (moved to metadata or renamed)", key)
				}
			}

			// Verify JSON roundtrip
			data, err := json.Marshal(out)
			if err != nil {
				t.Errorf("failed to marshal to JSON: %v", err)
			}
			var roundtrip map[string]any
			if err := json.Unmarshal(data, &roundtrip); err != nil {
				t.Errorf("failed to unmarshal from JSON: %v", err)
			}
		})
	}
}

func TestConformance_SchemaStructure_Traits(t *testing.T) {
	traits := findTraitsInDoc(t, "Fury", "1st-Level Features")

	for _, section := range traits {
		t.Run(section.Heading, func(t *testing.T) {
			parsed := parseTrait(t, section, "fury", "1")
			scc := "mcdm.heroes.v1/feature.trait.fury.level-1/" + content.Slugify(content.CleanHeading(section.Heading))
			out := TransformToSDKFormat(scc, parsed)

			assertRequiredSchemaFields(t, out)

			if out["type"] != "feature" {
				t.Errorf("type = %v, want feature", out["type"])
			}
			if out["feature_type"] != "trait" {
				t.Errorf("feature_type = %v, want trait", out["feature_type"])
			}

			effects := getEffects(t, out)
			if len(effects) < 1 {
				t.Error("effects must have at least 1 item")
			}
		})
	}
}

func TestConformance_NoUnevaluatedProperties(t *testing.T) {
	// The feature schema should use unevaluatedProperties: false (draft 2019-09)
	// to allow composition via allOf (see data-sdk-npm#13).
	// Verify we don't emit any fields outside the schema's properties list.
	allowedTopLevel := map[string]bool{
		"name": true, "icon": true, "type": true, "feature_type": true,
		"usage": true, "cost": true, "ability_type": true, "keywords": true,
		"distance": true, "target": true, "trigger": true, "effects": true,
		"flavor": true, "metadata": true,
	}

	abilities := findAbilitiesInDoc(t, "Fury", "1st-Level Features")
	for _, section := range abilities {
		parsed := parseAbility(t, section, "fury", "1")
		out := TransformToSDKFormat("", parsed)

		for key := range out {
			if !allowedTopLevel[key] {
				t.Errorf("ability %q: unexpected top-level field %q (not in schema)", section.Heading, key)
			}
		}
	}
}

// --- Helpers ---

func loadLegacyJSON(t *testing.T, relPath string) map[string]any {
	t.Helper()
	fullPath := filepath.Join(legacyBasePath, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Skipf("legacy file not found: %s", fullPath)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid legacy JSON %s: %v", relPath, err)
	}
	return result
}

func loadHeroesDoc(t *testing.T) *parser.Document {
	t.Helper()
	data, err := os.ReadFile(heroesInputPath)
	if err != nil {
		t.Skipf("heroes input not found: %v", err)
	}
	doc, err := parser.ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}
	return doc
}

func findClassSection(t *testing.T, doc *parser.Document, className string) *parser.Section {
	t.Helper()
	for _, top := range doc.Sections {
		for _, child := range top.Children {
			if strings.EqualFold(child.Heading, className) && child.Type() == "class" {
				return child
			}
		}
		// Check top level too
		if strings.EqualFold(top.Heading, className) && top.Type() == "class" {
			return top
		}
	}
	t.Fatalf("class %q not found in document", className)
	return nil
}

func findFeatureGroup(t *testing.T, classSection *parser.Section, groupName string) *parser.Section {
	t.Helper()
	for _, child := range classSection.Children {
		if child.Heading == groupName && child.Type() == "feature-group" {
			return child
		}
	}
	t.Fatalf("feature-group %q not found under %q", groupName, classSection.Heading)
	return nil
}

func findSectionByHeading(t *testing.T, parent *parser.Section, heading string) *parser.Section {
	t.Helper()
	if found := findSectionByHeadingRecursive(parent, heading); found != nil {
		return found
	}
	t.Fatalf("section %q not found under %q", heading, parent.Heading)
	return nil
}

func findSectionByHeadingRecursive(parent *parser.Section, heading string) *parser.Section {
	for _, child := range parent.Children {
		cleanName := content.CleanHeading(child.Heading)
		if cleanName == heading || child.Heading == heading {
			return child
		}
		if found := findSectionByHeadingRecursive(child, heading); found != nil {
			return found
		}
	}
	return nil
}

func findAbilitiesInDoc(t *testing.T, className, featureGroupName string) []*parser.Section {
	t.Helper()
	doc := loadHeroesDoc(t)
	classSection := findClassSection(t, doc, className)
	fg := findFeatureGroup(t, classSection, featureGroupName)

	var abilities []*parser.Section
	collectAbilitiesRecursive(fg, &abilities)
	if len(abilities) == 0 {
		t.Fatalf("no abilities found under %s > %s", className, featureGroupName)
	}
	return abilities
}

func collectAbilitiesRecursive(section *parser.Section, abilities *[]*parser.Section) {
	for _, child := range section.Children {
		if child.Type() == "ability" {
			*abilities = append(*abilities, child)
		}
		collectAbilitiesRecursive(child, abilities)
	}
}

func findTraitsInDoc(t *testing.T, className, featureGroupName string) []*parser.Section {
	t.Helper()
	doc := loadHeroesDoc(t)
	classSection := findClassSection(t, doc, className)
	fg := findFeatureGroup(t, classSection, featureGroupName)

	var traits []*parser.Section
	for _, child := range fg.Children {
		if child.Type() == "feature" {
			traits = append(traits, child)
		}
	}
	if len(traits) == 0 {
		t.Fatalf("no traits found under %s > %s", className, featureGroupName)
	}
	return traits
}

func transformAbilityFromDoc(t *testing.T, className, featureGroupName, abilityName string) map[string]any {
	t.Helper()
	doc := loadHeroesDoc(t)
	classSection := findClassSection(t, doc, className)
	fg := findFeatureGroup(t, classSection, featureGroupName)
	section := findSectionByHeading(t, fg, abilityName)

	parsed := parseAbility(t, section, strings.ToLower(className), "1")
	scc := "mcdm.heroes.v1/feature.ability." + strings.ToLower(className) + ".level-1/" + content.Slugify(content.CleanHeading(section.Heading))
	return TransformToSDKFormat(scc, parsed)
}

func transformTraitFromDoc(t *testing.T, className, featureGroupName, traitName string) map[string]any {
	t.Helper()
	doc := loadHeroesDoc(t)
	classSection := findClassSection(t, doc, className)
	fg := findFeatureGroup(t, classSection, featureGroupName)
	section := findSectionByHeading(t, fg, traitName)

	parsed := parseTrait(t, section, strings.ToLower(className), "1")
	scc := "mcdm.heroes.v1/feature.trait." + strings.ToLower(className) + ".level-1/" + content.Slugify(content.CleanHeading(section.Heading))
	return TransformToSDKFormat(scc, parsed)
}

func parseAbility(t *testing.T, section *parser.Section, classID, level string) *content.ParsedContent {
	t.Helper()
	stack := ctx.NewContextStack(ctx.Metadata{"book": "mcdm.heroes.v1"})
	stack.Push(2, ctx.Metadata{"type": "class", "id": classID})
	stack.Push(3, ctx.Metadata{"type": "feature-group", "level": level})

	p := &content.AbilityParser{}
	parsed, err := p.Parse(stack, section)
	if err != nil {
		t.Fatalf("AbilityParser.Parse failed for %q: %v", section.Heading, err)
	}
	return parsed
}

func parseTrait(t *testing.T, section *parser.Section, classID, level string) *content.ParsedContent {
	t.Helper()
	stack := ctx.NewContextStack(ctx.Metadata{"book": "mcdm.heroes.v1"})
	stack.Push(2, ctx.Metadata{"type": "class", "id": classID})
	stack.Push(3, ctx.Metadata{"type": "feature-group", "level": level})

	p := &content.FeatureParser{}
	parsed, err := p.Parse(stack, section)
	if err != nil {
		t.Fatalf("FeatureParser.Parse failed for %q: %v", section.Heading, err)
	}
	return parsed
}

func assertRequiredSchemaFields(t *testing.T, out map[string]any) {
	t.Helper()
	// name should be required per data-gen#94 (every feature has it)
	if out["name"] == nil {
		t.Error("missing required field: name")
	}
	if out["type"] == nil {
		t.Error("missing required field: type")
	}
	if out["feature_type"] == nil {
		t.Error("missing required field: feature_type")
	}
	if out["effects"] == nil {
		t.Error("missing required field: effects")
	}
}

func assertTopLevelMatch(t *testing.T, etl, legacy map[string]any, field string) {
	t.Helper()
	etlVal, _ := etl[field].(string)
	legacyVal, _ := legacy[field].(string)
	if etlVal != legacyVal {
		t.Errorf("%s: got %q, want %q", field, etlVal, legacyVal)
	}
}

func assertKeywordsMatch(t *testing.T, etl, legacy map[string]any) {
	t.Helper()
	etlKw := toStringSlice(etl["keywords"])
	legacyKw := toStringSlice(legacy["keywords"])

	if len(etlKw) != len(legacyKw) {
		t.Errorf("keywords length: got %d, want %d", len(etlKw), len(legacyKw))
		return
	}
	for i := range etlKw {
		if etlKw[i] != legacyKw[i] {
			t.Errorf("keywords[%d]: got %q, want %q", i, etlKw[i], legacyKw[i])
		}
	}
}

func assertMetadataField(t *testing.T, out map[string]any, field, want string) {
	t.Helper()
	meta, ok := out["metadata"].(map[string]any)
	if !ok {
		t.Errorf("metadata not found when checking field %q", field)
		return
	}
	got, _ := meta[field].(string)
	if got != want {
		t.Errorf("metadata.%s: got %q, want %q", field, got, want)
	}
}

func assertMetadataFieldInt(t *testing.T, out map[string]any, field string, want int) {
	t.Helper()
	meta, ok := out["metadata"].(map[string]any)
	if !ok {
		t.Errorf("metadata not found when checking field %q", field)
		return
	}
	got, ok := meta[field].(int)
	if !ok {
		t.Errorf("metadata.%s: not an int, got %T(%v)", field, meta[field], meta[field])
		return
	}
	if got != want {
		t.Errorf("metadata.%s: got %d, want %d", field, got, want)
	}
}

func assertEffectFieldMatch(t *testing.T, etlEffect, legacyEffect map[string]any, field string) {
	t.Helper()
	etlVal, _ := etlEffect[field].(string)
	legacyVal, _ := legacyEffect[field].(string)
	if etlVal != legacyVal {
		t.Errorf("effect.%s: got %q, want %q", field, etlVal, legacyVal)
	}
}

func getEffects(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()

	// Handle both typed and JSON-deserialized forms
	switch v := out["effects"].(type) {
	case []map[string]any:
		return v
	case []any:
		var result []map[string]any
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				result = append(result, m)
			}
		}
		return result
	default:
		t.Fatalf("effects is unexpected type: %T", out["effects"])
		return nil
	}
}

func toStringSlice(v any) []string {
	switch arr := v.(type) {
	case []string:
		return arr
	case []any:
		var result []string
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// isValidEffect checks if an effect satisfies one of the schema's anyOf constraints:
// 1. Has "effect" field
// 2. Has "roll" + "tier1" + "tier2" + "tier3"
// 3. Has "roll" + "t1" + "t2" + "t3"
// 4. Has "roll" + "11 or lower" + "12-16" + "17+"
func isValidEffect(eff map[string]any) bool {
	if _, ok := eff["effect"]; ok {
		return true
	}
	if _, ok := eff["roll"]; ok {
		if _, ok1 := eff["tier1"]; ok1 {
			if _, ok2 := eff["tier2"]; ok2 {
				if _, ok3 := eff["tier3"]; ok3 {
					return true
				}
			}
		}
		if _, ok1 := eff["t1"]; ok1 {
			if _, ok2 := eff["t2"]; ok2 {
				if _, ok3 := eff["t3"]; ok3 {
					return true
				}
			}
		}
		if _, ok1 := eff["11 or lower"]; ok1 {
			if _, ok2 := eff["12-16"]; ok2 {
				if _, ok3 := eff["17+"]; ok3 {
					return true
				}
			}
		}
	}
	return false
}
