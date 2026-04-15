package output

import "testing"

func TestStripAnnotations(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "remove frontmatter",
			input: `---
book: mcdm.heroes.v1
source: MCDM
---

# Chapter 1`,
			want: "# Chapter 1",
		},
		{
			name: "remove single-line annotation",
			input: `<!-- @type: class | @id: fury -->
## Fury

Some text.`,
			want: `## Fury

Some text.`,
		},
		{
			name: "remove multi-line annotation",
			input: `<!--
@type: ability
@cost: 3 Ferocity
-->
#### Gouge

Ability text.`,
			want: `#### Gouge

Ability text.`,
		},
		{
			name: "combined frontmatter and annotations",
			input: `---
book: mcdm.heroes.v1
---

<!-- @type: chapter | @id: classes -->
# Classes

Intro text.

<!-- @type: class | @id: fury -->
## Fury

<!--
@type: ability
@cost: 3 Ferocity
-->
#### Gouge

Gouge text.`,
			want: `# Classes

Intro text.

## Fury

#### Gouge

Gouge text.`,
		},
		{
			name:  "no annotations",
			input: "# Title\n\nSome text.\n",
			want:  "# Title\n\nSome text.\n",
		},
		{
			name: "collapse excessive blank lines",
			input: `<!-- @type: class | @id: fury -->
## Fury



<!-- @type: ability -->


#### Gouge`,
			want: `## Fury


#### Gouge`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripAnnotations(tt.input)
			if got != tt.want {
				t.Errorf("StripAnnotations:\ngot:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestStrippedGenerator_Finalize(t *testing.T) {
	dir := t.TempDir()
	outputPath := dir + "/clean/output.md"

	gen := &StrippedGenerator{
		OutputPath: outputPath,
		RawInput: []byte(`---
book: mcdm.heroes.v1
---

<!-- @type: class | @id: fury -->
## Fury

Class text.`),
	}

	// WriteSection should be a no-op
	if err := gen.WriteSection("scc", nil); err != nil {
		t.Fatalf("WriteSection should be no-op: %v", err)
	}

	if err := gen.Finalize(); err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	// Calling Finalize again should be idempotent
	if err := gen.Finalize(); err != nil {
		t.Fatalf("second Finalize failed: %v", err)
	}
}
