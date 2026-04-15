package output

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// StrippedGenerator removes annotations and frontmatter from the raw input,
// producing a clean markdown file suitable for distribution.
type StrippedGenerator struct {
	OutputPath string // e.g., "data-rules-clean/Draw Steel Heroes.md"
	RawInput   []byte // the original annotated input
	written    bool
}

func (g *StrippedGenerator) Format() string { return "stripped" }

// WriteSection is a no-op for StrippedGenerator since it operates on the full input.
// The actual work happens in Finalize.
func (g *StrippedGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	return nil
}

// Finalize strips annotations and frontmatter from the raw input and writes the clean output.
func (g *StrippedGenerator) Finalize() error {
	if g.written {
		return nil
	}
	g.written = true

	clean := StripAnnotations(string(g.RawInput))

	if err := os.MkdirAll(dirOf(g.OutputPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return os.WriteFile(g.OutputPath, []byte(clean), 0644)
}

// dirOf returns the directory portion of a path.
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

// annotation patterns
var (
	// Single-line: <!-- @type: class | @id: fury -->
	singleLineAnnotationRe = regexp.MustCompile(`(?m)^[ \t]*<!--\s*@[^>]+-->\s*\n?`)

	// Multi-line:
	// <!--
	// @type: ability
	// @cost: 3 Ferocity
	// -->
	multiLineAnnotationRe = regexp.MustCompile(`(?ms)^[ \t]*<!--\s*\n(?:[ \t]*@[^\n]*\n)+[ \t]*-->\s*\n?`)

	// YAML frontmatter: --- ... ---
	frontmatterRe = regexp.MustCompile(`(?ms)\A---\n.*?\n---\n?`)
)

// StripAnnotations removes HTML comment annotations and YAML frontmatter from markdown.
func StripAnnotations(input string) string {
	// Remove frontmatter first
	result := frontmatterRe.ReplaceAllString(input, "")

	// Remove multi-line annotations (must be before single-line to avoid partial matches)
	result = multiLineAnnotationRe.ReplaceAllString(result, "")

	// Remove single-line annotations
	result = singleLineAnnotationRe.ReplaceAllString(result, "")

	// Clean up excessive blank lines (more than 2 consecutive)
	result = collapseBlankLines(result)

	return strings.TrimLeft(result, "\n")
}

// collapseBlankLines reduces runs of 3+ blank lines to 2.
func collapseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, line)
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
