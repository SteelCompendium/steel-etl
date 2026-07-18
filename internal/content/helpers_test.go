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

func TestExtractField_MultipleLabelsOnSameLine(t *testing.T) {
	// Beastheart treasures often run several **Label:** fields together on one
	// source line (e.g. "Precious Collar"), unlike the one-field-per-line
	// convention used elsewhere. extractField must stop each field's value at
	// the next label boundary rather than swallowing the rest of the line.
	body := "**Item Prerequisite:** One collar worn by a royal pet **Project Source:** Texts or lore in Vaslorian **Project Roll Characteristic:** Reason or Intuition"

	tests := []struct {
		field string
		want  string
	}{
		{"Item Prerequisite", "One collar worn by a royal pet"},
		{"Project Source", "Texts or lore in Vaslorian"},
		{"Project Roll Characteristic", "Reason or Intuition"},
	}
	for _, tt := range tests {
		if got := extractField(body, tt.field); got != tt.want {
			t.Errorf("extractField(%q) = %q, want %q", tt.field, got, tt.want)
		}
	}
}

func TestExtractField_MultipleLabels_KeywordsAndPrerequisite(t *testing.T) {
	// "Longclaw": Keywords and Item Prerequisite share a line.
	body := "**Keywords:** Magic, Medium Weapon **Item Prerequisite:** The claws of a dragon"
	if got := extractField(body, "Keywords"); got != "Magic, Medium Weapon" {
		t.Errorf("extractField(Keywords) = %q, want %q", got, "Magic, Medium Weapon")
	}
	if got := extractField(body, "Item Prerequisite"); got != "The claws of a dragon" {
		t.Errorf("extractField(Item Prerequisite) = %q, want %q", got, "The claws of a dragon")
	}
}

func TestExtractField_SingleLabelPerLine_StillWorks(t *testing.T) {
	// Regression guard: the common one-field-per-line shape (used by
	// career/class/culture/kit/perk) must be unaffected by the multi-label fix.
	body := "**Item Prerequisite:** A ruby retrieved from an ancient sky elf ruin\n\n**Project Source:** Texts or lore in Hyrallic"
	if got := extractField(body, "Item Prerequisite"); got != "A ruby retrieved from an ancient sky elf ruin" {
		t.Errorf("extractField(Item Prerequisite) = %q", got)
	}
	if got := extractField(body, "Project Source"); got != "Texts or lore in Hyrallic" {
		t.Errorf("extractField(Project Source) = %q", got)
	}
}

func TestExtractField_LinkSweptLabel_StillWorks(t *testing.T) {
	// Heroes-book labels are SCC link-swept; the label itself must still match
	// by its stripped text ("Item Prerequisite"), one field per line.
	body := "**[Item Prerequisite](scc.v1:mcdm.heroes.v1/rule.downtime/item-prerequisite):** Three vials of black ash from the College of Black Ash"
	want := "Three vials of black ash from the College of Black Ash"
	if got := extractField(body, "Item Prerequisite"); got != want {
		t.Errorf("extractField(Item Prerequisite) = %q, want %q", got, want)
	}
}

func TestExtractField_NoBoldMarkers_StillWorks(t *testing.T) {
	// Plain "Label: value" lines with no bold markers at all (e.g. kit bonus
	// list items) must still resolve via the non-bold fallback path.
	body := "Stamina Bonus: +3 per echelon"
	if got := extractField(body, "Stamina Bonus"); got != "+3 per echelon" {
		t.Errorf("extractField(Stamina Bonus) = %q, want %q", got, "+3 per echelon")
	}
}
