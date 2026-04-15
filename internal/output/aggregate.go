package output

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// AggregateGenerator collects all sections and writes:
// 1. Per-section files (same as markdown) to the unified output directory
// 2. An index file listing all content by type
type AggregateGenerator struct {
	BaseDir  string // e.g., "data-unified/en/md"
	sections []aggregateEntry
}

type aggregateEntry struct {
	sccCode string
	name    string
	typeName string
}

func (g *AggregateGenerator) Format() string { return "aggregate" }

func (g *AggregateGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	// Write the section file to the aggregate directory
	relPath := SCCToFilePath(sccCode, ".md")
	fullPath := filepath.Join(g.BaseDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	out, err := BuildMarkdownFile(parsed)
	if err != nil {
		return fmt.Errorf("build markdown for %s: %w", sccCode, err)
	}

	if err := os.WriteFile(fullPath, []byte(out), 0644); err != nil {
		return err
	}

	// Track for index generation
	name, _ := parsed.Frontmatter["name"].(string)
	typeName, _ := parsed.Frontmatter["type"].(string)
	g.sections = append(g.sections, aggregateEntry{
		sccCode:  sccCode,
		name:     name,
		typeName: typeName,
	})

	return nil
}

// Finalize writes index files: one index per type and a master index.
func (g *AggregateGenerator) Finalize() error {
	if len(g.sections) == 0 {
		return nil
	}

	// Group by type
	byType := make(map[string][]aggregateEntry)
	for _, e := range g.sections {
		byType[e.typeName] = append(byType[e.typeName], e)
	}

	// Write per-type index files
	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)

	for _, typeName := range types {
		entries := byType[typeName]
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].name < entries[j].name
		})

		indexPath := filepath.Join(g.BaseDir, "_index", typeName+".md")
		if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
			return fmt.Errorf("create index directory: %w", err)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s Index\n\n", titleCase(typeName)))
		sb.WriteString(fmt.Sprintf("Total: %d\n\n", len(entries)))
		for _, e := range entries {
			relPath := SCCToFilePath(e.sccCode, ".md")
			sb.WriteString(fmt.Sprintf("- [%s](../%s)\n", e.name, relPath))
		}

		if err := os.WriteFile(indexPath, []byte(sb.String()), 0644); err != nil {
			return fmt.Errorf("write index %s: %w", typeName, err)
		}
	}

	// Write master index
	masterPath := filepath.Join(g.BaseDir, "_index", "README.md")
	var sb strings.Builder
	sb.WriteString("# Content Index\n\n")
	for _, typeName := range types {
		sb.WriteString(fmt.Sprintf("- [%s](%s.md) (%d items)\n",
			titleCase(typeName), typeName, len(byType[typeName])))
	}
	sb.WriteString(fmt.Sprintf("\nTotal items: %d\n", len(g.sections)))

	return os.WriteFile(masterPath, []byte(sb.String()), 0644)
}

// titleCase capitalizes the first letter of a string.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
