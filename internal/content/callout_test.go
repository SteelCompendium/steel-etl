package content

import (
	"reflect"
	"strings"
	"testing"
)

func TestIsLooseCalloutComment(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{"loose", "<!-- @type: callout | @owner: loose -->", true},
		{"loose trailing space", "<!-- @type: callout | @owner: loose --> ", true},
		{"loose reordered keys", "<!-- @owner: loose | @type: callout -->", true},
		{"self is not loose", "<!-- @type: callout | @owner: self -->", false},
		{"callout without owner", "<!-- @type: callout -->", false},
		{"unrelated comment", "<!-- @type: feature | @id: x -->", false},
		{"prose mentioning callout", "This callout explains loose treasure rules.", false},
		{"blockquote line", "> **Minions and Treasures**", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isLooseCalloutComment(c.line); got != c.want {
				t.Errorf("isLooseCalloutComment(%q) = %v, want %v", c.line, got, c.want)
			}
		})
	}
}

func TestStripLooseCallouts(t *testing.T) {
	t.Run("removes loose callout at end of body", func(t *testing.T) {
		body := "Para one.\n\nPara two.\n\n<!-- @type: callout | @owner: loose -->\n> **Title**\n>\n> Body of callout.\n> - bullet"
		got := stripLooseCallouts(body)
		want := "Para one.\n\nPara two."
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("removes loose callout in middle, collapses blanks", func(t *testing.T) {
		body := "Para one.\n\n<!-- @type: callout | @owner: loose -->\n> **Title**\n> line\n\nPara two."
		got := stripLooseCallouts(body)
		want := "Para one.\n\nPara two."
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("keeps self callout", func(t *testing.T) {
		body := "Para.\n\n<!-- @type: callout | @owner: self -->\n> **Alt Rule**\n> Use this instead."
		got := stripLooseCallouts(body)
		if !strings.Contains(got, "Alt Rule") {
			t.Errorf("self callout was stripped: %q", got)
		}
	})

	t.Run("keeps untagged blockquote", func(t *testing.T) {
		body := "Para.\n\n> *flavor quote*\n> — Someone"
		got := stripLooseCallouts(body)
		if got != body {
			t.Errorf("untagged blockquote altered: got %q", got)
		}
	})

	t.Run("strips loose, keeps adjacent untagged blockquote", func(t *testing.T) {
		body := "> *flavor*\n\n<!-- @type: callout | @owner: loose -->\n> **Drop me**\n> gone"
		got := stripLooseCallouts(body)
		want := "> *flavor*"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("body with no callout is unchanged", func(t *testing.T) {
		body := "Just some prose.\n\nWith two paragraphs."
		if got := stripLooseCallouts(body); got != body {
			t.Errorf("unchanged body altered: %q", got)
		}
	})
}

func TestScanCallouts(t *testing.T) {
	body := "x\n<!-- @type: callout | @owner: loose -->\n> a\n\ny\n<!-- @type: callout -->\n> b\n\nz\n<!-- @type: callout | @owner: bogus -->\n> c"
	got := ScanCallouts(body)
	want := []CalloutAnnotation{
		{Owner: "loose", HasOwner: true, OwnerKnown: true},
		{Owner: "", HasOwner: false, OwnerKnown: false},
		{Owner: "bogus", HasOwner: true, OwnerKnown: false},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d callouts, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Owner != want[i].Owner || got[i].HasOwner != want[i].HasOwner || got[i].OwnerKnown != want[i].OwnerKnown {
			t.Errorf("callout %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	_ = reflect.DeepEqual // keep import if future use
}
