package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMdPathToURLPath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"top-level page", "Browse/condition/dazed.md", "Browse/condition/dazed/"},
		{"nested page", "Browse/feature/ability/fury/gouge.md", "Browse/feature/ability/fury/gouge/"},
		{"index page", "Browse/feature/index.md", "Browse/feature/"},
		{"site root index", "index.md", ""},
		{"deeply nested index", "Browse/feature/ability/fury/index.md", "Browse/feature/ability/fury/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mdPathToURLPath(c.in)
			if got != c.want {
				t.Errorf("mdPathToURLPath(%q) = %q; want %q", c.in, got, c.want)
			}
		})
	}
}

func TestRelativeFromStub(t *testing.T) {
	cases := []struct {
		name     string
		scc      string
		friendly string
		want     string
	}{
		{
			name:     "single-segment scc",
			scc:      "foo",
			friendly: "Browse/x/",
			want:     "../../Browse/x/",
		},
		{
			name:     "two-segment scc",
			scc:      "a/b",
			friendly: "Browse/x/",
			want:     "../../../Browse/x/",
		},
		{
			name:     "three-segment scc",
			scc:      "mcdm.heroes.v1/feature.ability.fury/aspect-of-the-wild",
			friendly: "Browse/feature/ability/fury/aspect-of-the-wild/",
			want:     "../../../../Browse/feature/ability/fury/aspect-of-the-wild/",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := relativeFromStub(c.scc, c.friendly)
			if got != c.want {
				t.Errorf("relativeFromStub(%q, %q) = %q; want %q", c.scc, c.friendly, got, c.want)
			}
		})
	}
}

func TestRenderStub_ContainsKeyElements(t *testing.T) {
	target := "../../../../Browse/feature/ability/fury/aspect-of-the-wild/"
	html := renderStub(target)

	mustContain := []string{
		`<meta name="robots" content="noindex">`,
		`<link rel="canonical" href="` + target + `">`,
		`<meta http-equiv="refresh" content="0; url=` + target + `">`,
		`location.replace("` + target + `" + location.search + location.hash)`,
	}
	for _, s := range mustContain {
		if !strings.Contains(html, s) {
			t.Errorf("stub missing %q\nfull stub:\n%s", s, html)
		}
	}
}

func TestRenderStub_EscapesHTMLInTarget(t *testing.T) {
	// Targets should be URL-safe in practice, but defensively check the
	// attribute is HTML-escaped (no raw `<`/`&` injection from frontmatter).
	target := `../../Browse/x?q=1&r=2`
	html := renderStub(target)
	if strings.Contains(html, "&r=2") && !strings.Contains(html, "&amp;r=2") {
		t.Errorf("unescaped & in attribute: %s", html)
	}
}

func TestGenerateSCCStubs_BasicPage(t *testing.T) {
	docsDir := t.TempDir()
	mustWritePage(t, docsDir, "Browse/condition/dazed.md", "---\nname: Dazed\ntype: condition\nscc: mcdm.heroes.v1/condition/dazed\n---\n\nbody")

	count, errs := generateSCCStubs(docsDir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if count != 1 {
		t.Fatalf("expected 1 stub, got %d", count)
	}

	stub := filepath.Join(docsDir, "scc", "mcdm.heroes.v1", "condition", "dazed", "index.html")
	body, err := os.ReadFile(stub)
	if err != nil {
		t.Fatalf("stub not found: %v", err)
	}
	// Stub at depth 4 (scc/, mcdm.heroes.v1/, condition/, dazed/) -> 4 dotdots.
	if !strings.Contains(string(body), "../../../../Browse/condition/dazed/") {
		t.Errorf("stub missing expected relative target. body:\n%s", string(body))
	}
}

func TestGenerateSCCStubs_SkipsPagesWithoutSCC(t *testing.T) {
	docsDir := t.TempDir()
	mustWritePage(t, docsDir, "Browse/index.md", "# Browse\n")                      // no frontmatter
	mustWritePage(t, docsDir, "Browse/foo.md", "---\nname: Foo\n---\n\nbody")       // frontmatter, no scc
	mustWritePage(t, docsDir, "Browse/bar.md", "---\nname: Bar\nscc: x/y\n---\nbb") // valid

	count, errs := generateSCCStubs(docsDir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if count != 1 {
		t.Fatalf("expected 1 stub (only bar.md has scc), got %d", count)
	}
	if _, err := os.Stat(filepath.Join(docsDir, "scc", "x", "y", "index.html")); err != nil {
		t.Errorf("expected stub for x/y: %v", err)
	}
}

func TestGenerateSCCStubs_IndexPageResolvesToDirURL(t *testing.T) {
	docsDir := t.TempDir()
	mustWritePage(t, docsDir, "Browse/feature/ability/fury/index.md",
		"---\nname: Fury Abilities\nscc: mcdm.heroes.v1/feature.ability/fury\n---\nbody")

	count, errs := generateSCCStubs(docsDir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if count != 1 {
		t.Fatalf("expected 1 stub, got %d", count)
	}

	stub := filepath.Join(docsDir, "scc", "mcdm.heroes.v1", "feature.ability", "fury", "index.html")
	body, err := os.ReadFile(stub)
	if err != nil {
		t.Fatalf("stub not found: %v", err)
	}
	// index.md => Browse/feature/ability/fury/ (no trailing /index/).
	if !strings.Contains(string(body), "Browse/feature/ability/fury/") {
		t.Errorf("stub target wrong. body:\n%s", string(body))
	}
	if strings.Contains(string(body), "fury/index/") {
		t.Errorf("stub target should not include /index/. body:\n%s", string(body))
	}
}

func TestGenerateSCCStubs_DoesNotDescendIntoOwnDir(t *testing.T) {
	// If a prior build left scc/<old>/index.html lying around with a stray
	// .md file, the generator must not pick it up as a source page.
	docsDir := t.TempDir()
	mustWritePage(t, docsDir, "Browse/keep.md", "---\nname: Keep\nscc: a/b\n---\nx")
	// Pre-populate a bogus markdown file under scc/ to confirm it's ignored.
	mustWritePage(t, docsDir, "scc/old/junk.md", "---\nname: Junk\nscc: old/junk\n---\nx")

	count, errs := generateSCCStubs(docsDir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if count != 1 {
		t.Fatalf("expected 1 stub (only Browse/keep.md), got %d", count)
	}
	// scc/ should have been wiped + repopulated with just the new stub.
	if _, err := os.Stat(filepath.Join(docsDir, "scc", "old", "junk", "index.html")); err == nil {
		t.Error("stale stub under scc/old/junk/ should have been removed")
	}
	if _, err := os.Stat(filepath.Join(docsDir, "scc", "a", "b", "index.html")); err != nil {
		t.Errorf("expected new stub at scc/a/b/index.html: %v", err)
	}
}

func TestGenerateSCCStubs_PreservesNestedSCCStructure(t *testing.T) {
	docsDir := t.TempDir()
	mustWritePage(t, docsDir, "Browse/feature/ability/Kits/battlemind-unmooring.md",
		"---\nname: Battlemind (Unmooring)\nscc: mcdm.heroes.v1/feature.ability.battlemind/unmooring\n---\nbody")

	count, _ := generateSCCStubs(docsDir)
	if count != 1 {
		t.Fatalf("expected 1 stub, got %d", count)
	}

	stub := filepath.Join(docsDir, "scc", "mcdm.heroes.v1", "feature.ability.battlemind", "unmooring", "index.html")
	body, err := os.ReadFile(stub)
	if err != nil {
		t.Fatalf("expected stub at nested path: %v", err)
	}
	want := "../../../../Browse/feature/ability/Kits/battlemind-unmooring/"
	if !strings.Contains(string(body), want) {
		t.Errorf("stub target wrong; want %q in:\n%s", want, string(body))
	}
}

func TestGenerateSCCStubs_DoesNotWriteManifest(t *testing.T) {
	// The friendly→SCC manifest (scc-manifest.js) was retired along with the
	// client-side address-bar rewrite. The generator must no longer emit it.
	docsDir := t.TempDir()
	mustWritePage(t, docsDir, "Browse/condition/dazed.md",
		"---\nname: Dazed\nscc: mcdm.heroes.v1/condition/dazed\n---\nbody")

	count, errs := generateSCCStubs(docsDir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if count != 1 {
		t.Fatalf("expected 1 stub, got %d", count)
	}

	if _, err := os.Stat(filepath.Join(docsDir, "javascripts", "scc-manifest.js")); !os.IsNotExist(err) {
		t.Errorf("scc-manifest.js should not be written; stat err = %v", err)
	}
}

func mustWritePage(t *testing.T, base, rel, content string) {
	t.Helper()
	path := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
