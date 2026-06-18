package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestSaintParser(t *testing.T) {
	p := &SaintParser{}
	if p.Type() != "saint" {
		t.Errorf("Type() = %q, want saint", p.Type())
	}

	section := &parser.Section{
		Heading:    "Llewellyn the Valiant",
		Annotation: map[string]string{"type": "saint", "id": "llewellyn-the-valiant", "patron": "cavall"},
		BodySource: "**Domains:** Life, Protection\n\nA legendary knight.",
	}
	result, err := p.Parse(context.NewContextStack(nil), section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Frontmatter["type"] != "saint" {
		t.Errorf("type = %v, want saint", result.Frontmatter["type"])
	}
	if result.Frontmatter["name"] != "Llewellyn the Valiant" {
		t.Errorf("name = %v", result.Frontmatter["name"])
	}
	if result.Frontmatter["patron"] != "cavall" {
		t.Errorf("patron = %v, want cavall", result.Frontmatter["patron"])
	}
	domains, _ := result.Frontmatter["domains"].([]string)
	if len(domains) != 2 || domains[0] != "Life" {
		t.Errorf("domains = %v, want [Life Protection]", result.Frontmatter["domains"])
	}
	if result.ItemID != "llewellyn-the-valiant" {
		t.Errorf("ItemID = %q", result.ItemID)
	}
	if len(result.TypePath) != 2 || result.TypePath[0] != "religion" || result.TypePath[1] != "saint" {
		t.Errorf("TypePath = %v, want [religion saint]", result.TypePath)
	}
}

func TestSaintParserNameOverride(t *testing.T) {
	section := &parser.Section{
		Heading:    "The Calling of Lady Magnetar",
		Annotation: map[string]string{"type": "saint", "id": "lady-magnetar", "patron": "nebular", "name": "Lady Magnetar"},
		BodySource: "**Domains:** Life, Sun\n\nProse.",
	}
	result, _ := (&SaintParser{}).Parse(context.NewContextStack(nil), section)
	if result.Frontmatter["name"] != "Lady Magnetar" {
		t.Errorf("name = %v, want Lady Magnetar", result.Frontmatter["name"])
	}
}
