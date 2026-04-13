package content

import "testing"

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
