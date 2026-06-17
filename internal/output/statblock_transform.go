package output

import (
	"github.com/SteelCompendium/steel-etl/internal/content"
)

// statblockScalarKeys are frontmatter fields copied straight into SDK output.
// Keys are only emitted when present in frontmatter, so fixture-only fields
// (statblock_kind, terrain_type — Summoner book) are absent from normal statblocks.
var statblockScalarKeys = []string{
	"name", "type", "level", "role", "organization", "keywords", "ev", "cost",
	"stamina", "speed", "movement", "size", "stability", "free_strike",
	"might", "agility", "reason", "intuition", "presence",
	"immunities", "weaknesses", "with_captain",
	"statblock_kind", "terrain_type", // fixture statblocks (Summoner book)
}

// statblockDefaults are schema-required fields defaulted when absent in source.
var statblockDefaults = map[string]any{
	"role": "", "organization": "", "keywords": []string{},
	"ev": "", "stamina": "", "level": 0,
	"speed": 0, "size": "", "stability": 0, "free_strike": 0,
	"might": 0, "agility": 0, "reason": 0, "intuition": 0, "presence": 0,
}

// transformStatblock builds an SDK statblock object: scalar stats from the
// parsed frontmatter plus a features[] array parsed from the body blockquotes.
func transformStatblock(sccCode string, parsed *content.ParsedContent) map[string]any {
	out := map[string]any{}
	for _, key := range statblockScalarKeys {
		if v, ok := parsed.Frontmatter[key]; ok {
			out[key] = v
		}
	}
	out["type"] = "statblock"

	for k, dv := range statblockDefaults {
		if _, ok := out[k]; !ok {
			out[k] = dv
		}
	}

	if feats := content.ParseStatblockFeatures(parsed.Body); len(feats) > 0 {
		out["features"] = feats
	}

	out["metadata"] = map[string]any{"scc": sccCode, "source": extractSource(sccCode)}
	return out
}
