package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"gopkg.in/yaml.v3"
)

// MarkdownGenerator writes per-section .md files with YAML frontmatter.
type MarkdownGenerator struct {
	BaseDir string // e.g., "data-rules/en/md"
}

func (g *MarkdownGenerator) Format() string { return "md" }

// WriteSection writes a single parsed section as a .md file.
func (g *MarkdownGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	relPath := SCCToFilePath(sccCode, ".md")
	fullPath := filepath.Join(g.BaseDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	out, err := BuildMarkdownFile(parsed)
	if err != nil {
		return fmt.Errorf("build markdown for %s: %w", sccCode, err)
	}

	return os.WriteFile(fullPath, []byte(out), 0644)
}

// SCCToFilePath converts an SCC code to a relative file path with the given extension.
//
// Examples:
//
//	SCCToFilePath("mcdm.heroes.v1/feature.ability.fury.level-1/gouge", ".md")
//	  → "feature/ability/fury/level-1/gouge.md"
//	SCCToFilePath("mcdm.heroes.v1/class/fury", ".json")
//	  → "class/fury.json"
func SCCToFilePath(sccCode string, ext string) string {
	parts := strings.Split(sccCode, "/")
	if len(parts) < 2 {
		return "unknown" + ext
	}

	// Skip the source component (first slash-separated part)
	// Expand dots to path separators in remaining parts
	var pathParts []string
	for _, part := range parts[1:] {
		pathParts = append(pathParts, strings.Split(part, ".")...)
	}

	if len(pathParts) == 0 {
		return "unknown" + ext
	}

	pathParts[len(pathParts)-1] += ext
	return filepath.Join(pathParts...)
}

// BuildMarkdownFile creates file content with YAML frontmatter + body.
func BuildMarkdownFile(parsed *content.ParsedContent) (string, error) {
	fm := copyFrontmatter(parsed.Frontmatter)

	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(fmBytes)
	sb.WriteString("---\n\n")

	if parsed.Body != "" {
		sb.WriteString(parsed.Body)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// copyFrontmatter returns a shallow copy of the frontmatter map.
func copyFrontmatter(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// writeFile is a shared helper for writing content to the correct output path.
func writeFile(baseDir, sccCode, ext string, data []byte) error {
	relPath := SCCToFilePath(sccCode, ext)
	fullPath := filepath.Join(baseDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}
