package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

// LinkedGenerator writes markdown files with scc: links resolved to relative paths.
type LinkedGenerator struct {
	BaseDir  string // e.g., "data-rules/en/md-linked"
	Resolver *scc.Resolver
	LinkMode scc.LinkMode
}

func (g *LinkedGenerator) Format() string  { return "md-linked" }
func (g *LinkedGenerator) CleanDir() string { return g.BaseDir }

func (g *LinkedGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	// Reading pages use the full book-order subtree render when available,
	// falling back to the structured Body for sections without one.
	bodySource := parsed.PageBody
	if bodySource == "" {
		bodySource = parsed.Body
	}

	// Create a copy with resolved links in the body
	resolved := &content.ParsedContent{
		Frontmatter: parsed.Frontmatter,
		Body:        g.Resolver.ResolveLinks(bodySource, sccCode, g.LinkMode),
		TypePath:    parsed.TypePath,
		ItemID:      parsed.ItemID,
	}

	relPath := SCCToFilePath(sccCode, ".md")
	fullPath := filepath.Join(g.BaseDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	out, err := BuildMarkdownFile(resolved)
	if err != nil {
		return fmt.Errorf("build linked markdown for %s: %w", sccCode, err)
	}

	return os.WriteFile(fullPath, []byte(out), 0644)
}
