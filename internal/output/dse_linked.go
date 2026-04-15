package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

// DSELinkedGenerator writes DSE-formatted markdown with scc: links resolved.
type DSELinkedGenerator struct {
	BaseDir  string // e.g., "data-rules/en/md-dse-linked"
	Resolver *scc.Resolver
}

func (g *DSELinkedGenerator) Format() string { return "md-dse-linked" }

func (g *DSELinkedGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	// Resolve links in body before building DSE output
	resolved := &content.ParsedContent{
		Frontmatter: parsed.Frontmatter,
		Body:        g.Resolver.ResolveLinks(parsed.Body, sccCode),
		TypePath:    parsed.TypePath,
		ItemID:      parsed.ItemID,
	}

	relPath := SCCToFilePath(sccCode, ".md")
	fullPath := filepath.Join(g.BaseDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	out, err := buildDSEFile(sccCode, resolved)
	if err != nil {
		return fmt.Errorf("build DSE-linked for %s: %w", sccCode, err)
	}

	return os.WriteFile(fullPath, []byte(out), 0644)
}
