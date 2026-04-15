package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"gopkg.in/yaml.v3"
)

func TestYAMLGenerator_WriteSection(t *testing.T) {
	dir := t.TempDir()
	gen := &YAMLGenerator{BaseDir: dir}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":  "Gouge",
			"type":  "ability",
			"class": "fury",
			"cost":  "3 Ferocity",
		},
		Body: "Some ability body text.",
	}

	err := gen.WriteSection("mcdm.heroes.v1/feature.ability.fury.level-1/gouge", parsed)
	if err != nil {
		t.Fatalf("WriteSection failed: %v", err)
	}

	expectedPath := filepath.Join(dir, "feature", "ability", "fury", "level-1", "gouge.yaml")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid YAML output: %v", err)
	}

	if result["name"] != "Gouge" {
		t.Errorf("expected name=Gouge, got %v", result["name"])
	}
	// Abilities are now SDK-format: type=feature, feature_type=ability
	if result["type"] != "feature" {
		t.Errorf("expected type=feature, got %v", result["type"])
	}
	if result["feature_type"] != "ability" {
		t.Errorf("expected feature_type=ability, got %v", result["feature_type"])
	}
	// class and content are in metadata
	meta, ok := result["metadata"].(map[string]any)
	if !ok {
		t.Fatal("expected metadata object")
	}
	if meta["class"] != "fury" {
		t.Errorf("expected metadata.class=fury, got %v", meta["class"])
	}
	if meta["content"] != "Some ability body text." {
		t.Errorf("expected metadata.content field, got %v", meta["content"])
	}
	// effects array is required
	if result["effects"] == nil {
		t.Error("expected effects array")
	}
}

func TestYAMLGenerator_NilAndEmpty(t *testing.T) {
	gen := &YAMLGenerator{BaseDir: t.TempDir()}

	if err := gen.WriteSection("some/code", nil); err != nil {
		t.Errorf("expected nil error for nil parsed, got %v", err)
	}
	if err := gen.WriteSection("", &content.ParsedContent{}); err != nil {
		t.Errorf("expected nil error for empty SCC, got %v", err)
	}
}
