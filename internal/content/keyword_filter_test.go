package content

import (
	"reflect"
	"testing"
)

// A lone dash in a keyword / list cell is a book convention for "none" and must
// never survive as a list item (it otherwise renders as a stray "-" chip).
func TestParseKeywordsDropsDashPlaceholder(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"-", nil},
		{"—", nil},
		{"–", nil},
		{" - ", nil},
		{"Melee, Strike, Weapon", []string{"Melee", "Strike", "Weapon"}},
		{"Magic, -", []string{"Magic"}},
	}
	for _, c := range cases {
		if got := parseKeywords(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseKeywords(%q) = %#v, want %#v", c.in, got, c.want)
		}
	}
}

func TestSplitCommaListDropsDashPlaceholder(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"-", nil},
		{"—", nil},
		{"Fire, Cold", []string{"Fire", "Cold"}},
		{"Fire-2", []string{"Fire-2"}}, // an internal hyphen is not a placeholder
	}
	for _, c := range cases {
		if got := splitCommaList(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitCommaList(%q) = %#v, want %#v", c.in, got, c.want)
		}
	}
}
