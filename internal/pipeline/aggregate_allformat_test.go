package pipeline

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/output"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

// baseDirOf2 extends baseDirOf with the AggregateGenerator type.
func baseDirOf2(g output.Generator) string {
	if a, ok := g.(*output.AggregateGenerator); ok {
		return a.BaseDir
	}
	return baseDirOf(g)
}

func TestBuildAggregateGeneratorsAllFormats(t *testing.T) {
	cfg := &Config{
		Locale:    "en",
		ConfigDir: "/repo/steel-etl",
		Output: OutputConfig{
			Formats:   []string{"md", "json", "yaml"},
			Variants:  VariantsConfig{Linked: true, DSE: true, DSELinked: true},
			Aggregate: AggregateConfig{Enabled: true, OutputDir: "../data/data-unified"},
		},
	}
	gens := buildAggregateGenerators(cfg, scc.NewRegistry(), "en")

	want := []string{"md", "json", "yaml", "md-linked", "md-dse", "md-dse-linked"}
	for _, format := range want {
		suffix := filepath.Join("en", "unified", format)
		found := false
		for _, g := range gens {
			if strings.HasSuffix(baseDirOf2(g), suffix) {
				found = true
			}
		}
		if !found {
			t.Errorf("no aggregate generator for %q (suffix %q)", format, suffix)
		}
	}
}
