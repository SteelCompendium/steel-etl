package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestFirstFlavorParagraph(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			"plain prose first paragraph",
			"You were born to the road, never staying in one place.\n\n**Skills:** Nature",
			"You were born to the road, never staying in one place.",
		},
		{
			"italic treasure descriptor",
			"*A worn leather bag that holds far more than its size suggests.*\n\n**Keywords:** Magic",
			"A worn leather bag that holds far more than its size suggests.",
		},
		{
			"skips heading then returns prose",
			"#### Flavor\n\nAn ancient order of knights.",
			"An ancient order of knights.",
		},
		{
			"skips bold stat line",
			"**Level:** 3\n\nThis blade hums with power.",
			"This blade hums with power.",
		},
		{
			"strips links and emphasis",
			"You can become [frightened](rule.combat/frightened.md) by **nothing**.",
			"You can become frightened by nothing.",
		},
		{
			"skips blockquote, table, list, rule",
			"---\n\n> a quote\n\n| a | b |\n\n- item\n\nReal flavor here.",
			"Real flavor here.",
		},
		{"empty body", "", ""},
		{"no prose at all", "#### Heading\n\n**Level:** 1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstFlavorParagraph(tt.body); got != tt.want {
				t.Errorf("firstFlavorParagraph() = %q, want %q", got, tt.want)
			}
		})
	}
}

// newLeafSection builds a minimal annotated leaf section whose FullBodySource()
// returns body, for driving a parser in isolation.
func newLeafSection(heading, body string, ann map[string]string) *parser.Section {
	return &parser.Section{
		Heading:      heading,
		HeadingLevel: 4,
		Annotation:   ann,
		BodySource:   body,
	}
}

func TestParsers_EmitFlavor(t *testing.T) {
	body := "An ancient bloodline of stone-skinned giants.\n\n**Signature Trait:** Mighty\n"
	want := "An ancient bloodline of stone-skinned giants."
	tests := []struct {
		name   string
		parser interface {
			Parse(*context.ContextStack, *parser.Section) (*ParsedContent, error)
		}
		heading string
	}{
		{"ancestry", &AncestryParser{}, "Hakaan"},
		{"culture", &CultureParser{}, "Nomadic"},
		{"title", &TitleParser{}, "Demonslayer"},
		{"perk", &PerkParser{}, "Alert"},
		{"class", &ClassParser{}, "Fury"},
		{"kit", &KitParser{}, "Panther"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec := newLeafSection(tt.heading, body, nil)
			got, err := tt.parser.Parse(context.NewContextStack(nil), sec)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got.Frontmatter["flavor"] != want {
				t.Errorf("flavor = %v, want %q", got.Frontmatter["flavor"], want)
			}
		})
	}
}

func TestCareerParser_FlavorStripsPrompt(t *testing.T) {
	body := "You worked as a spy for a powerful noble. In defining your career, think about the following questions:\n\n**Renown:** 1\n"
	sec := newLeafSection("Spy", body, nil)
	got, err := (&CareerParser{}).Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Frontmatter["flavor"] != "You worked as a spy for a powerful noble." {
		t.Errorf("flavor = %v, want stripped lead-in", got.Frontmatter["flavor"])
	}
}

func TestComplicationParser_FlavorSkipsBenefit(t *testing.T) {
	body := "A debt you can never seem to repay.\n\n**Benefit:** You know lenders.\n\n**Drawback:** They know you.\n"
	sec := newLeafSection("Crushing Debt", body, nil)
	got, err := (&ComplicationParser{}).Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Frontmatter["flavor"] != "A debt you can never seem to repay." {
		t.Errorf("flavor = %v, want flavor above benefit/drawback", got.Frontmatter["flavor"])
	}
}

func TestTreasureParser_FlavorAndProjectFields(t *testing.T) {
	body := "" +
		"*A bag that holds far more than its size suggests.*\n\n" +
		"**Keywords:** Magic\n\n" +
		"**Project Goal:** 45\n\n" +
		"**Project Roll Characteristic:** Reason\n"
	sec := newLeafSection("Bag of Holding", body, map[string]string{"type": "treasure"})
	got, err := (&TreasureParser{}).Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Frontmatter["flavor"] != "A bag that holds far more than its size suggests." {
		t.Errorf("flavor = %v", got.Frontmatter["flavor"])
	}
	if got.Frontmatter["project_goal"] != "45" {
		t.Errorf("project_goal = %v, want 45", got.Frontmatter["project_goal"])
	}
	if got.Frontmatter["project_roll_characteristic"] != "Reason" {
		t.Errorf("project_roll_characteristic = %v, want Reason", got.Frontmatter["project_roll_characteristic"])
	}
}
