package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

func TestAggregateGenerator(t *testing.T) {
	dir := t.TempDir()
	gen := &AggregateGenerator{BaseDir: dir}

	sections := []struct {
		scc    string
		parsed *content.ParsedContent
	}{
		{
			"mcdm.heroes.v1/class/fury",
			&content.ParsedContent{
				Frontmatter: map[string]any{"name": "Fury", "type": "class"},
				Body:        "Fury class description.",
				ItemID:      "fury",
			},
		},
		{
			"mcdm.heroes.v1/class/shadow",
			&content.ParsedContent{
				Frontmatter: map[string]any{"name": "Shadow", "type": "class"},
				Body:        "Shadow class description.",
				ItemID:      "shadow",
			},
		},
		{
			"mcdm.heroes.v1/condition/dazed",
			&content.ParsedContent{
				Frontmatter: map[string]any{"name": "Dazed", "type": "condition"},
				Body:        "A dazed creature...",
				ItemID:      "dazed",
			},
		},
	}

	for _, s := range sections {
		if err := gen.WriteSection(s.scc, s.parsed); err != nil {
			t.Fatalf("WriteSection failed: %v", err)
		}
	}

	// Verify section files were written
	furyPath := filepath.Join(dir, "class", "fury.md")
	if _, err := os.Stat(furyPath); os.IsNotExist(err) {
		t.Error("expected fury.md to be written")
	}

	// Finalize to write indexes
	if err := gen.Finalize(); err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	// Check class index
	classIndex := filepath.Join(dir, "_index", "class.md")
	data, err := os.ReadFile(classIndex)
	if err != nil {
		t.Fatalf("failed to read class index: %v", err)
	}
	indexContent := string(data)
	if !strings.Contains(indexContent, "Fury") {
		t.Error("class index should mention Fury")
	}
	if !strings.Contains(indexContent, "Shadow") {
		t.Error("class index should mention Shadow")
	}
	if !strings.Contains(indexContent, "Total: 2") {
		t.Error("class index should show total: 2")
	}

	// Check master index
	masterIndex := filepath.Join(dir, "_index", "README.md")
	data, err = os.ReadFile(masterIndex)
	if err != nil {
		t.Fatalf("failed to read master index: %v", err)
	}
	masterContent := string(data)
	if !strings.Contains(masterContent, "Total items: 3") {
		t.Errorf("master index should show total items: 3, got:\n%s", masterContent)
	}
}
