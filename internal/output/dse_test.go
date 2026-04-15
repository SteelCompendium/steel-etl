package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

func TestDSEGenerator_Ability(t *testing.T) {
	dir := t.TempDir()
	gen := &DSEGenerator{BaseDir: dir}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":                      "Gouge",
			"type":                      "ability",
			"class":                     "fury",
			"cost":                      "3 Ferocity",
			"action_type":               "Main action",
			"keywords":                  []string{"Melee", "Strike", "Weapon"},
			"distance":                  "Melee 1",
			"target":                    "One creature",
			"flavor":                    "Your sharp claws tear into your foe.",
			"power_roll_characteristic": "Might",
			"tier1":                     "4 + M damage",
			"tier2":                     "7 + M damage",
			"tier3":                     "10 + M damage",
			"scc":                       "mcdm.heroes.v1/feature.ability.fury.level-1/gouge",
		},
		Body:     "Ability body text.",
		TypePath: []string{"feature", "ability", "fury", "level-1"},
		ItemID:   "gouge",
	}

	err := gen.WriteSection("mcdm.heroes.v1/feature.ability.fury.level-1/gouge", parsed)
	if err != nil {
		t.Fatalf("WriteSection failed: %v", err)
	}

	expectedPath := filepath.Join(dir, "feature", "ability", "fury", "level-1", "gouge.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	out := string(data)

	// Should have frontmatter
	if !strings.Contains(out, "---\n") {
		t.Error("expected frontmatter delimiters")
	}

	// Should have DSE-specific fields in frontmatter
	if !strings.Contains(out, "feature_type: ability") {
		t.Error("expected feature_type: ability in frontmatter")
	}
	if !strings.Contains(out, "item_id: gouge") {
		t.Error("expected item_id in frontmatter")
	}
	if !strings.Contains(out, "cost_amount:") {
		t.Error("expected cost_amount in frontmatter")
	}
	if !strings.Contains(out, "cost_resource: Ferocity") {
		t.Error("expected cost_resource in frontmatter")
	}

	// Should have ds-feature codeblock
	if !strings.Contains(out, "```ds-feature") {
		t.Error("expected ds-feature codeblock")
	}
	if !strings.Contains(out, "type: feature") {
		t.Error("expected type: feature in codeblock")
	}
	if !strings.Contains(out, "name: Gouge") {
		t.Error("expected name: Gouge in codeblock")
	}
	if !strings.Contains(out, "Power Roll + Might") {
		t.Error("expected power roll in effects")
	}
}

func TestDSEGenerator_Condition(t *testing.T) {
	dir := t.TempDir()
	gen := &DSEGenerator{BaseDir: dir}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Dazed",
			"type": "condition",
		},
		Body:     "A dazed creature can do only one thing on their turn.",
		TypePath: []string{"condition"},
		ItemID:   "dazed",
	}

	err := gen.WriteSection("mcdm.heroes.v1/condition/dazed", parsed)
	if err != nil {
		t.Fatalf("WriteSection failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "condition", "dazed.md"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	out := string(data)

	// Conditions should NOT have ds-feature codeblock
	if strings.Contains(out, "```ds-feature") {
		t.Error("conditions should not have ds-feature codeblock")
	}

	// Should have plain markdown body
	if !strings.Contains(out, "A dazed creature") {
		t.Error("expected body text in output")
	}
}

func TestDSEGenerator_Trait(t *testing.T) {
	dir := t.TempDir()
	gen := &DSEGenerator{BaseDir: dir}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":  "Growing Ferocity",
			"type":  "trait",
			"class": "fury",
			"level": "1",
		},
		Body:     "At the start of each of your turns, you gain 1d3 ferocity.",
		TypePath: []string{"feature", "trait", "fury", "level-1"},
		ItemID:   "growing-ferocity",
	}

	err := gen.WriteSection("mcdm.heroes.v1/feature.trait.fury.level-1/growing-ferocity", parsed)
	if err != nil {
		t.Fatalf("WriteSection failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "feature", "trait", "fury", "level-1", "growing-ferocity.md"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	out := string(data)

	// Traits should have ds-feature codeblock
	if !strings.Contains(out, "```ds-feature") {
		t.Error("traits should have ds-feature codeblock")
	}
	if !strings.Contains(out, "feature_type: trait") {
		t.Error("expected feature_type: trait")
	}
}

func TestParseCost(t *testing.T) {
	tests := []struct {
		input        string
		wantAmount   string
		wantResource string
	}{
		{"3 Ferocity", "3", "Ferocity"},
		{"11 Piety", "11", "Piety"},
		{"5 Wrath", "5", "Wrath"},
		{"free", "free", ""},
	}

	for _, tt := range tests {
		amount, resource := parseCost(tt.input)
		if amount != tt.wantAmount {
			t.Errorf("parseCost(%q) amount = %q, want %q", tt.input, amount, tt.wantAmount)
		}
		if resource != tt.wantResource {
			t.Errorf("parseCost(%q) resource = %q, want %q", tt.input, resource, tt.wantResource)
		}
	}
}
