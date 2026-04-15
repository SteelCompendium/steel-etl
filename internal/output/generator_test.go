package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

func TestSCCToFilePath(t *testing.T) {
	tests := []struct {
		scc  string
		ext  string
		want string
	}{
		{"mcdm.heroes.v1/feature.ability.fury.level-1/gouge", ".md", "feature/ability/fury/level-1/gouge.md"},
		{"mcdm.heroes.v1/class/fury", ".md", "class/fury.md"},
		{"mcdm.heroes.v1/feature.trait.fury.level-1/growing-ferocity", ".md", "feature/trait/fury/level-1/growing-ferocity.md"},
		{"mcdm.heroes.v1/chapter/introduction", ".md", "chapter/introduction.md"},
		{"mcdm.heroes.v1/condition/dazed", ".md", "condition/dazed.md"},
		{"mcdm.heroes.v1/feature.ability.common/grab", ".md", "feature/ability/common/grab.md"},
		{"mcdm.heroes.v1/feature.trait.fury.level-1.boren/kit-bonuses", ".md", "feature/trait/fury/level-1/boren/kit-bonuses.md"},
		{"mcdm.heroes.v1/class/fury", ".json", "class/fury.json"},
		{"mcdm.heroes.v1/class/fury", ".yaml", "class/fury.yaml"},
	}

	for _, tt := range tests {
		got := SCCToFilePath(tt.scc, tt.ext)
		if got != tt.want {
			t.Errorf("SCCToFilePath(%q, %q) = %q, want %q", tt.scc, tt.ext, got, tt.want)
		}
	}
}

func TestWriteSection(t *testing.T) {
	dir := t.TempDir()
	gen := &MarkdownGenerator{BaseDir: dir}

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

	// Verify file exists
	expectedPath := filepath.Join(dir, "feature", "ability", "fury", "level-1", "gouge.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)

	// Verify frontmatter
	if !strings.Contains(content, "---") {
		t.Error("expected YAML frontmatter delimiters")
	}
	if !strings.Contains(content, "name: Gouge") {
		t.Error("expected name: Gouge in frontmatter")
	}
	if !strings.Contains(content, "class: fury") {
		t.Error("expected class: fury in frontmatter")
	}
	if !strings.Contains(content, "cost: 3 Ferocity") {
		t.Error("expected cost in frontmatter")
	}

	// Verify body
	if !strings.Contains(content, "Some ability body text.") {
		t.Error("expected body text in output")
	}
}

func TestWriteSectionNilParsed(t *testing.T) {
	gen := &MarkdownGenerator{BaseDir: t.TempDir()}
	err := gen.WriteSection("some/code", nil)
	if err != nil {
		t.Errorf("expected nil error for nil parsed, got %v", err)
	}
}

func TestWriteSectionEmptySCC(t *testing.T) {
	gen := &MarkdownGenerator{BaseDir: t.TempDir()}
	err := gen.WriteSection("", &content.ParsedContent{})
	if err != nil {
		t.Errorf("expected nil error for empty SCC, got %v", err)
	}
}
