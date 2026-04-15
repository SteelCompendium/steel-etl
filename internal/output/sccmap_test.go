package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

func TestSCCMapGenerator(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "scc-to-path.json")

	gen := &SCCMapGenerator{OutputPath: outputPath}

	// Write a few sections
	sections := []struct {
		scc    string
		parsed *content.ParsedContent
	}{
		{
			"mcdm.heroes.v1/class/fury",
			&content.ParsedContent{
				Frontmatter: map[string]any{"name": "Fury", "type": "class"},
				ItemID:      "fury",
			},
		},
		{
			"mcdm.heroes.v1/condition/dazed",
			&content.ParsedContent{
				Frontmatter: map[string]any{"name": "Dazed", "type": "condition"},
				ItemID:      "dazed",
			},
		},
	}

	for _, s := range sections {
		if err := gen.WriteSection(s.scc, s.parsed); err != nil {
			t.Fatalf("WriteSection failed: %v", err)
		}
	}

	if err := gen.Finalize(); err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	// Entries should be sorted by SCC
	if result[0]["scc"] != "mcdm.heroes.v1/class/fury" {
		t.Errorf("first entry scc = %v, want mcdm.heroes.v1/class/fury", result[0]["scc"])
	}
	if result[0]["path"] != "class/fury.md" {
		t.Errorf("first entry path = %v, want class/fury.md", result[0]["path"])
	}
}

func TestSCCMapGenerator_Empty(t *testing.T) {
	gen := &SCCMapGenerator{OutputPath: filepath.Join(t.TempDir(), "map.json")}

	// Finalize with no entries should be a no-op
	if err := gen.Finalize(); err != nil {
		t.Fatalf("Finalize on empty should not error: %v", err)
	}
}
