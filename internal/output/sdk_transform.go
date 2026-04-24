package output

import (
	"strconv"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// TransformToSDKFormat converts a ParsedContent into a map that conforms to
// the data-sdk-npm feature.schema.json for abilities and traits.
// Other content types are passed through with minimal transformation.
func TransformToSDKFormat(sccCode string, parsed *content.ParsedContent) map[string]any {
	fm := parsed.Frontmatter
	contentType, _ := fm["type"].(string)

	switch contentType {
	case "ability":
		return transformAbility(sccCode, parsed)
	case "trait":
		return transformTrait(sccCode, parsed)
	case "kit":
		return transformKit(sccCode, parsed)
	default:
		return transformPassthrough(parsed)
	}
}

// transformAbility produces a feature.schema.json-compliant map for abilities.
func transformAbility(sccCode string, parsed *content.ParsedContent) map[string]any {
	fm := parsed.Frontmatter
	out := make(map[string]any)

	// Required schema fields
	out["type"] = "feature"
	out["feature_type"] = "ability"

	// Top-level fields that stay at top level
	setIfPresent(out, "name", fm, "name")
	setIfPresent(out, "flavor", fm, "flavor")
	setIfPresent(out, "distance", fm, "distance")
	setIfPresent(out, "target", fm, "target")
	setIfPresent(out, "trigger", fm, "trigger")
	setIfPresent(out, "cost", fm, "cost")
	setIfPresent(out, "keywords", fm, "keywords")

	// action_type → usage
	if v, ok := fm["action_type"]; ok {
		out["usage"] = v
	}

	// subtype → ability_type (capitalize)
	if v, ok := fm["subtype"].(string); ok && v != "" {
		out["ability_type"] = capitalizeFirst(v)
	}

	// Build effects array
	effects := buildAbilityEffects(fm)
	out["effects"] = effects

	// Build metadata
	meta := buildAbilityMetadata(sccCode, parsed)
	if len(meta) > 0 {
		out["metadata"] = meta
	}

	return out
}

// transformTrait produces a feature.schema.json-compliant map for traits.
func transformTrait(sccCode string, parsed *content.ParsedContent) map[string]any {
	fm := parsed.Frontmatter
	out := make(map[string]any)

	// Required schema fields
	out["type"] = "feature"
	out["feature_type"] = "trait"

	setIfPresent(out, "name", fm, "name")

	// Body becomes the single effect
	effects := []map[string]any{}
	if parsed.Body != "" {
		effects = append(effects, map[string]any{
			"effect": parsed.Body,
		})
	}
	if len(effects) == 0 {
		// Schema requires minItems: 1
		effects = append(effects, map[string]any{
			"effect": "",
		})
	}
	out["effects"] = effects

	// Build metadata
	meta := buildTraitMetadata(sccCode, parsed)
	if len(meta) > 0 {
		out["metadata"] = meta
	}

	return out
}

// transformPassthrough handles types without an SDK schema (class, kit, etc.).
// Returns the frontmatter as-is with the body as "content".
func transformPassthrough(parsed *content.ParsedContent) map[string]any {
	out := copyFrontmatter(parsed.Frontmatter)
	if parsed.Body != "" {
		out["content"] = parsed.Body
	}
	return out
}

// transformKit produces a kit.schema.json-compliant map.
// Kit fields are copied from frontmatter, and the signature ability (if present
// in Children) is transformed into a nested feature object.
func transformKit(sccCode string, parsed *content.ParsedContent) map[string]any {
	out := copyFrontmatter(parsed.Frontmatter)
	if parsed.Body != "" {
		out["content"] = parsed.Body
	}

	// Embed signature ability as a nested feature object
	if parsed.Children != nil {
		if sigParsed, ok := parsed.Children["signature_ability"]; ok {
			out["signature_ability"] = transformAbility("", sigParsed)
		}
	}

	return out
}

// buildAbilityEffects constructs the effects[] array from ability frontmatter fields.
func buildAbilityEffects(fm map[string]any) []map[string]any {
	var effects []map[string]any

	// Power roll effect
	if char, ok := fm["power_roll_characteristic"].(string); ok && char != "" {
		rollEffect := map[string]any{
			"roll": "Power Roll + " + char,
		}
		if v, ok := fm["tier1"].(string); ok {
			rollEffect["tier1"] = v
		}
		if v, ok := fm["tier2"].(string); ok {
			rollEffect["tier2"] = v
		}
		if v, ok := fm["tier3"].(string); ok {
			rollEffect["tier3"] = v
		}
		effects = append(effects, rollEffect)
	}

	// Effect entry
	if v, ok := fm["effect"].(string); ok && v != "" {
		effects = append(effects, map[string]any{
			"name":   "Effect",
			"effect": v,
		})
	}

	// Spend entry: stored as "cost_text: effect_text"
	if v, ok := fm["spend"].(string); ok && v != "" {
		spendEffect := parseSpendField(v)
		effects = append(effects, spendEffect)
	}

	// If no effects were built, wrap body as a single effect (schema requires minItems: 1)
	if len(effects) == 0 {
		effects = append(effects, map[string]any{
			"effect": "",
		})
	}

	return effects
}

// parseSpendField splits a spend string "cost: effect" into an SDK effect entry.
// Input format: "1 Wrath: You can end one effect..."
// Output: {cost: "Spend 1 Wrath", effect: "You can end one effect..."}
func parseSpendField(s string) map[string]any {
	parts := strings.SplitN(s, ": ", 2)
	if len(parts) == 2 {
		return map[string]any{
			"cost":   "Spend " + strings.TrimSpace(parts[0]),
			"effect": strings.TrimSpace(parts[1]),
		}
	}
	// Fallback: treat entire string as effect
	return map[string]any{
		"effect": s,
	}
}

// buildAbilityMetadata creates the metadata object for an ability.
func buildAbilityMetadata(sccCode string, parsed *content.ParsedContent) map[string]any {
	fm := parsed.Frontmatter
	meta := make(map[string]any)

	// Fields that move into metadata
	setIfPresent(meta, "class", fm, "class")
	setIfPresent(meta, "feature_type", fm, "type") // original type before transform
	setIfPresent(meta, "source", fm, "source")

	// level: convert string to int if possible
	if v, ok := fm["level"].(string); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			meta["level"] = n
		} else {
			meta["level"] = v
		}
	}

	// Duplicate certain fields into metadata for backwards compat
	setIfPresent(meta, "action_type", fm, "action_type")
	setIfPresent(meta, "distance", fm, "distance")
	setIfPresent(meta, "target", fm, "target")
	setIfPresent(meta, "flavor", fm, "flavor")
	setIfPresent(meta, "keywords", fm, "keywords")

	if v, ok := fm["subtype"].(string); ok && v != "" {
		meta["ability_type"] = capitalizeFirst(v)
	}

	if v, ok := fm["name"].(string); ok {
		meta["item_name"] = v
	}

	// Item ID from ParsedContent
	if parsed.ItemID != "" {
		meta["item_id"] = parsed.ItemID
	}

	// SCC code
	if sccCode != "" {
		meta["scc"] = []string{sccCode}
	}

	// Type path (e.g., "feature/ability/fury/level-1")
	if len(parsed.TypePath) > 0 {
		meta["type"] = strings.Join(parsed.TypePath, "/")
	}

	// Raw markdown body
	if parsed.Body != "" {
		meta["content"] = parsed.Body
	}

	return meta
}

// buildTraitMetadata creates the metadata object for a trait.
func buildTraitMetadata(sccCode string, parsed *content.ParsedContent) map[string]any {
	fm := parsed.Frontmatter
	meta := make(map[string]any)

	setIfPresent(meta, "class", fm, "class")
	setIfPresent(meta, "kit", fm, "kit")
	setIfPresent(meta, "feature_type", fm, "type") // original type before transform
	setIfPresent(meta, "source", fm, "source")

	// level: convert string to int if possible
	if v, ok := fm["level"].(string); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			meta["level"] = n
		} else {
			meta["level"] = v
		}
	}

	if v, ok := fm["name"].(string); ok {
		meta["item_name"] = v
	}

	if parsed.ItemID != "" {
		meta["item_id"] = parsed.ItemID
	}

	if sccCode != "" {
		meta["scc"] = []string{sccCode}
	}

	if len(parsed.TypePath) > 0 {
		meta["type"] = strings.Join(parsed.TypePath, "/")
	}

	// action_type for traits is "feature" in legacy format
	meta["action_type"] = "feature"

	return meta
}

// setIfPresent copies a value from src[srcKey] to dst[dstKey] if present.
func setIfPresent(dst map[string]any, dstKey string, src map[string]any, srcKey string) {
	if v, ok := src[srcKey]; ok {
		dst[dstKey] = v
	}
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
