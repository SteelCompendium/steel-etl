package output

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"gopkg.in/yaml.v3"
)

// DSEGenerator writes markdown files in Draw Steel Elements (Obsidian plugin) format.
// Abilities and traits use ```ds-feature YAML codeblocks.
// Other types use plain markdown with DSE-specific frontmatter.
type DSEGenerator struct {
	BaseDir string // e.g., "data-rules/en/md-dse"
}

func (g *DSEGenerator) Format() string   { return "md-dse" }
func (g *DSEGenerator) CleanDir() string { return g.BaseDir }

func (g *DSEGenerator) WriteSection(sccCode string, parsed *content.ParsedContent) error {
	if sccCode == "" || parsed == nil {
		return nil
	}

	relPath := SCCToFilePath(sccCode, ".md")
	fullPath := filepath.Join(g.BaseDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	out, err := buildDSEFile(sccCode, parsed)
	if err != nil {
		return fmt.Errorf("build DSE for %s: %w", sccCode, err)
	}

	return os.WriteFile(fullPath, []byte(out), 0644)
}

// buildDSEFile creates the DSE-formatted file content.
func buildDSEFile(sccCode string, parsed *content.ParsedContent) (string, error) {
	fm := buildDSEFrontmatter(sccCode, parsed)

	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(fmBytes)
	sb.WriteString("---\n\n")

	featureType, _ := parsed.Frontmatter["type"].(string)

	// Abilities and traits get ds-feature codeblock format
	switch featureType {
	case "ability", "trait", "feature":
		codeblock, err := buildDSFeatureBlock(parsed)
		if err != nil {
			return "", err
		}
		sb.WriteString(codeblock)
	case "statblock":
		// Statblocks get a ds-sb codeblock (F2 OD-1). The pre-rendered stat
		// table body is dropped — DSE renders the block (the plain-markdown
		// rendering remains available in the md/md-linked formats).
		codeblock, err := buildDSDataBlock("ds-sb", transformStatblock(sccCode, parsed))
		if err != nil {
			return "", err
		}
		sb.WriteString(codeblock)
	case "featureblock", "dynamic-terrain":
		// Featureblocks and dynamic terrain get a ds-fb codeblock (F2 OD-1).
		codeblock, err := buildDSDataBlock("ds-fb", transformFeatureblock(sccCode, parsed))
		if err != nil {
			return "", err
		}
		sb.WriteString(codeblock)
	default:
		// Other types get plain markdown body
		body := parsed.Body
		if featureType == "kit" {
			// The rendered "Kit Bonuses" rows and "Signature Ability" prose+table
			// duplicate the frontmatter *_bonus fields (chrome rows) and the
			// ds-feature fence below, respectively — drop them (F2 OD-1
			// precedent), keeping flavor prose + Equipment (body-only) intact.
			body = stripKitBonusesAndSignatureSections(body)
		}
		if body != "" {
			sb.WriteString(body)
			sb.WriteString("\n")
		}

		// For kits, also render the signature ability as a ds-feature codeblock
		if featureType == "kit" && parsed.Children != nil {
			if sigParsed, ok := parsed.Children["signature_ability"]; ok {
				sb.WriteString("\n")
				codeblock, err := buildDSFeatureBlock(sigParsed)
				if err != nil {
					return "", err
				}
				sb.WriteString(codeblock)
			}
		}
	}

	return sb.String(), nil
}

// buildDSEFrontmatter creates DSE-specific frontmatter with additional metadata fields.
func buildDSEFrontmatter(sccCode string, parsed *content.ParsedContent) map[string]any {
	fm := copyFrontmatter(parsed.Frontmatter)

	// DSE-specific fields
	name, _ := parsed.Frontmatter["name"].(string)
	fm["item_name"] = name
	fm["item_id"] = parsed.ItemID

	// Derive file_basename and file_dpath from SCC path
	relPath := SCCToFilePath(sccCode, "")
	fm["file_basename"] = filepath.Base(relPath)
	fm["file_dpath"] = filepath.Dir(relPath)

	// Feature-specific enrichment
	featureType, _ := parsed.Frontmatter["type"].(string)
	if featureType == "ability" || featureType == "trait" || featureType == "feature" {
		fm["feature_type"] = featureType
		if featureType == "ability" {
			fm["action_type"] = getStringOr(parsed.Frontmatter, "action_type", "Main action")
		} else {
			fm["action_type"] = "feature"
		}
	}

	// Decompose cost into amount + resource
	if cost, ok := parsed.Frontmatter["cost"].(string); ok && cost != "" {
		amount, resource := parseCost(cost)
		if amount != "" {
			fm["cost_amount"] = amount
		}
		if resource != "" {
			fm["cost_resource"] = resource
		}
	}

	// Add source from SCC
	if sccCode != "" {
		parts := strings.Split(sccCode, "/")
		if len(parts) > 0 {
			fm["source"] = parts[0]
		}
	}

	return fm
}

// buildDSFeatureBlock creates the ```ds-feature YAML codeblock.
func buildDSFeatureBlock(parsed *content.ParsedContent) (string, error) {
	feature := map[string]any{
		"type":         "feature",
		"feature_type": getStringOr(parsed.Frontmatter, "type", "ability"),
	}

	if name, ok := parsed.Frontmatter["name"]; ok {
		feature["name"] = name
	}
	if cost, ok := parsed.Frontmatter["cost"]; ok {
		feature["cost"] = cost
	}
	if flavor, ok := parsed.Frontmatter["flavor"]; ok {
		feature["flavor"] = flavor
	}
	if kw, ok := parsed.Frontmatter["keywords"]; ok {
		feature["keywords"] = kw
	}
	if action, ok := parsed.Frontmatter["action_type"]; ok {
		feature["usage"] = action
	}
	if dist, ok := parsed.Frontmatter["distance"]; ok {
		feature["distance"] = dist
	}
	if target, ok := parsed.Frontmatter["target"]; ok {
		feature["target"] = target
	}
	if trigger, ok := parsed.Frontmatter["trigger"]; ok {
		feature["trigger"] = trigger
	}

	// Build metadata (mirrors the frontmatter)
	feature["metadata"] = parsed.Frontmatter

	// Build effects from parsed fields
	effects := buildEffects(parsed)
	if len(effects) > 0 {
		feature["effects"] = effects
	}

	featureBytes, err := yaml.Marshal(feature)
	if err != nil {
		return "", fmt.Errorf("marshal ds-feature: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("```ds-feature\n")
	sb.Write(featureBytes)
	sb.WriteString("```\n")

	return sb.String(), nil
}

// buildDSDataBlock marshals an SDK-shaped object (the same payload the yaml/
// format writes, via transformStatblock/transformFeatureblock) into a fenced
// ```<lang> YAML codeblock — ds-sb for statblocks, ds-fb for featureblocks
// and dynamic terrain (F2 OD-1).
func buildDSDataBlock(lang string, obj map[string]any) (string, error) {
	objBytes, err := yaml.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshal %s: %w", lang, err)
	}

	var sb strings.Builder
	sb.WriteString("```" + lang + "\n")
	sb.Write(objBytes)
	sb.WriteString("```\n")

	return sb.String(), nil
}

// buildEffects extracts the effects list from parsed frontmatter fields.
func buildEffects(parsed *content.ParsedContent) []map[string]any {
	var effects []map[string]any

	// Add main effect if present
	if effect, ok := parsed.Frontmatter["effect"].(string); ok && effect != "" {
		effects = append(effects, map[string]any{"effect": effect})
	}

	// Add power roll if present
	if char, ok := parsed.Frontmatter["power_roll_characteristic"].(string); ok && char != "" {
		pr := map[string]any{"roll": "Power Roll + " + char}
		if t1, ok := parsed.Frontmatter["tier1"].(string); ok {
			pr["tier1"] = t1
		}
		if t2, ok := parsed.Frontmatter["tier2"].(string); ok {
			pr["tier2"] = t2
		}
		if t3, ok := parsed.Frontmatter["tier3"].(string); ok {
			pr["tier3"] = t3
		}
		effects = append(effects, pr)
	}

	// Add spend effect if present
	if spend, ok := parsed.Frontmatter["spend"].(string); ok && spend != "" {
		effects = append(effects, map[string]any{
			"name":   "Spend",
			"effect": spend,
		})
	}

	// If no structured effects were extracted, use the body as a single effect
	if len(effects) == 0 && parsed.Body != "" {
		effects = append(effects, map[string]any{"effect": parsed.Body})
	}

	return effects
}

// reKitBonusesOrSignatureHeading matches the "##### Kit Bonuses" or
// "##### Signature Ability" heading (any level, case-insensitive) that
// starts a rendered duplicate section in a kit's book-source body. In the
// common layout both headings are unannotated and appear back to back
// (Equipment, Kit Bonuses, Signature Ability), so matching "Kit Bonuses"
// truncates both in one cut. Some kits (e.g. the stormwight forms) carry
// their "Kit Bonuses" as a separately annotated feature section that
// FullBodySource already excludes from the body, leaving only "Signature
// Ability" to match — hence matching either heading, whichever comes first.
var reKitBonusesOrSignatureHeading = regexp.MustCompile(`(?mi)^#{1,6}\s+(Kit Bonuses|Signature Ability)\s*$`)

// stripKitBonusesAndSignatureSections drops the rendered "Kit Bonuses" and
// "Signature Ability" sections from a kit's body (md-dse only): the bonus
// rows duplicate the frontmatter *_bonus fields (rendered as chrome rows by
// DSE), and the signature-ability prose+table duplicates the ds-feature
// fence emitted alongside it. Flavor prose and the "Equipment" section
// (body-only — no frontmatter/fence equivalent) are preserved.
func stripKitBonusesAndSignatureSections(body string) string {
	loc := reKitBonusesOrSignatureHeading.FindStringIndex(body)
	if loc == nil {
		return body
	}
	return strings.TrimRight(body[:loc[0]], "\n \t")
}

// parseCost splits "3 Ferocity" into ("3", "Ferocity").
func parseCost(cost string) (string, string) {
	parts := strings.SplitN(cost, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return cost, ""
}

// getStringOr returns a string value from a map, or a default.
func getStringOr(m map[string]any, key, fallback string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return fallback
}
