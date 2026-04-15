package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

func TestDSELinkedGenerator_Format(t *testing.T) {
	gen := &DSELinkedGenerator{}
	if got := gen.Format(); got != "md-dse-linked" {
		t.Errorf("Format() = %q, want md-dse-linked", got)
	}
}

func TestDSELinkedGenerator_WriteSection_Ability(t *testing.T) {
	dir := t.TempDir()

	registry := scc.NewRegistry()
	registry.Add("mcdm.heroes.v1/condition/dazed")
	resolver := scc.NewResolver(registry, ".md")

	gen := &DSELinkedGenerator{
		BaseDir:  dir,
		Resolver: resolver,
	}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":                      "Gouge",
			"type":                      "ability",
			"class":                     "fury",
			"cost":                      "3 Ferocity",
			"power_roll_characteristic": "Might",
			"tier1":                     "4 + M damage",
			"tier2":                     "7 + M damage",
			"tier3":                     "10 + M damage; target is [Dazed](scc:mcdm.heroes.v1/condition/dazed)",
		},
		Body:     "Ability with link to scc:mcdm.heroes.v1/condition/dazed.",
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

	// Should have DSE format (ds-feature codeblock for ability)
	if !strings.Contains(out, "```ds-feature") {
		t.Error("expected ds-feature codeblock")
	}

	// Should have DSE frontmatter fields
	if !strings.Contains(out, "item_id: gouge") {
		t.Error("expected item_id in frontmatter")
	}

	// DSE-specific frontmatter should be present
	if !strings.Contains(out, "source: mcdm.heroes.v1") {
		t.Error("expected source field in frontmatter")
	}
	if !strings.Contains(out, "feature_type: ability") {
		t.Error("expected feature_type in frontmatter")
	}
}

func TestDSELinkedGenerator_WriteSection_Condition(t *testing.T) {
	dir := t.TempDir()

	registry := scc.NewRegistry()
	resolver := scc.NewResolver(registry, ".md")

	gen := &DSELinkedGenerator{
		BaseDir:  dir,
		Resolver: resolver,
	}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Dazed",
			"type": "condition",
		},
		Body:     "A dazed creature can do only one thing.",
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

	// Should have body text
	if !strings.Contains(out, "A dazed creature") {
		t.Error("expected body text")
	}
}

func TestDSELinkedGenerator_NilAndEmpty(t *testing.T) {
	registry := scc.NewRegistry()
	resolver := scc.NewResolver(registry, ".md")
	gen := &DSELinkedGenerator{BaseDir: t.TempDir(), Resolver: resolver}

	if err := gen.WriteSection("some/code", nil); err != nil {
		t.Errorf("expected nil error for nil parsed, got %v", err)
	}
	if err := gen.WriteSection("", &content.ParsedContent{}); err != nil {
		t.Errorf("expected nil error for empty SCC, got %v", err)
	}
}
