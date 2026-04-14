package parser

import (
	"strings"
	"testing"
)

func TestFullBodySource_NoChildren(t *testing.T) {
	s := &Section{
		Heading:      "Simple Feature",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "This feature does something cool.",
	}

	got := s.FullBodySource()
	if got != "This feature does something cool." {
		t.Errorf("FullBodySource with no children should equal BodySource, got %q", got)
	}
}

func TestFullBodySource_UnannotatedChildrenIncluded(t *testing.T) {
	tableBody := "| Ferocity | Benefit |\n|----------|----------|\n| 2 | Knockback bonus. |"

	s := &Section{
		Heading:      "Growing Ferocity",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "You gain benefits based on ferocity.",
		Children: []*Section{
			{
				Heading:      "Berserker Growing Ferocity Table",
				HeadingLevel: 6,
				BodySource:   tableBody,
				// No annotation — unannotated child
			},
		},
	}

	got := s.FullBodySource()

	if !strings.Contains(got, "You gain benefits based on ferocity.") {
		t.Error("FullBodySource should contain the parent's BodySource")
	}
	if !strings.Contains(got, "###### Berserker Growing Ferocity Table") {
		t.Error("FullBodySource should contain the unannotated child's heading")
	}
	if !strings.Contains(got, "Knockback bonus.") {
		t.Error("FullBodySource should contain the unannotated child's body")
	}
}

func TestFullBodySource_AnnotatedChildrenExcluded(t *testing.T) {
	s := &Section{
		Heading:      "1st-Level Features",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "feature-group", "level": "1"},
		BodySource:   "As a 1st-level fury, you gain the following features.",
		Children: []*Section{
			{
				Heading:      "Growing Ferocity",
				HeadingLevel: 4,
				Annotation:   map[string]string{"type": "feature"},
				BodySource:   "You gain 1d3 ferocity at the start of each turn.",
			},
			{
				Heading:      "Brutal Slam",
				HeadingLevel: 4,
				Annotation:   map[string]string{"type": "ability"},
				BodySource:   "Power roll stuff.",
			},
		},
	}

	got := s.FullBodySource()

	if !strings.Contains(got, "As a 1st-level fury") {
		t.Error("FullBodySource should contain the section's own BodySource")
	}
	if strings.Contains(got, "Growing Ferocity") {
		t.Error("FullBodySource should NOT contain annotated children's content")
	}
	if strings.Contains(got, "Brutal Slam") {
		t.Error("FullBodySource should NOT contain annotated children's content")
	}
}

func TestFullBodySource_MixedChildren(t *testing.T) {
	s := &Section{
		Heading:      "Growing Ferocity",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "You gain benefits based on ferocity.",
		Children: []*Section{
			{
				// Unannotated — should be included
				Heading:      "Berserker Growing Ferocity Table",
				HeadingLevel: 6,
				BodySource:   "| Ferocity | Benefit |\n| 2 | Knockback. |",
			},
			{
				// Annotated — should be excluded
				Heading:      "Some Nested Ability",
				HeadingLevel: 5,
				Annotation:   map[string]string{"type": "ability"},
				BodySource:   "This ability should not appear in parent.",
			},
			{
				// Unannotated — should be included
				Heading:      "Reaver Growing Ferocity Table",
				HeadingLevel: 6,
				BodySource:   "| Ferocity | Benefit |\n| 2 | Slide bonus. |",
			},
		},
	}

	got := s.FullBodySource()

	if !strings.Contains(got, "Berserker Growing Ferocity Table") {
		t.Error("should include first unannotated child")
	}
	if !strings.Contains(got, "Knockback.") {
		t.Error("should include first unannotated child's body")
	}
	if strings.Contains(got, "Some Nested Ability") {
		t.Error("should NOT include annotated child")
	}
	if strings.Contains(got, "should not appear in parent") {
		t.Error("should NOT include annotated child's body")
	}
	if !strings.Contains(got, "Reaver Growing Ferocity Table") {
		t.Error("should include second unannotated child")
	}
	if !strings.Contains(got, "Slide bonus.") {
		t.Error("should include second unannotated child's body")
	}
}

func TestFullBodySource_NestedUnannotatedChildren(t *testing.T) {
	s := &Section{
		Heading:      "Complex Feature",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "Top-level description.",
		Children: []*Section{
			{
				Heading:      "Sub-Section",
				HeadingLevel: 5,
				BodySource:   "Sub-section intro.",
				// No annotation
				Children: []*Section{
					{
						Heading:      "Deep Table",
						HeadingLevel: 6,
						BodySource:   "| A | B |\n| 1 | 2 |",
						// No annotation — nested unannotated
					},
				},
			},
		},
	}

	got := s.FullBodySource()

	if !strings.Contains(got, "Top-level description.") {
		t.Error("should contain own body")
	}
	if !strings.Contains(got, "##### Sub-Section") {
		t.Error("should contain unannotated child heading at correct level")
	}
	if !strings.Contains(got, "Sub-section intro.") {
		t.Error("should contain unannotated child body")
	}
	if !strings.Contains(got, "###### Deep Table") {
		t.Error("should contain nested unannotated grandchild heading")
	}
	if !strings.Contains(got, "| 1 | 2 |") {
		t.Error("should contain nested unannotated grandchild body")
	}
}

func TestFullBodySource_EmptyParentWithUnannotatedChildren(t *testing.T) {
	s := &Section{
		Heading:      "Container",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "",
		Children: []*Section{
			{
				Heading:      "Content Table",
				HeadingLevel: 6,
				BodySource:   "| Col1 | Col2 |",
			},
		},
	}

	got := s.FullBodySource()

	if !strings.Contains(got, "###### Content Table") {
		t.Error("should include unannotated child even when parent body is empty")
	}
	if !strings.Contains(got, "| Col1 | Col2 |") {
		t.Error("should include unannotated child body")
	}
}

func TestFullBodySource_AnnotationWithoutType(t *testing.T) {
	// A child with an annotation but no @type should be included
	// (only children with a non-empty Type() are excluded)
	s := &Section{
		Heading:      "Feature",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "Parent body.",
		Children: []*Section{
			{
				Heading:      "Note Section",
				HeadingLevel: 5,
				Annotation:   map[string]string{"note": "this section is incomplete"},
				BodySource:   "Some noted content.",
			},
		},
	}

	got := s.FullBodySource()

	// Annotation without @type means Type() returns "" — should be included
	if !strings.Contains(got, "Note Section") {
		t.Error("child with annotation but no @type should be included")
	}
	if !strings.Contains(got, "Some noted content.") {
		t.Error("child body with annotation but no @type should be included")
	}
}

func TestFullBodySource_PreservesHeadingLevel(t *testing.T) {
	s := &Section{
		Heading:      "Feature",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "Body.",
		Children: []*Section{
			{
				Heading:      "H5 Child",
				HeadingLevel: 5,
				BodySource:   "H5 body.",
			},
			{
				Heading:      "H6 Child",
				HeadingLevel: 6,
				BodySource:   "H6 body.",
			},
		},
	}

	got := s.FullBodySource()

	if !strings.Contains(got, "##### H5 Child") {
		t.Error("H5 child should be rendered with 5 hash marks")
	}
	if !strings.Contains(got, "###### H6 Child") {
		t.Error("H6 child should be rendered with 6 hash marks")
	}
}
