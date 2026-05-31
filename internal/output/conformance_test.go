package output

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
	ctx "github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// Conformance tests verify that TransformToSDKFormat produces output that is
// structurally compatible with the data-sdk-npm feature schema.
//
// These tests parse the real Heroes input document, run sections through the
// content parsers + SDK transform, and validate the shape/structure of the result.
//
// Note: byte-level comparison against the legacy data-gen baselines
// (data/data-rules-json/) was retired 2026-05-31. That reference predates the SCC
// link audit and is in the legacy colon-delimited format that the current pipeline
// no longer emits, so a faithful regeneration is impossible, and comparing against
// a steel-etl-generated baseline would be circular. The structural/schema tests
// below are the source of truth.

const heroesInputPath = "../../input/heroes/Draw Steel Heroes.md"

// --- Trait body rendering ---

// TestConformance_GrowingFerocity verifies a trait's body renders as a single
// effect that includes nested sub-table content (the Berserker/Reaver Growing
// Ferocity tables). This is structural coverage for RenderSubtree body assembly,
// not a legacy-file comparison.
func TestConformance_GrowingFerocity(t *testing.T) {
	etl := transformTraitFromDoc(t, "Fury", "1st-Level Features", "Growing Ferocity")

	assertRequiredSchemaFields(t, etl)
	if etl["name"] != "Growing Ferocity" {
		t.Errorf("name = %v, want Growing Ferocity", etl["name"])
	}
	if etl["type"] != "feature" {
		t.Errorf("type = %v, want feature", etl["type"])
	}
	if etl["feature_type"] != "trait" {
		t.Errorf("feature_type = %v, want trait", etl["feature_type"])
	}

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
