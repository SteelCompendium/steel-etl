package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestProjectParser(t *testing.T) {
	p := &ProjectParser{}

	if p.Type() != "project" {
		t.Errorf("Type() = %q, want %q", p.Type(), "project")
	}

	section := &parser.Section{
		Heading:      "Build Airship",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "project"},
		BodySource:   "**Project Goal:** 1,000\n\nYou build a working airship.",
	}

	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Build Airship" {
		t.Errorf("name = %v, want Build Airship", result.Frontmatter["name"])
	}
	if result.Frontmatter["type"] != "project" {
		t.Errorf("type = %v, want project", result.Frontmatter["type"])
	}
	if result.ItemID != "build-airship" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "build-airship")
	}
	if len(result.TypePath) != 1 || result.TypePath[0] != "project" {
		t.Errorf("TypePath = %v, want [project]", result.TypePath)
	}
}

func TestGodParser(t *testing.T) {
	p := &GodParser{}

	if p.Type() != "god" {
		t.Errorf("Type() = %q, want %q", p.Type(), "god")
	}

	section := &parser.Section{
		Heading:      "Cavall",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "god"},
		BodySource:   "The god of duty and protection.",
	}

	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Cavall" {
		t.Errorf("name = %v, want Cavall", result.Frontmatter["name"])
	}
	if result.Frontmatter["type"] != "god" {
		t.Errorf("type = %v, want god", result.Frontmatter["type"])
	}
	if result.ItemID != "cavall" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "cavall")
	}
	if len(result.TypePath) != 1 || result.TypePath[0] != "god" {
		t.Errorf("TypePath = %v, want [god]", result.TypePath)
	}
}

// TestGodParserIDOverride verifies an explicit @id is used instead of the
// slugified heading (needed for names with non-ASCII characters like "Adûn").
func TestGodParserIDOverride(t *testing.T) {
	section := &parser.Section{
		Heading:      "Adûn",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "god", "id": "adun"},
		BodySource:   "The god of the sun.",
	}

	result, err := (&GodParser{}).Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.ItemID != "adun" {
		t.Errorf("ItemID = %q, want %q (explicit @id)", result.ItemID, "adun")
	}
}

// TestProjectParserRegistered confirms the new parsers are wired into the registry.
func TestProjectGodRegistered(t *testing.T) {
	r := NewRegistry()
	for _, typ := range []string{"project", "god"} {
		if !r.Has(typ) {
			t.Errorf("registry missing parser for %q", typ)
		}
	}
}
