package parser

import (
	"os"
	"testing"
)

func TestSmokeParseRealHeroesDocument(t *testing.T) {
	path := "../../input/heroes/Draw Steel Heroes.md"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("skipping smoke test: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	// Verify frontmatter
	if doc.Frontmatter["book"] != "mcdm.heroes.v1" {
		t.Errorf("expected book=mcdm.heroes.v1, got %v", doc.Frontmatter["book"])
	}

	// Count all sections
	var totalSections int
	var annotatedSections int
	var sectionsByType = make(map[string]int)

	var walk func(sections []*Section)
	walk = func(sections []*Section) {
		for _, s := range sections {
			totalSections++
			if s.Annotation != nil {
				annotatedSections++
				sectionsByType[s.Type()]++
			}
			walk(s.Children)
		}
	}
	walk(doc.Sections)

	t.Logf("Total sections: %d", totalSections)
	t.Logf("Annotated sections: %d", annotatedSections)
	for typ, count := range sectionsByType {
		t.Logf("  %s: %d", typ, count)
	}

	// 1523 annotations total, but 25 abilities lack annotations in the source
	// (annotation script gap). Parser correctly associates 1498.
	if annotatedSections < 1490 {
		t.Errorf("expected ~1498 annotated sections, got %d", annotatedSections)
	}

	// Spot-check: 9 classes
	if sectionsByType["class"] != 9 {
		t.Errorf("expected 9 class sections, got %d", sectionsByType["class"])
	}

	// Spot-check: Fury should exist
	var furyFound bool
	walk2 := func(sections []*Section) {}
	walk2 = func(sections []*Section) {
		for _, s := range sections {
			if s.Type() == "class" && s.ID() == "fury" {
				furyFound = true
				if s.BodySource == "" {
					t.Error("Fury has no body content")
				}
			}
			walk2(s.Children)
		}
	}
	walk2(doc.Sections)

	if !furyFound {
		t.Error("Fury class not found in parsed document")
	}
}
