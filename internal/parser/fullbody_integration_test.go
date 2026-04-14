package parser

import (
	"os"
	"strings"
	"testing"
)

// Integration tests that parse the feature_with_subheadings fixture and verify
// FullBodySource includes unannotated children from a real parsed document tree.

func TestParsedDocument_FullBodySource_GrowingFerocity(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/feature_with_subheadings.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	// Navigate: Classes > Fury > 1st-Level Features > Growing Ferocity
	fury := doc.Sections[0].Children[0] // Classes > Fury
	fg1 := findChild(fury, "1st-Level Features")
	if fg1 == nil {
		t.Fatal("could not find '1st-Level Features' section")
	}

	gf := findChild(fg1, "Growing Ferocity")
	if gf == nil {
		t.Fatal("could not find 'Growing Ferocity' section")
	}

	// BodySource should only have the intro text (stops at first child heading)
	if strings.Contains(gf.BodySource, "Berserker") {
		t.Error("BodySource should NOT contain child heading content")
	}

	// FullBodySource should include the unannotated table children
	full := gf.FullBodySource()

	if !strings.Contains(full, "You gain certain benefits") {
		t.Error("FullBodySource should contain the section's own body")
	}
	if !strings.Contains(full, "Berserker Growing Ferocity Table") {
		t.Error("FullBodySource should contain unannotated 'Berserker Growing Ferocity Table' heading")
	}
	if !strings.Contains(full, "Knockback bonus equal to Might score.") {
		t.Error("FullBodySource should contain Berserker table data")
	}
	if !strings.Contains(full, "Reaver Growing Ferocity Table") {
		t.Error("FullBodySource should contain unannotated 'Reaver Growing Ferocity Table' heading")
	}
	if !strings.Contains(full, "Agility") {
		t.Error("FullBodySource should contain Reaver table data")
	}
}

func TestParsedDocument_FullBodySource_AspectFeatures(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/feature_with_subheadings.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	fury := doc.Sections[0].Children[0]
	fg1 := findChild(fury, "1st-Level Features")
	if fg1 == nil {
		t.Fatal("could not find '1st-Level Features'")
	}

	af := findChild(fg1, "1st-Level Aspect Features")
	if af == nil {
		t.Fatal("could not find '1st-Level Aspect Features'")
	}

	full := af.FullBodySource()

	if !strings.Contains(full, "1st-Level Aspect Features Table") {
		t.Error("FullBodySource should contain the unannotated table heading")
	}
	if !strings.Contains(full, "Berserker") {
		t.Error("FullBodySource should contain table content")
	}
	if !strings.Contains(full, "Stormwight") {
		t.Error("FullBodySource should contain all table rows")
	}
}

func TestParsedDocument_FullBodySource_ClassWithBasics(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/feature_with_subheadings.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	fury := doc.Sections[0].Children[0]

	// Fury's BodySource stops at the first child heading (Basics)
	if strings.Contains(fury.BodySource, "Starting Characteristics") {
		t.Error("BodySource should NOT contain Basics content (it's under a child heading)")
	}

	// FullBodySource should include unannotated Basics + its nested Advancement Table
	full := fury.FullBodySource()

	if !strings.Contains(full, "Heroic Resource: Ferocity") {
		t.Error("FullBodySource should contain Fury's own body")
	}
	if !strings.Contains(full, "### Basics") {
		t.Error("FullBodySource should contain unannotated Basics heading")
	}
	if !strings.Contains(full, "Starting Characteristics") {
		t.Error("FullBodySource should contain Basics body content")
	}
	if !strings.Contains(full, "Fury Advancement Table") {
		t.Error("FullBodySource should contain nested unannotated Advancement Table")
	}

	// Annotated children (feature-groups) should NOT be included
	if strings.Contains(full, "1st-Level Features") {
		t.Error("FullBodySource should NOT contain annotated feature-group children")
	}
}

func TestParsedDocument_FullBodySource_AbilityNoSubheadings(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/feature_with_subheadings.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	fury := doc.Sections[0].Children[0]
	fg1 := findChild(fury, "1st-Level Features")
	if fg1 == nil {
		t.Fatal("could not find '1st-Level Features'")
	}

	bs := findChild(fg1, "Brutal Slam")
	if bs == nil {
		t.Fatal("could not find 'Brutal Slam'")
	}

	// Brutal Slam has no children, so FullBodySource should equal BodySource
	if bs.FullBodySource() != bs.BodySource {
		t.Error("for sections without children, FullBodySource should equal BodySource")
	}
}

func TestParsedDocument_FullBodySource_DamagingFerocity(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/feature_with_subheadings.md")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	doc, err := ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	fury := doc.Sections[0].Children[0]
	fg2 := findChild(fury, "2nd-Level Features")
	if fg2 == nil {
		t.Fatal("could not find '2nd-Level Features'")
	}

	df := findChild(fg2, "Damaging Ferocity")
	if df == nil {
		t.Fatal("could not find 'Damaging Ferocity'")
	}

	full := df.FullBodySource()

	if !strings.Contains(full, "Damaging Ferocity Additions") {
		t.Error("FullBodySource should contain the unannotated additions table heading")
	}
	if !strings.Contains(full, "2 surges") {
		t.Error("FullBodySource should contain additions table content")
	}
}

// findChild locates a direct child section by heading text.
func findChild(s *Section, heading string) *Section {
	for _, child := range s.Children {
		if child.Heading == heading {
			return child
		}
	}
	return nil
}
