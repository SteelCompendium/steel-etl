package content

import "testing"

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
