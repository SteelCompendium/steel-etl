package content

import (
	"slices"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestSkillGroupParserGroupLanding(t *testing.T) {
	p := &SkillGroupParser{}
	if p.Type() != "skill-group" {
		t.Fatalf("Type() = %q, want skill-group", p.Type())
	}
	sec := &parser.Section{
		Heading:    "Crafting Skills",
		Annotation: map[string]string{"type": "skill-group", "id": "crafting"},
		BodySource: "Skills from the crafting skill group are used in creation.",
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if want := []string{"skill", "group"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
	if got.ItemID != "crafting" {
		t.Errorf("ItemID = %q, want crafting", got.ItemID)
	}
	if got.Frontmatter["type"] != "skill-group" {
		t.Errorf("type = %v, want skill-group", got.Frontmatter["type"])
	}
	if got.Frontmatter["name"] != "Crafting Skills" {
		t.Errorf("name = %v, want \"Crafting Skills\"", got.Frontmatter["name"])
	}
}

func TestSkillGroupParserDerivesIDFromHeading(t *testing.T) {
	p := &SkillGroupParser{}
	sec := &parser.Section{
		Heading:    "Lore Skills",
		Annotation: map[string]string{"type": "skill-group"},
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got.ItemID != "lore-skills" {
		t.Errorf("ItemID = %q, want lore-skills (slug of heading when @id absent)", got.ItemID)
	}
	if want := []string{"skill", "group"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
}
