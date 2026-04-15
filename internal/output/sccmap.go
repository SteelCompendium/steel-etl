package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// SCCMapGenerator builds a mapping from SCC codes to file paths.
// The output is a JSON file used by the website for URL resolution.
type SCCMapGenerator struct {
	OutputPath string // e.g., "output/scc-to-path.json"
	entries    map[string]sccMapEntry
}

type sccMapEntry struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (g *SCCMapGenerator) Format() string { return "scc-map" }

func (g *SCCMapGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	if g.entries == nil {
		g.entries = make(map[string]sccMapEntry)
	}

	name, _ := parsed.Frontmatter["name"].(string)
	typeName, _ := parsed.Frontmatter["type"].(string)

	g.entries[sccCode] = sccMapEntry{
		Path: SCCToFilePath(sccCode, ".md"),
		Name: name,
		Type: typeName,
	}

	return nil
}

// Finalize writes the accumulated SCC-to-path mapping as JSON.
func (g *SCCMapGenerator) Finalize() error {
	if len(g.entries) == 0 {
		return nil
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(g.entries))
	for k := range g.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		e := g.entries[k]
		ordered = append(ordered, map[string]any{
			"scc":  k,
			"path": e.Path,
			"name": e.Name,
			"type": e.Type,
		})
	}

	data, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal scc-map: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(g.OutputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return os.WriteFile(g.OutputPath, data, 0644)
}
