package parser

import (
	"os"
	"testing"
)

func TestParseDocumentFrontmatter(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/simple_class.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	if doc.Frontmatter["book"] != "mcdm.heroes.v1" {
		t.Errorf("expected book=mcdm.heroes.v1, got %v", doc.Frontmatter["book"])
	}
	if doc.Frontmatter["source"] != "MCDM" {
		t.Errorf("expected source=MCDM, got %v", doc.Frontmatter["source"])
	}
	if doc.Frontmatter["title"] != "Draw Steel Heroes" {
		t.Errorf("expected title=Draw Steel Heroes, got %v", doc.Frontmatter["title"])
	}
}

func TestParseDocumentTopLevelSections(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/simple_class.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	// Should have 1 top-level section: # Classes
	if len(doc.Sections) != 1 {
		t.Fatalf("expected 1 top-level section, got %d", len(doc.Sections))
	}

	classes := doc.Sections[0]
	if classes.Heading != "Classes" {
		t.Errorf("expected heading=Classes, got %s", classes.Heading)
	}
	if classes.HeadingLevel != 1 {
		t.Errorf("expected level=1, got %d", classes.HeadingLevel)
	}
	if classes.Type() != "chapter" {
		t.Errorf("expected type=chapter, got %s", classes.Type())
	}
}

func TestParseDocumentClassChildren(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/simple_class.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	classes := doc.Sections[0]

	// Classes should have 2 children: Fury and Shadow
	if len(classes.Children) != 2 {
		t.Fatalf("expected 2 class children, got %d", len(classes.Children))
	}

	fury := classes.Children[0]
	if fury.Heading != "Fury" {
		t.Errorf("expected heading=Fury, got %s", fury.Heading)
	}
	if fury.Type() != "class" {
		t.Errorf("expected type=class, got %s", fury.Type())
	}
	if fury.ID() != "fury" {
		t.Errorf("expected id=fury, got %s", fury.ID())
	}

	shadow := classes.Children[1]
	if shadow.Heading != "Shadow" {
		t.Errorf("expected heading=Shadow, got %s", shadow.Heading)
	}
	if shadow.ID() != "shadow" {
		t.Errorf("expected id=shadow, got %s", shadow.ID())
	}
}

func TestParseDocumentFeatureGroups(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/simple_class.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	fury := doc.Sections[0].Children[0]

	// Fury should have 2 feature-group children: 1st-Level and 2nd-Level
	if len(fury.Children) != 2 {
		t.Fatalf("expected 2 feature-group children under Fury, got %d", len(fury.Children))
	}

	fg1 := fury.Children[0]
	if fg1.Type() != "feature-group" {
		t.Errorf("expected type=feature-group, got %s", fg1.Type())
	}
	if fg1.Annotation["level"] != "1" {
		t.Errorf("expected level=1, got %s", fg1.Annotation["level"])
	}

	fg2 := fury.Children[1]
	if fg2.Annotation["level"] != "2" {
		t.Errorf("expected level=2, got %s", fg2.Annotation["level"])
	}
}

func TestParseDocumentAbilities(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/simple_class.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	fg1 := doc.Sections[0].Children[0].Children[0] // Classes > Fury > 1st-Level

	// 1st-Level should have 3 children: Growing Ferocity, Brutal Slam, Gouge
	if len(fg1.Children) != 3 {
		t.Fatalf("expected 3 children under 1st-Level Features, got %d", len(fg1.Children))
	}

	growingFerocity := fg1.Children[0]
	if growingFerocity.Heading != "Growing Ferocity" {
		t.Errorf("expected Growing Ferocity, got %s", growingFerocity.Heading)
	}
	if growingFerocity.Type() != "feature" {
		t.Errorf("expected type=feature, got %s", growingFerocity.Type())
	}

	brutalSlam := fg1.Children[1]
	if brutalSlam.Heading != "Brutal Slam" {
		t.Errorf("expected Brutal Slam, got %s", brutalSlam.Heading)
	}
	if brutalSlam.Type() != "ability" {
		t.Errorf("expected type=ability, got %s", brutalSlam.Type())
	}
	if brutalSlam.Annotation["subtype"] != "signature" {
		t.Errorf("expected subtype=signature, got %s", brutalSlam.Annotation["subtype"])
	}

	gouge := fg1.Children[2]
	if gouge.Heading != "Gouge" {
		t.Errorf("expected Gouge, got %s", gouge.Heading)
	}
	if gouge.Annotation["cost"] != "3 Ferocity" {
		t.Errorf("expected cost=3 Ferocity, got %s", gouge.Annotation["cost"])
	}
}

func TestParseDocumentBodyContent(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/simple_class.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	fury := doc.Sections[0].Children[0]

	// Fury body should contain overview text but not sub-section content
	if fury.BodySource == "" {
		t.Error("expected Fury to have body content")
	}
	if len(fury.BodySource) < 10 {
		t.Errorf("Fury body too short: %q", fury.BodySource)
	}
}

func TestParseDocumentAllSections(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/simple_class.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	// Count all sections recursively
	var count int
	for _, s := range doc.Sections {
		count += len(s.AllSections())
	}

	// Classes(1) + Fury(1) + 1st-Level(1) + GrowingFerocity(1) + BrutalSlam(1) + Gouge(1)
	// + 2nd-Level(1) + BloodForBlood(1) + Shadow(1) = 9
	if count != 9 {
		t.Errorf("expected 9 total sections, got %d", count)
	}
}

func TestParseDocumentParentLinks(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/simple_class.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	fury := doc.Sections[0].Children[0]
	if fury.Parent != doc.Sections[0] {
		t.Error("expected Fury's parent to be Classes")
	}

	fg1 := fury.Children[0]
	if fg1.Parent != fury {
		t.Error("expected 1st-Level Features parent to be Fury")
	}
}
