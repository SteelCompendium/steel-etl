package content

import (
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// newSection builds a Section with the given heading, level, annotation, and body.
func newSection(heading string, level int, ann map[string]string, body string) *parser.Section {
	return &parser.Section{
		Heading:      heading,
		HeadingLevel: level,
		Annotation:   ann,
		BodySource:   body,
	}
}

func TestStatblockParser(t *testing.T) {
	body := cursespitterGrid + "\n" + cursespitterFeatures
	sec := newSection("Goblin Cursespitter", 7, map[string]string{"type": "statblock"}, body)

	ctx := context.NewContextStack(nil)
	ctx.Push(2, map[string]string{"category": "goblins"})

	p := &StatblockParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got.ItemID != "goblin-cursespitter" {
		t.Errorf("ItemID: got %q", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "monster/goblins/statblock" {
		t.Errorf("TypePath: got %v, want [monster goblins statblock]", got.TypePath)
	}
	if got.Frontmatter["type"] != "statblock" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
	if got.Frontmatter["level"] != 1 {
		t.Errorf("level: got %v", got.Frontmatter["level"])
	}
	if got.Frontmatter["role"] != "Hexer" || got.Frontmatter["organization"] != "Horde" {
		t.Errorf("role/org: got %v / %v", got.Frontmatter["role"], got.Frontmatter["organization"])
	}
	if got.Frontmatter["ev"] != "3" {
		t.Errorf("ev: got %v", got.Frontmatter["ev"])
	}
	if got.Frontmatter["might"] != -2 || got.Frontmatter["intuition"] != 2 {
		t.Errorf("scores: got might=%v int=%v", got.Frontmatter["might"], got.Frontmatter["intuition"])
	}
	if got.Frontmatter["movement"] != "Climb" {
		t.Errorf("movement: got %v", got.Frontmatter["movement"])
	}
}

func TestMonsterParser(t *testing.T) {
	sec := newSection("Goblins", 2, map[string]string{
		"type": "monster", "category": "goblins",
	}, "Goblins are small and crafty...")

	p := &MonsterParser{}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ItemID != "goblins" {
		t.Errorf("ItemID: got %q", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "monster/goblins" {
		t.Errorf("TypePath: got %v, want [monster goblins]", got.TypePath)
	}
	if got.Frontmatter["type"] != "monster" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
}

func TestFeatureblockParser(t *testing.T) {
	body := "" +
		"At the start of any goblin's turn, you can spend Malice...\n\n" +
		"> ⭐️ **Goblin Mode (3 Malice)**\n>\n> Each goblin gains +2 speed.\n"
	sec := newSection("Goblin Malice (Malice Features)", 9,
		map[string]string{"type": "featureblock"}, body)

	ctx := context.NewContextStack(nil)
	ctx.Push(2, map[string]string{"category": "goblins"})

	p := &FeatureblockParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ItemID != "goblin-malice" {
		t.Errorf("ItemID: got %q (want goblin-malice)", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "monster/goblins" {
		t.Errorf("TypePath: got %v, want [monster goblins]", got.TypePath)
	}
	if got.Frontmatter["type"] != "featureblock" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
}

func TestMonsterGroupParser(t *testing.T) {
	sec := newSection("Environmental Hazards", 3, map[string]string{
		"type": "monster-group", "domain": "dynamic-terrain", "category": "environmental-hazards",
	}, "intro prose")
	p := &MonsterGroupParser{}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.TypePath != nil || got.ItemID != "" {
		t.Errorf("expected no classification, got TypePath=%v ItemID=%q", got.TypePath, got.ItemID)
	}
	if got.Frontmatter["type"] != "monster-group" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
}
