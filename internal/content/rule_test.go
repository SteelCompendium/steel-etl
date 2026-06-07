package content

import (
	"slices"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestRuleParserGroupedTypePath(t *testing.T) {
	p := &RuleParser{}
	sec := &parser.Section{
		Heading:      "Flanking",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "rule", "group": "combat", "id": "flanking"},
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if want := []string{"rule", "combat"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
	if got.ItemID != "flanking" {
		t.Errorf("ItemID = %q, want flanking", got.ItemID)
	}
	if got.Frontmatter["type"] != "rule" {
		t.Errorf("type = %v, want rule", got.Frontmatter["type"])
	}
	if got.Frontmatter["name"] != "Flanking" {
		t.Errorf("name = %v, want Flanking", got.Frontmatter["name"])
	}
}

func TestRuleParserDerivesIDFromHeading(t *testing.T) {
	p := &RuleParser{}
	sec := &parser.Section{
		Heading:      "Dying and Death",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "rule", "group": "health"},
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got.ItemID != "dying-and-death" {
		t.Errorf("ItemID = %q, want dying-and-death (slug of heading when @id absent)", got.ItemID)
	}
}

func TestRuleParserFlatWhenNoGroup(t *testing.T) {
	p := &RuleParser{}
	sec := &parser.Section{
		Heading:      "Reward",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "rule", "id": "reward"},
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(got.TypePath) != 1 || got.TypePath[0] != "rule" {
		t.Errorf("TypePath = %v, want [rule]", got.TypePath)
	}
}

func TestRuleParserRegistered(t *testing.T) {
	r := NewRegistry()
	if !r.Has("rule") {
		t.Error("registry missing parser for \"rule\"")
	}
}
