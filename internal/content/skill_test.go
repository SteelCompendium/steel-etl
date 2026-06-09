package content

import (
	"slices"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestSkillParserGroupedTypePath(t *testing.T) {
	p := &SkillParser{}
	sec := &parser.Section{
		Heading:    "Alchemy",
		Annotation: map[string]string{"type": "skill", "id": "alchemy", "group": "crafting"},
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if want := []string{"skill", "crafting"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
	if got.ItemID != "alchemy" {
		t.Errorf("ItemID = %q, want alchemy", got.ItemID)
	}
	if got.Frontmatter["type"] != "skill" {
		t.Errorf("type = %v, want skill", got.Frontmatter["type"])
	}
}

func TestSkillParserFlatWhenNoGroup(t *testing.T) {
	p := &SkillParser{}
	sec := &parser.Section{
		Heading:    "Alchemy",
		Annotation: map[string]string{"type": "skill", "id": "alchemy"},
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if want := []string{"skill"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v (flat fallback)", got.TypePath, want)
	}
}
