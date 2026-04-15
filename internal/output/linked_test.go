package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

func TestLinkedGenerator_Format(t *testing.T) {
	gen := &LinkedGenerator{}
	if got := gen.Format(); got != "md-linked" {
		t.Errorf("Format() = %q, want md-linked", got)
	}
}

func TestLinkedGenerator_WriteSection(t *testing.T) {
	dir := t.TempDir()

	registry := scc.NewRegistry()
	registry.Add("mcdm.heroes.v1/feature.ability.fury.level-1/gouge")
	registry.Add("mcdm.heroes.v1/condition/dazed")
	resolver := scc.NewResolver(registry, ".md")

	gen := &LinkedGenerator{
		BaseDir:  dir,
		Resolver: resolver,
	}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":  "Fury",
			"type":  "class",
			"class": "fury",
		},
		Body:     "See [Gouge](scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge) for details.",
		TypePath: []string{"class"},
		ItemID:   "fury",
	}

	err := gen.WriteSection("mcdm.heroes.v1/class/fury", parsed)
	if err != nil {
		t.Fatalf("WriteSection failed: %v", err)
	}

	expectedPath := filepath.Join(dir, "class", "fury.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	out := string(data)

	// Should have frontmatter
	if !strings.Contains(out, "name: Fury") {
		t.Error("expected frontmatter with name")
	}

	// scc: links should be resolved to file paths
	if strings.Contains(out, "scc:mcdm.heroes.v1") {
		t.Error("expected scc: links to be resolved")
	}
	if !strings.Contains(out, "feature/ability/fury/level-1/gouge.md") {
		t.Error("expected resolved path in output")
	}
}

func TestLinkedGenerator_NilAndEmpty(t *testing.T) {
	registry := scc.NewRegistry()
	resolver := scc.NewResolver(registry, ".md")
	gen := &LinkedGenerator{BaseDir: t.TempDir(), Resolver: resolver}

	if err := gen.WriteSection("some/code", nil); err != nil {
		t.Errorf("expected nil error for nil parsed, got %v", err)
	}
	if err := gen.WriteSection("", &content.ParsedContent{}); err != nil {
		t.Errorf("expected nil error for empty SCC, got %v", err)
	}
}

func TestLinkedGenerator_UnresolvedLinks(t *testing.T) {
	dir := t.TempDir()

	// Empty registry — links won't resolve
	registry := scc.NewRegistry()
	resolver := scc.NewResolver(registry, ".md")

	gen := &LinkedGenerator{
		BaseDir:  dir,
		Resolver: resolver,
	}

	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Fury",
			"type": "class",
		},
		Body:     "See [Unknown](scc:mcdm.heroes.v1/feature.ability.fury.level-1/unknown) link.",
		TypePath: []string{"class"},
		ItemID:   "fury",
	}

	err := gen.WriteSection("mcdm.heroes.v1/class/fury", parsed)
	if err != nil {
		t.Fatalf("WriteSection failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "class", "fury.md"))
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Unresolved link should remain as-is
	if !strings.Contains(string(data), "scc:mcdm.heroes.v1/feature.ability.fury.level-1/unknown") {
		t.Error("expected unresolved scc: link to remain unchanged")
	}
}
