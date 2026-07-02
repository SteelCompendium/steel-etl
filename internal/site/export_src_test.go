package site

import (
	"strings"
	"testing"
)

func TestAppendSourceTemplate(t *testing.T) {
	carded := []byte("---\nname: X\n---\n\n<article class=\"sc-ability\">card</article>")
	out := string(appendSourceTemplate(carded, "# X\n\n**Melee \"quoted\"** & <b>bold</b>\n"))
	if !strings.Contains(out, `<template class="sc-src" data-fmt="md" data-src="`) {
		t.Fatal("template missing")
	}
	if !strings.Contains(out, "&lt;b&gt;bold&lt;/b&gt;") {
		t.Error("body must be HTML-escaped")
	}
	if !strings.Contains(out, "&amp;") {
		t.Error("ampersand must be escaped")
	}
	if !strings.Contains(out, "&#34;quoted&#34;") {
		t.Error("quotes must be escaped (attribute safety)")
	}
	if strings.Contains(out, "# X\n") {
		t.Error("newlines must be encoded (&#10;) so the tag stays single-line")
	}
	if !strings.Contains(out, "&#10;") {
		t.Error("expected &#10; newline encoding")
	}
	out2 := string(appendSourceTemplate(carded, "see [rules](../rule/x.md) and `code` plus a \\* literal\n"))
	for _, bad := range []string{"[", "`", "\\"} {
		if strings.Contains(strings.SplitN(out2, "<template", 2)[1], bad) {
			t.Errorf("%q must be entity-encoded in data-src (python-markdown link/backtick/escape patterns outrank raw-HTML and mangle the attribute)", bad)
		}
	}
	if !strings.Contains(out2, "&#91;rules&#93;") && !strings.Contains(out2, "&#91;rules]") {
		t.Error("link brackets not encoded")
	}
	if !strings.HasSuffix(strings.TrimRight(out, "\n"), "</template>") {
		t.Error("template must be appended at the end")
	}
	if strings.Index(out, "sc-ability") > strings.Index(out, "sc-src") {
		t.Error("template must come after the card")
	}
}

func TestDropSourceTemplate(t *testing.T) {
	in := "<article class=\"sc-ability\">card</article>\n\n<template class=\"sc-src\" data-fmt=\"md\" data-src=\"# X&#10;body\"></template>\n"
	out := dropSourceTemplate(in)
	if strings.Contains(out, "sc-src") {
		t.Errorf("template not stripped: %q", out)
	}
	if !strings.Contains(out, "sc-ability") {
		t.Errorf("card lost: %q", out)
	}
}
