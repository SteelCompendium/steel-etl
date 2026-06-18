package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Brutal Slam", "brutal-slam"},
		{"Blood for Blood!", "blood-for-blood"},
		{"Growing Ferocity", "growing-ferocity"},
		{"1st-Level Features", "1st-level-features"},
		{"Back Blasphemer!", "back-blasphemer"},
		{"Every Step... Death!", "every-step-death"},
		{"Your Allies Cannot Save You!", "your-allies-cannot-save-you"},
		{"Halt Miscreant!", "halt-miscreant"},
		{"Fury", "fury"},
		{"", ""},
		// Apostrophe handling
		{"Saint's Raiment", "saints-raiment"},
		{"Corruption's Curse", "corruptions-curse"},
		{"Judgment's Hammer", "judgments-hammer"},
		{"God\u2019s Machine", "gods-machine"}, // right single quote
	}

	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCleanHeading(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Alacrity of the Heart (11 Piety)", "Alacrity of the Heart"},
		{"Brutal Slam", "Brutal Slam"},
		{"Font of Wrath (3 Piety)", "Font of Wrath"},
		{"Driving Assault (3 Wrath)", "Driving Assault"},
		{"Growing Ferocity", "Growing Ferocity"},
		{"", ""},
	}

	for _, tt := range tests {
		got := CleanHeading(tt.input)
		if got != tt.want {
			t.Errorf("CleanHeading(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractDomains(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{"standard line", "**Domains:** Creation, Life, Love, Protection\n\nProse.", []string{"Creation", "Life", "Love", "Protection"}},
		{"two domains", "**Domains:** Life, War", []string{"Life", "War"}},
		{"no line", "Just prose, no domains.", nil},
		{"trims spaces", "**Domains:**  Sun ,  Storm ", []string{"Sun", "Storm"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractDomains(c.body)
			if len(got) != len(c.want) {
				t.Fatalf("extractDomains() = %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("extractDomains()[%d] = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestHeadingName(t *testing.T) {
	if got := headingName(&parser.Section{Heading: "Val"}); got != "Val" {
		t.Errorf("headingName plain = %q, want Val", got)
	}
	s := &parser.Section{Heading: "Devil Gods", Annotation: map[string]string{"name": "Lords of Hell"}}
	if got := headingName(s); got != "Lords of Hell" {
		t.Errorf("headingName override = %q, want Lords of Hell", got)
	}
}
