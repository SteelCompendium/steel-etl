package site

import (
	"strings"
	"testing"
)

const fbMalicePage = `---
name: Basilisk Malice
type: featureblock
kind: malice
flavor: At the start of any basilisk's turn, you can spend Malice to activate one of the following features.
features:
    - icon: "🔳"
      name: Walleye
      cost: 7 Malice
      body: A basilisk spews reflective spittle across an adjacent vertical surface.
---

At the start of any basilisk's turn, you can spend Malice to activate one of the following features.

> 🔳 **Walleye (7 Malice)**
>
> A basilisk spews reflective spittle across an adjacent vertical surface.
`

func TestBuildFeatureblockPage_NonFeatureblockPassesThrough(t *testing.T) {
	in := []byte("---\nname: Foo\ntype: ability\n---\n\nbody\n")
	out, ok := buildFeatureblockPage(in)
	if ok {
		t.Fatalf("ability page should not be handled by the featureblock renderer")
	}
	if string(out) != string(in) {
		t.Fatalf("non-featureblock data must be returned unchanged")
	}
}

func TestBuildFeatureblockPage_MaliceWrap(t *testing.T) {
	out, ok := buildFeatureblockPage([]byte(fbMalicePage))
	if !ok {
		t.Fatal("featureblock page should be handled")
	}
	s := string(out)
	// frontmatter preserved
	if !strings.HasPrefix(s, "---\n") || !strings.Contains(s, "type: featureblock") {
		t.Errorf("frontmatter not preserved:\n%s", s)
	}
	for _, want := range []string{
		`class="fb-wrap"`, `data-role="malice"`, `data-kind="malice"`,
		`class="fb md-typeset"`, `class="fb__head"`,
		`class="fb__eyebrow"`, "Malice Features",
		`class="fb__name"`, "Basilisk Malice",
		`class="fb__flavor"`, "spend Malice to activate",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}
