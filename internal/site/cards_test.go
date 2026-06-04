package site

import (
	"strings"
	"testing"
)

// inlineMD renders the small subset of inline markdown (links + emphasis) that
// frontmatter card values carry, and rewrites ".md" link targets to the served
// directory-URL form. The card markup nests these values inside non-attributed
// raw-HTML divs where md_in_html will not process them, so they are pre-rendered
// to HTML here. See inlineMD / dirURL in cards.go.
func TestInlineMD(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"link rewritten to dir url", "[Brag](../skill/brag.md)", `<a href="../skill/brag/">Brag</a>`},
		{"emphasis", "*Quick Build:*", "<em>Quick Build:</em>"},
		{
			"mixed",
			"One skill (*Quick Build:* [Brag](../skill/brag.md), [Society](../skill/society.md).)",
			`One skill (<em>Quick Build:</em> <a href="../skill/brag/">Brag</a>, <a href="../skill/society/">Society</a>.)`,
		},
		{"plain", "Two skills from the crafting skill group", "Two skills from the crafting skill group"},
		{"empty", "", ""},
		{"escapes html", "a < b & c", "a &lt; b &amp; c"},
		{"no p wrapper", "hello", "hello"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := inlineMD(tc.in)
			if got != tc.want {
				t.Errorf("inlineMD(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDirURL(t *testing.T) {
	tests := []struct{ in, want string }{
		{"agent.md", "agent/"},
		{"../skill/sneak.md", "../skill/sneak/"},
		{"../skill/sneak.md#quick", "../skill/sneak/#quick"},
		{"index.md", ""},
		{"../skill/index.md", "../skill/"},
		{"already/dir/", "already/dir/"},
		{"https://example.com/x.md", "https://example.com/x.md"},
		{"#anchor", "#anchor"},
		{"mailto:a@b.c", "mailto:a@b.c"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := dirURL(tc.in); got != tc.want {
			t.Errorf("dirURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// careerCard's Skills line must emit a real, resolvable anchor (directory URL),
// not literal markdown — the regression the md_in_html-not-firing bug produced.
func TestCareerCardSkillsRendersLink(t *testing.T) {
	fm := "---\nname: Agent\ntype: career\nskills:\n" +
		"    - One skill from the interpersonal group (*Quick Build:* [Brag](../skill/brag.md).)\n---"
	out := careerCard(fm, "", "agent.md", "Agent")
	if !strings.Contains(out, `<a href="../skill/brag/">Brag</a>`) {
		t.Errorf("expected rendered directory-URL link in Skills line, got:\n%s", out)
	}
	if strings.Contains(out, "[Brag](") {
		t.Errorf("Skills line still contains literal markdown link:\n%s", out)
	}
	if strings.Contains(out, "*Quick Build:*") {
		t.Errorf("Skills line still contains literal emphasis markup:\n%s", out)
	}
	if strings.Contains(out, ".md") {
		t.Errorf("Skills line still contains a dead .md link:\n%s", out)
	}
}

// card() must use the stretched-link structure: a <div> wrapper (never an <a>
// wrapping the whole card, which would nest the inner links) plus one overlay
// anchor pointing at the directory URL of the card's page.
func TestCardStretchedLinkStructure(t *testing.T) {
	out := card("agent.md", "briefcase", "Career", "Agent", "  <div class=\"sc-card__line\">x</div>\n")
	if !strings.HasPrefix(out, `<div class="sc-card sc-fil">`) {
		t.Errorf("card must be a <div> wrapper, got:\n%s", out)
	}
	if strings.Contains(out, `<a class="sc-card sc-fil"`) || strings.Contains(out, `<a class="sc-card sc-card--wide`) {
		t.Errorf("card wrapper must not be an <a> (breaks inner links):\n%s", out)
	}
	if !strings.Contains(out, `<a class="sc-card__link" href="agent/" aria-label="Agent"></a>`) {
		t.Errorf("expected stretched overlay link to directory URL, got:\n%s", out)
	}
}
