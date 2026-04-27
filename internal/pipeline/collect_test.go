package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectSCCCodes_Basic(t *testing.T) {
	// Create a temp input file with annotated markdown
	dir := t.TempDir()
	input := filepath.Join(dir, "test.md")
	content := `---
book: test.book.v1
---

<!-- @type: chapter -->
# Introduction

Some intro text.

<!-- @type: class -->
## Fury

Fury overview.

<!-- @type: ability | @class: fury | @level: 1 -->
### Gouge

A powerful strike.
`
	if err := os.WriteFile(input, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Book:   "test.book.v1",
		Input:  input,
		Locale: "en",
		Output: OutputConfig{
			Formats: []string{"md"},
		},
	}

	result, err := CollectSCCCodes(cfg, input)
	if err != nil {
		t.Fatalf("CollectSCCCodes: %v", err)
	}

	if len(result.Codes) == 0 {
		t.Fatal("expected at least one SCC code")
	}

	// Check that we got the expected codes
	found := make(map[string]bool)
	for _, code := range result.Codes {
		found[code] = true
	}

	if !found["test.book.v1/chapter/introduction"] {
		t.Errorf("expected chapter/introduction, got codes: %v", result.Codes)
	}
	if !found["test.book.v1/class/fury"] {
		t.Errorf("expected class/fury, got codes: %v", result.Codes)
	}

	if len(result.Duplicates) != 0 {
		t.Errorf("expected no duplicates, got: %v", result.Duplicates)
	}
}

func TestCollectSCCCodes_Duplicates(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "test.md")
	// Two chapters with the same heading will produce duplicate SCCs
	content := `---
book: test.book.v1
---

<!-- @type: chapter -->
# Introduction

First intro.

<!-- @type: chapter -->
# Introduction

Duplicate intro.
`
	if err := os.WriteFile(input, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Book:   "test.book.v1",
		Input:  input,
		Locale: "en",
		Output: OutputConfig{
			Formats: []string{"md"},
		},
	}

	result, err := CollectSCCCodes(cfg, input)
	if err != nil {
		t.Fatalf("CollectSCCCodes: %v", err)
	}

	if len(result.Duplicates) == 0 {
		t.Error("expected duplicates for repeated chapter heading")
	}
}

func TestCollectSCCCodes_EmptyInput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "empty.md")
	content := `---
book: test.book.v1
---

No annotated sections here.
`
	if err := os.WriteFile(input, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Book:   "test.book.v1",
		Input:  input,
		Locale: "en",
		Output: OutputConfig{
			Formats: []string{"md"},
		},
	}

	result, err := CollectSCCCodes(cfg, input)
	if err != nil {
		t.Fatalf("CollectSCCCodes: %v", err)
	}

	if len(result.Codes) != 0 {
		t.Errorf("expected no codes for unannotated input, got: %v", result.Codes)
	}
}
