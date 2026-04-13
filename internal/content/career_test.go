package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestCareerParser(t *testing.T) {
	p := &CareerParser{}

	if p.Type() != "career" {
		t.Errorf("Type() = %q, want %q", p.Type(), "career")
	}

	section := &parser.Section{
		Heading:      "Artisan",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "career"},
		BodySource: `You are a skilled crafter who creates works of art.

**Skill:** Crafting
**Language:** One language of your choice
**Renown:** 1
**Wealth:** 2
**Project Points:** 10
**Perk:** Handy`,
	}

	ctx := context.NewContextStack(nil)
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	tests := map[string]string{
		"name":           "Artisan",
		"type":           "career",
		"skill":          "Crafting",
		"language":       "One language of your choice",
		"renown":         "1",
		"wealth":         "2",
		"project_points": "10",
		"perk":           "Handy",
	}

	for key, want := range tests {
		got, ok := result.Frontmatter[key]
		if !ok {
			t.Errorf("missing field %q", key)
			continue
		}
		if got != want {
			t.Errorf("%s = %v, want %v", key, got, want)
		}
	}

	if result.ItemID != "artisan" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "artisan")
	}
	if len(result.TypePath) != 1 || result.TypePath[0] != "career" {
		t.Errorf("TypePath = %v, want [career]", result.TypePath)
	}
}

func TestCultureParser(t *testing.T) {
	p := &CultureParser{}

	if p.Type() != "culture" {
		t.Errorf("Type() = %q, want %q", p.Type(), "culture")
	}

	section := &parser.Section{
		Heading:      "Nomadic",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "culture"},
		BodySource: `A culture of traveling peoples.

**Environment:** Wilderness
**Organization:** Communal
**Upbringing:** Practical`,
	}

	ctx := context.NewContextStack(nil)
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["environment"] != "Wilderness" {
		t.Errorf("environment = %v, want Wilderness", result.Frontmatter["environment"])
	}
	if result.Frontmatter["organization"] != "Communal" {
		t.Errorf("organization = %v, want Communal", result.Frontmatter["organization"])
	}
	if result.Frontmatter["upbringing"] != "Practical" {
		t.Errorf("upbringing = %v, want Practical", result.Frontmatter["upbringing"])
	}
}
