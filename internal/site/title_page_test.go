package site

import (
	"strings"
	"testing"
)

// A title leaf page shows its echelon as an "**Echelon:** 1st" line above the
// Prerequisite line (site-only, from the `echelon` frontmatter — the data body
// stays book-faithful; the book conveys echelon via group headers instead).
func TestBuildTitleEchelonPage_InsertsLineBeforePrerequisite(t *testing.T) {
	page := `---
echelon: "2"
name: Faction Member
type: title
---

*You joined a faction.*

**Prerequisite:** You gain membership.

**Effect:** Choose one benefit.
`
	out, ok := buildTitleEchelonPage([]byte(page))
	if !ok {
		t.Fatalf("buildTitleEchelonPage ok=false, want true for echelon title page")
	}
	s := string(out)
	iEch := strings.Index(s, "**Echelon:** 2nd")
	iPre := strings.Index(s, "**Prerequisite:**")
	iFlavor := strings.Index(s, "*You joined a faction.*")
	if iEch < 0 {
		t.Fatalf("expected '**Echelon:** 2nd' line, got:\n%s", s)
	}
	if !(iFlavor < iEch && iEch < iPre) {
		t.Errorf("echelon line misplaced: flavor=%d echelon=%d prereq=%d\n%s", iFlavor, iEch, iPre, s)
	}
	// Blank-line separated so it renders as its own paragraph.
	if !strings.Contains(s, "\n\n**Echelon:** 2nd\n\n") {
		t.Errorf("echelon line not paragraph-separated, got:\n%s", s)
	}
}

// The Prerequisite label may be link-wrapped in some sources; the anchor match
// must tolerate "**[Prerequisite](…)**:" too. A page without any anchor gets the
// line at the top of the body instead so the echelon is never dropped.
func TestBuildTitleEchelonPage_NoPrerequisiteAnchor(t *testing.T) {
	page := "---\nechelon: \"4\"\nname: Monarch\ntype: title\n---\n\n*You rule.*\n\n**Effect:** You reign.\n"
	out, ok := buildTitleEchelonPage([]byte(page))
	if !ok {
		t.Fatalf("ok=false, want true")
	}
	s := string(out)
	iEch := strings.Index(s, "**Echelon:** 4th")
	iEff := strings.Index(s, "**Effect:**")
	if iEch < 0 || iEch > iEff {
		t.Errorf("expected echelon line before effect, got:\n%s", s)
	}
}

// Pages without echelon frontmatter, and non-title pages, pass through untouched.
func TestBuildTitleEchelonPage_PassThrough(t *testing.T) {
	for _, page := range []string{
		"---\nname: Stronghold\ntype: title\n---\n\nBody.\n",
		"---\nechelon: \"1\"\nname: Dart\ntype: treasure\n---\n\n**Prerequisite:** X.\n",
	} {
		if _, ok := buildTitleEchelonPage([]byte(page)); ok {
			t.Errorf("ok=true, want false (pass through) for:\n%s", page)
		}
	}
}
