package output

import (
	"github.com/SteelCompendium/steel-etl/internal/content"
)

// featureblockScalarKeys are frontmatter fields copied straight into SDK
// output. The parser builds features[]/stats[] at parse time (unlike
// statblocks, which re-parse the body here) — see featureblock.schema.json.
var featureblockScalarKeys = []string{
	"name", "type", "kind", "level", "flavor",
	"role", "terrain_type", "stats", "features",
}

// transformFeatureblock builds an SDK featureblock object (covers both
// `type: featureblock` and `type: dynamic-terrain`).
func transformFeatureblock(sccCode string, parsed *content.ParsedContent) map[string]any {
	out := map[string]any{}
	for _, key := range featureblockScalarKeys {
		if v, ok := parsed.Frontmatter[key]; ok {
			out[key] = v
		}
	}
	if _, ok := out["features"]; !ok {
		out["features"] = []map[string]any{}
	}
	out["metadata"] = map[string]any{"scc": sccCode, "source": extractSource(sccCode)}
	return out
}
