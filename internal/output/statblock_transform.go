package output

import (
	"github.com/SteelCompendium/steel-etl/internal/content"
)

// statblockScalarKeys are frontmatter fields copied straight into SDK output.
// Keys are only emitted when present in frontmatter, so fixture-only fields
// (statblock_kind, terrain_type — Summoner book) are absent from normal statblocks.
var statblockScalarKeys = []string{
	"name", "type", "flavor", "level", "role", "organization", "keywords", "ev", "cost",
	"stamina", "speed", "movement", "size", "stability", "free_strike",
	"might", "agility", "reason", "intuition", "presence",
	"immunities", "weaknesses", "with_captain", "skills",
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

	feats := content.ParseStatblockFeatures(parsed.Body)
	// Coded member entities (a beastheart companion's Level-1 abilities/features,
	// SC-390) are embedded in document order after any blockquote features.
	for _, member := range parsed.Members {
		if f := statblockMemberFeature(member); f != nil {
			feats = append(feats, f)
		}
	}
	if len(feats) > 0 {
		out["features"] = feats
	}

	out["metadata"] = map[string]any{"scc": sccCode, "source": extractSource(sccCode)}
	return out
}

// statblockMemberFeature renders a coded member entity (ability/feature) as an
// embedded statblock feature: the same SDK feature object the member's own json
// file carries, with the signature-ability wording statblock features use
// ("Signature Ability", as ParseStatblockFeatures emits for monsters) instead of
// the standalone ability's "Signature".
func statblockMemberFeature(member *content.ParsedContent) map[string]any {
	if member == nil {
		return nil
	}
	code, _ := member.Frontmatter["scc"].(string)
	f := TransformToSDKFormat(code, member)
	if at, _ := f["ability_type"].(string); at == "Signature" {
		f["ability_type"] = "Signature Ability"
	}
	return f
}
