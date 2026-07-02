package site

import (
	"strings"
	"testing"
)

func TestBuildChapterHead(t *testing.T) {
	in := "---\nname: Rewards\nprinting_book: \"The Beastheart\"\norder: 2\ntype: chapter\n---\n\n# Rewards\n\n---\n\nbody\n"
	out := string(buildChapterHead([]byte(in), nil))
	for _, want := range []string{
		`<div class="sc-cheyebrow">The Beastheart · Chapter 2</div>`,
		"\n\n# Rewards {.sc-chtitle}\n",
		"\n---\n\nbody\n", // body + its hr preserved
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "---\n\n<div") {
		t.Errorf("eyebrow must precede the h1:\n%s", out)
	}

	// order 0 = unnumbered opener: book-only eyebrow
	in0 := strings.Replace(in, "order: 2", "order: 0", 1)
	out0 := string(buildChapterHead([]byte(in0), nil))
	if strings.Contains(out0, "Chapter 0") {
		t.Errorf("order 0 must not render as Chapter 0:\n%s", out0)
	}
	if !strings.Contains(out0, `sc-cheyebrow">The Beastheart</div>`) {
		t.Errorf("order 0 keeps the book eyebrow:\n%s", out0)
	}

	// an h1 already carrying an attr list is left untagged (but keeps eyebrow)
	inAttr := strings.Replace(in, "# Rewards\n", "# Rewards {data-scc=\"x/y/z\"}\n", 1)
	outAttr := string(buildChapterHead([]byte(inAttr), nil))
	if strings.Contains(outAttr, "sc-chtitle") {
		t.Errorf("existing attr list must not be doubled:\n%s", outAttr)
	}

	// non-chapter pages pass through untouched
	other := "---\nname: X\ntype: ability\n---\n\n# X\n\nbody\n"
	if got := string(buildChapterHead([]byte(other), nil)); got != other {
		t.Errorf("non-chapter page must pass through, got:\n%s", got)
	}
}
