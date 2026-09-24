package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	covSize     = "mcdm.heroes.v1/rule.character/size"
	covShifting = "mcdm.heroes.v1/movement/shifting"
	covWalk     = "mcdm.heroes.v1/movement/walk"
	// A code whose section has no Browse page (e.g. site.yaml excludes it):
	// its Read copy must stay searchable.
	covNoBrowse = "mcdm.heroes.v1/feature.trait.ancestry-traits/devil-traits"
)

func TestMarkCoveredHeadings(t *testing.T) {
	covered := map[string]bool{covSize: true, covShifting: true, covWalk: true}
	in := strings.Join([]string{
		"---",
		"name: Combat",
		"# a YAML comment, not a heading",
		"---",
		"",
		"# Combat {.sc-chtitle}",
		"Intro.",
		`#### Size and Space {data-scc="` + covSize + `"}`,
		"Size text.",
		"###### Creature Sizes Table",
		"### Movement",
		"Move freely through an ally's space.",
		"#### Can't Cut Corners",
		`#### Shifting {data-scc="` + covShifting + `"}`,
		"#### Movement Types",
		`##### Walk {data-scc="` + covWalk + `"}`,
		"###### Walk Detail   ",
		"### End of Combat",
		"#### How Combat Ends",
		"```",
		"# code, not a heading",
		"```",
		`#### Devil Traits {data-scc="` + covNoBrowse + `"}`,
	}, "\n")
	want := strings.Join([]string{
		"---",
		"name: Combat",
		"# a YAML comment, not a heading",
		"---",
		"",
		"# Combat {.sc-chtitle}",
		"Intro.",
		`#### Size and Space {data-scc="` + covSize + `" data-search-exclude=""}`,
		"Size text.",
		`###### Creature Sizes Table {data-search-exclude=""}`,
		"### Movement",
		"Move freely through an ally's space.",
		"#### Can't Cut Corners",
		`#### Shifting {data-scc="` + covShifting + `" data-search-exclude=""}`,
		"#### Movement Types",
		`##### Walk {data-scc="` + covWalk + `" data-search-exclude=""}`,
		`###### Walk Detail {data-search-exclude=""}`,
		"### End of Combat",
		"#### How Combat Ends",
		"```",
		"# code, not a heading",
		"```",
		`#### Devil Traits {data-scc="` + covNoBrowse + `"}`,
	}, "\n")

	got, n := markCoveredHeadings(in, covered)
	if got != want {
		t.Errorf("markCoveredHeadings mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
	if n != 5 {
		t.Errorf("marked %d headings, want 5", n)
	}
}

func TestMarkCoveredHeadings_Idempotent(t *testing.T) {
	covered := map[string]bool{covSize: true}
	in := `#### Size and Space {data-scc="` + covSize + `"}` + "\n###### Creature Sizes Table\n"
	once, n1 := markCoveredHeadings(in, covered)
	twice, n2 := markCoveredHeadings(once, covered)
	if n1 != 2 {
		t.Errorf("first pass marked %d, want 2", n1)
	}
	if n2 != 0 || twice != once {
		t.Errorf("second pass must be a no-op: marked %d\n%s", n2, twice)
	}
}

func writeCoverageFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCollectIndexedCodes(t *testing.T) {
	dir := t.TempDir()
	// A container page: its own code in frontmatter, a coded child inline under
	// its own heading, and a heading-looking line inside fenced code (ignored).
	writeCoverageFile(t, dir, "rule/combat/condition.md",
		"---\nname: Conditions\nscc: mcdm.heroes.v1/rule.combat/condition\ntype: rule\n---\n\n# Conditions\n\n"+
			"## Bleeding {data-scc=\"mcdm.heroes.v1/condition/bleeding\"}\n\nText.\n\n"+
			"```\n## Fake {data-scc=\"mcdm.heroes.v1/rule.combat/fake\"}\n```\n")
	// Quoted frontmatter value.
	writeCoverageFile(t, dir, "class/fury.md", "---\nname: Fury\nscc: \"mcdm.heroes.v1/class/fury\"\ntype: class\n---\n\n# Fury\n")
	// Non-markdown files are ignored.
	writeCoverageFile(t, dir, "notes.txt", "scc: mcdm.heroes.v1/rule.combat/ignored\n")

	codes := map[string]bool{}
	if errs := collectIndexedCodes(dir, codes); len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	want := []string{
		"mcdm.heroes.v1/rule.combat/condition",
		"mcdm.heroes.v1/condition/bleeding",
		"mcdm.heroes.v1/class/fury",
	}
	for _, c := range want {
		if !codes[c] {
			t.Errorf("missing code %s", c)
		}
	}
	if len(codes) != len(want) {
		t.Errorf("got %d codes, want %d: %v", len(codes), len(want), codes)
	}
}

func TestCollectIndexedCodes_MissingDir(t *testing.T) {
	codes := map[string]bool{}
	if errs := collectIndexedCodes(filepath.Join(t.TempDir(), "nope"), codes); len(errs) > 0 {
		t.Errorf("missing dir must not error: %v", errs)
	}
	if len(codes) != 0 {
		t.Errorf("missing dir must add no codes: %v", codes)
	}
}

func TestApplyUncoveredOnlySearch(t *testing.T) {
	dir := t.TempDir()
	combat := "---\nname: Combat\ntype: chapter\n---\n\n# Combat\n\n" +
		`#### Size and Space {data-scc="` + covSize + `"}` + "\n\nSize text.\n\n### Movement\n\nMove freely.\n"
	tests := "---\nname: Tests\ntype: chapter\n---\n\n# Tests\n\nNo coded headings.\n"
	writeCoverageFile(t, dir, "Read/heroes/combat.md", combat)
	writeCoverageFile(t, dir, "Read/heroes/tests.md", tests)

	pages, headings, errs := applyUncoveredOnlySearch(dir, "Read", map[string]bool{covSize: true})
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	if pages != 2 || headings != 1 {
		t.Errorf("pages=%d headings=%d, want 2 and 1", pages, headings)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "Read/heroes/combat.md"))
	if !strings.Contains(string(got), `#### Size and Space {data-scc="`+covSize+`" data-search-exclude=""}`) {
		t.Errorf("covered heading not marked:\n%s", got)
	}
	if !strings.Contains(string(got), "### Movement\n") {
		t.Errorf("uncovered heading must stay unmarked:\n%s", got)
	}
	if strings.Contains(string(got), "search:") {
		t.Errorf("uncovered-only pages must not get a search: frontmatter key:\n%s", got)
	}
	untouched, _ := os.ReadFile(filepath.Join(dir, "Read/heroes/tests.md"))
	if string(untouched) != tests {
		t.Errorf("page without covered headings must be byte-identical:\n%s", untouched)
	}

	if p, h, e := applyUncoveredOnlySearch(dir, "Missing", nil); p != 0 || h != 0 || e != nil {
		t.Errorf("missing section: got %d %d %v", p, h, e)
	}
}

func TestWalkErrors(t *testing.T) {
	// Skip if running as root (chmod has no effect)
	if os.Geteuid() == 0 {
		t.Skip("test requires non-root user")
	}

	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	writeCoverageFile(t, subdir, "test.md", "---\nscc: test.code\n---\n# Test\n")

	// Remove read permissions from subdirectory
	if err := os.Chmod(subdir, 0); err != nil {
		t.Fatal(err)
	}
	// Restore permissions in cleanup
	t.Cleanup(func() { os.Chmod(subdir, 0755) })

	// collectIndexedCodes should report walk error
	codes := map[string]bool{}
	errs := collectIndexedCodes(dir, codes)
	if len(errs) == 0 {
		t.Errorf("collectIndexedCodes must report walk error, got none")
	}
	found := false
	for _, err := range errs {
		if strings.Contains(err, subdir) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("collectIndexedCodes errors don't mention subdir: %v", errs)
	}

	// applyUncoveredOnlySearch should report walk error
	p, h, errs := applyUncoveredOnlySearch(dir, "subdir", nil)
	if len(errs) == 0 {
		t.Errorf("applyUncoveredOnlySearch must report walk error, got none")
	}
	found = false
	for _, err := range errs {
		if strings.Contains(err, subdir) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("applyUncoveredOnlySearch errors don't mention subdir: %v", errs)
	}
	if p != 0 || h != 0 {
		t.Errorf("applyUncoveredOnlySearch on unreadable dir: got p=%d h=%d, want 0 0", p, h)
	}
}
