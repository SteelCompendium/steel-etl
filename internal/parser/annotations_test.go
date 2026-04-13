package parser

import (
	"testing"
)

func TestSingleLineAnnotation(t *testing.T) {
	input := `Some text

<!-- @type: class | @id: fury -->
## Fury
`
	anns := ExtractAnnotations(input)
	if len(anns) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(anns))
	}
	a := anns[0]
	if a.Fields["type"] != "class" {
		t.Errorf("expected type=class, got %s", a.Fields["type"])
	}
	if a.Fields["id"] != "fury" {
		t.Errorf("expected id=fury, got %s", a.Fields["id"])
	}
	if a.Line != 3 {
		t.Errorf("expected line=3, got %d", a.Line)
	}
}

func TestMultiLineAnnotation(t *testing.T) {
	input := `<!--
@type: ability
@id: gouge
@cost: 3 Ferocity
-->
#### Gouge
`
	anns := ExtractAnnotations(input)
	if len(anns) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(anns))
	}
	a := anns[0]
	if a.Fields["type"] != "ability" {
		t.Errorf("expected type=ability, got %s", a.Fields["type"])
	}
	if a.Fields["id"] != "gouge" {
		t.Errorf("expected id=gouge, got %s", a.Fields["id"])
	}
	if a.Fields["cost"] != "3 Ferocity" {
		t.Errorf("expected cost=3 Ferocity, got %s", a.Fields["cost"])
	}
}

func TestMultipleAnnotations(t *testing.T) {
	input := `<!-- @type: class | @id: fury -->
## Fury

<!-- @type: feature-group | @level: 1 -->
### 1st-Level Features

<!-- @type: ability | @subtype: signature -->
#### Brutal Slam
`
	anns := ExtractAnnotations(input)
	if len(anns) != 3 {
		t.Fatalf("expected 3 annotations, got %d", len(anns))
	}
	if anns[0].Fields["id"] != "fury" {
		t.Errorf("expected first annotation id=fury, got %s", anns[0].Fields["id"])
	}
	if anns[1].Fields["level"] != "1" {
		t.Errorf("expected second annotation level=1, got %s", anns[1].Fields["level"])
	}
	if anns[2].Fields["subtype"] != "signature" {
		t.Errorf("expected third annotation subtype=signature, got %s", anns[2].Fields["subtype"])
	}
}

func TestNoAnnotations(t *testing.T) {
	input := `# Just a heading

Some content with no annotations.

## Another heading
`
	anns := ExtractAnnotations(input)
	if len(anns) != 0 {
		t.Fatalf("expected 0 annotations, got %d", len(anns))
	}
}

func TestEndMarker(t *testing.T) {
	input := `<!-- @type: ability | @id: brutal-slam -->
#### Brutal Slam
...content...
<!-- @end: brutal-slam -->
`
	anns := ExtractAnnotations(input)
	if len(anns) != 1 {
		t.Fatalf("expected 1 annotation (end markers not counted), got %d", len(anns))
	}

	ends := ExtractEndMarkers(input)
	if len(ends) != 1 {
		t.Fatalf("expected 1 end marker, got %d", len(ends))
	}
	if ends[0].ID != "brutal-slam" {
		t.Errorf("expected end marker id=brutal-slam, got %s", ends[0].ID)
	}
}

func TestAnnotationLineNumbers(t *testing.T) {
	input := `line 1
line 2
<!-- @type: chapter | @id: intro -->
# Introduction
line 5
line 6
<!--
@type: class
@id: fury
-->
## Fury
`
	anns := ExtractAnnotations(input)
	if len(anns) != 2 {
		t.Fatalf("expected 2 annotations, got %d", len(anns))
	}
	// First annotation starts at line 3
	if anns[0].Line != 3 {
		t.Errorf("expected first annotation at line 3, got %d", anns[0].Line)
	}
	// Second annotation starts at line 7 (the <!-- line)
	if anns[1].Line != 7 {
		t.Errorf("expected second annotation at line 7, got %d", anns[1].Line)
	}
}

func TestUnknownKeysPassedThrough(t *testing.T) {
	input := `<!-- @type: ability | @custom-field: some value | @note: important -->
#### Something
`
	anns := ExtractAnnotations(input)
	if len(anns) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(anns))
	}
	if anns[0].Fields["custom-field"] != "some value" {
		t.Errorf("expected custom-field=some value, got %s", anns[0].Fields["custom-field"])
	}
	if anns[0].Fields["note"] != "important" {
		t.Errorf("expected note=important, got %s", anns[0].Fields["note"])
	}
}

func TestNonAnnotationCommentIgnored(t *testing.T) {
	input := `<!-- This is a regular comment -->
## Heading

<!-- Another regular comment with no @ fields -->
### Sub
`
	anns := ExtractAnnotations(input)
	if len(anns) != 0 {
		t.Fatalf("expected 0 annotations, got %d", len(anns))
	}
}

func TestAnnotationEndLine(t *testing.T) {
	input := `<!--
@type: ability
@id: gouge
@cost: 3 Ferocity
-->
#### Gouge
`
	anns := ExtractAnnotations(input)
	if len(anns) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(anns))
	}
	// Multi-line: starts at line 1, ends at line 5 (the --> line)
	if anns[0].EndLine != 5 {
		t.Errorf("expected end line=5, got %d", anns[0].EndLine)
	}
}

func TestSingleLineAnnotationEndLine(t *testing.T) {
	input := `<!-- @type: class | @id: fury -->
## Fury
`
	anns := ExtractAnnotations(input)
	if len(anns) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(anns))
	}
	// Single-line: start and end are the same
	if anns[0].EndLine != 1 {
		t.Errorf("expected end line=1, got %d", anns[0].EndLine)
	}
}

func TestSCCOverrideFields(t *testing.T) {
	input := `<!--
@type: ability
@scc: mcdm.heroes.v1/abilities.fury/reactive-strike
@scc-alias: mcdm.heroes.v1/abilities.common/reactive-strike
-->
#### Reactive Strike
`
	anns := ExtractAnnotations(input)
	if len(anns) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(anns))
	}
	if anns[0].Fields["scc"] != "mcdm.heroes.v1/abilities.fury/reactive-strike" {
		t.Errorf("expected scc override, got %s", anns[0].Fields["scc"])
	}
	if anns[0].Fields["scc-alias"] != "mcdm.heroes.v1/abilities.common/reactive-strike" {
		t.Errorf("expected scc-alias, got %s", anns[0].Fields["scc-alias"])
	}
}

func TestHTMLCommentEndMarkerFormat(t *testing.T) {
	input := `<!-- @type: ability | @id: slam -->
#### Slam
content
<!-- @end: slam -->
more content
<!-- @end: other-id -->
`
	ends := ExtractEndMarkers(input)
	if len(ends) != 2 {
		t.Fatalf("expected 2 end markers, got %d", len(ends))
	}
	if ends[0].ID != "slam" {
		t.Errorf("expected first end id=slam, got %s", ends[0].ID)
	}
	if ends[0].Line != 4 {
		t.Errorf("expected first end at line 4, got %d", ends[0].Line)
	}
	if ends[1].ID != "other-id" {
		t.Errorf("expected second end id=other-id, got %s", ends[1].ID)
	}
}
