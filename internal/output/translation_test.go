package output

import (
	"strings"
	"testing"
)

func TestTranslationTemplate_WithFrontmatter(t *testing.T) {
	input := `---
book: mcdm.heroes.v1
source: MCDM
---

<!-- @type: chapter | @id: classes -->
# Classes

Intro text.`

	result := TranslationTemplate(input, "Draw Steel Heroes.md")

	// Frontmatter should still be first
	if !strings.HasPrefix(result, "---\n") {
		t.Error("frontmatter should remain at start of file")
	}

	// Guide should appear after frontmatter
	fmEnd := strings.Index(result[4:], "\n---\n")
	if fmEnd < 0 {
		t.Fatal("could not find frontmatter end")
	}
	afterFM := result[fmEnd+4+4+1:]
	if !strings.Contains(afterFM, "TRANSLATION TEMPLATE") {
		t.Error("translation guide should appear after frontmatter")
	}

	// Annotations should be preserved
	if !strings.Contains(result, "<!-- @type: chapter | @id: classes -->") {
		t.Error("annotations should be preserved")
	}

	// Content should be preserved
	if !strings.Contains(result, "# Classes") {
		t.Error("headings should be preserved")
	}
	if !strings.Contains(result, "Intro text.") {
		t.Error("body text should be preserved")
	}

	// Filename should appear in the guide
	if !strings.Contains(result, "input/i18n/{locale}/Draw Steel Heroes.md") {
		t.Error("guide should reference the filename")
	}
}

func TestTranslationTemplate_WithoutFrontmatter(t *testing.T) {
	input := `<!-- @type: class | @id: fury -->
## Fury

Some text.`

	result := TranslationTemplate(input, "test.md")

	// Guide should be at the start
	if !strings.HasPrefix(result, "<!-- ===") {
		t.Error("guide should be at start when no frontmatter")
	}

	// Original content should follow
	if !strings.Contains(result, "<!-- @type: class | @id: fury -->") {
		t.Error("annotations should be preserved")
	}
	if !strings.Contains(result, "## Fury") {
		t.Error("content should be preserved")
	}
}

func TestTranslationTemplate_GuideInstructions(t *testing.T) {
	result := TranslationTemplate("# Test", "heroes.md")

	checks := []string{
		"DO NOT translate or modify",
		"HTML comment annotations",
		"YAML frontmatter",
		"SCC codes",
		"steel-etl gen --locale",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("guide should contain %q", check)
		}
	}
}
