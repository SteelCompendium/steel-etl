package output

import (
	"fmt"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"gopkg.in/yaml.v3"
)

// YAMLGenerator writes per-section .yaml files.
type YAMLGenerator struct {
	BaseDir string // e.g., "data-rules/en/yaml"
}

func (g *YAMLGenerator) Format() string { return "yaml" }

func (g *YAMLGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	out := TransformToSDKFormat(sccCode, parsed)

	data, err := yaml.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal yaml for %s: %w", sccCode, err)
	}

	return writeFile(g.BaseDir, sccCode, ".yaml", data)
}
