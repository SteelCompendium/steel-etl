package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestConditionParser(t *testing.T) {
	p := &ConditionParser{}

	if p.Type() != "condition" {
		t.Errorf("Type() = %q, want %q", p.Type(), "condition")
	}

	section := &parser.Section{
		Heading:      "Dazed",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "condition"},
		BodySource:   "A dazed creature can do only one thing on their turn.",
	}

	ctx := context.NewContextStack(nil)
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Dazed" {
		t.Errorf("name = %v, want Dazed", result.Frontmatter["name"])
	}
	if result.Frontmatter["type"] != "condition" {
		t.Errorf("type = %v, want condition", result.Frontmatter["type"])
	}
	if result.ItemID != "dazed" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "dazed")
	}
	if len(result.TypePath) != 1 || result.TypePath[0] != "condition" {
		t.Errorf("TypePath = %v, want [condition]", result.TypePath)
	}
}

func TestComplicationParser(t *testing.T) {
	p := &ComplicationParser{}

	if p.Type() != "complication" {
		t.Errorf("Type() = %q, want %q", p.Type(), "complication")
	}

	section := &parser.Section{
		Heading:      "Criminal Past",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "complication"},
		BodySource:   "You have a criminal record that follows you.",
	}

	ctx := context.NewContextStack(nil)
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Criminal Past" {
		t.Errorf("name = %v, want Criminal Past", result.Frontmatter["name"])
	}
	if result.ItemID != "criminal-past" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "criminal-past")
	}
}

func TestPerkParser(t *testing.T) {
	p := &PerkParser{}

	if p.Type() != "perk" {
		t.Errorf("Type() = %q, want %q", p.Type(), "perk")
	}

	section := &parser.Section{
		Heading:      "Alertness",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "perk"},
		BodySource:   "**Prerequisite:** None\n\nYou gain an edge on initiative tests.",
	}

	ctx := context.NewContextStack(nil)
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Alertness" {
		t.Errorf("name = %v, want Alertness", result.Frontmatter["name"])
	}
	if result.Frontmatter["type"] != "perk" {
		t.Errorf("type = %v, want perk", result.Frontmatter["type"])
	}
	if result.ItemID != "alertness" {
		t.Errorf("ItemID = %q, want %q", result.ItemID, "alertness")
	}
}

func TestPerkParserWithPrerequisites(t *testing.T) {
	section := &parser.Section{
		Heading:      "Durable",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "perk"},
		BodySource:   "**Prerequisites:** Stamina 12+\n\nYou gain extra stamina.",
	}

	ctx := context.NewContextStack(nil)
	result, err := (&PerkParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Frontmatter["prerequisites"] != "Stamina 12+" {
		t.Errorf("prerequisites = %v, want 'Stamina 12+'", result.Frontmatter["prerequisites"])
	}
}
