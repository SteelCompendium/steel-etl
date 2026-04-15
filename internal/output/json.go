package output

import (
	"encoding/json"
	"fmt"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// JSONGenerator writes per-section .json files.
type JSONGenerator struct {
	BaseDir string // e.g., "data-rules/en/json"
}

func (g *JSONGenerator) Format() string { return "json" }

func (g *JSONGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	out := TransformToSDKFormat(sccCode, parsed)

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json for %s: %w", sccCode, err)
	}
	data = append(data, '\n')

	return writeFile(g.BaseDir, sccCode, ".json", data)
}
