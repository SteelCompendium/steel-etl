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
	LinkMode scc.LinkMode
}

func (g *DSELinkedGenerator) Format() string   { return "md-dse-linked" }
func (g *DSELinkedGenerator) CleanDir() string { return g.BaseDir }

func (g *DSELinkedGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	// Resolve links in both body and structured frontmatter fields before building DSE output
	resolved := &content.ParsedContent{
		Frontmatter: g.Resolver.ResolveFrontmatter(parsed.Frontmatter, sccCode, g.LinkMode),
		Body:        g.Resolver.ResolveLinks(parsed.Body, sccCode, g.LinkMode),
		TypePath:    parsed.TypePath,
		ItemID:      parsed.ItemID,
		Children:    g.resolveChildren(parsed.Children, sccCode),
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

// resolveChildren returns a copy of a ParsedContent's Children map with scc:
// links resolved in each child's body and frontmatter, relative to the
// parent's SCC code — children (e.g. a kit's signature ability) are embedded
// inline in the parent's own output file, so their links resolve as if they
// lived at the parent's location. Without this, DSELinkedGenerator would
// silently drop Children, and buildDSEFile would never emit the child's
// ds-feature fence (e.g. a kit's signature ability) in md-dse-linked output.
func (g *DSELinkedGenerator) resolveChildren(children map[string]*content.ParsedContent, sccCode string) map[string]*content.ParsedContent {
	if children == nil {
		return nil
	}
	resolved := make(map[string]*content.ParsedContent, len(children))
	for key, child := range children {
		if child == nil {
			resolved[key] = nil
			continue
		}
		resolved[key] = &content.ParsedContent{
			Frontmatter: g.Resolver.ResolveFrontmatter(child.Frontmatter, sccCode, g.LinkMode),
			Body:        g.Resolver.ResolveLinks(child.Body, sccCode, g.LinkMode),
			TypePath:    child.TypePath,
			ItemID:      child.ItemID,
			Children:    g.resolveChildren(child.Children, sccCode),
		}
	}
	return resolved
}
