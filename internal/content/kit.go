package content

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// KitParser handles @type: kit sections.
type KitParser struct{}

func (p *KitParser) Type() string { return "kit" }

func (p *KitParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "kit",
	}

	body := section.FullBodySource()

	// Extract individual stat bonus fields from **Field Bonus:** value lines
	// and also from list items (- **Field Bonus:** value or - Field Bonus: value)
	extractKitBonusFields(body, fm)

	// Extract equipment text — look for the paragraph after "##### Equipment" heading
	extractKitEquipmentText(body, fm)

	// Extract kit type from annotation
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["kit-type"]; ok {
			fm["kit_type"] = v
		}
	}

	result := &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"kit"},
		ItemID:      id,
	}

	// Find the signature ability from the section tree children.
	// The ability is annotated with @type: ability | @subtype: signature and
	// may be nested under an unannotated "Signature Ability" heading.
	if sigAbility := findSignatureAbilityChild(section); sigAbility != nil {
		abilityParser := &AbilityParser{}
		parsed, err := abilityParser.Parse(context.NewContextStack(nil), sigAbility)
		if err == nil {
			result.Children = map[string]*ParsedContent{
				"signature_ability": parsed,
			}
		}
	}

	return result, nil
}

// findSignatureAbilityChild searches the section's children (recursively through
// unannotated intermediate sections like "##### Signature Ability") for a child
// with @type: ability and @subtype: signature.
func findSignatureAbilityChild(section *parser.Section) *parser.Section {
	for _, child := range section.Children {
		if child.Type() == "ability" {
			if sub, ok := child.Annotation["subtype"]; ok && sub == "signature" {
				return child
			}
		}
		// Recurse through unannotated children (e.g., "##### Signature Ability" heading)
		if child.Type() == "" {
			if found := findSignatureAbilityChild(child); found != nil {
				return found
			}
		}
	}
	return nil
}

// kitBonusFields maps the field label (as it appears in markdown after stripping
// bold markers and list prefixes) to the schema field name.
var kitBonusFields = map[string]string{
	"Stamina Bonus":         "stamina_bonus",
	"Speed Bonus":           "speed_bonus",
	"Stability Bonus":       "stability_bonus",
	"Melee Damage Bonus":    "melee_damage_bonus",
	"Ranged Damage Bonus":   "ranged_damage_bonus",
	"Melee Distance Bonus":  "melee_distance_bonus",
	"Ranged Distance Bonus": "ranged_distance_bonus",
	"Disengage Bonus":       "disengage_bonus",
}

// extractKitBonusFields parses kit bonus lines in all supported formats:
//   - **Stamina Bonus:** +3 per echelon
//   - **Stamina Bonus:** +3 per echelon  (list item)
//   - Stamina Bonus: +3 per echelon      (list item, no bold)
func extractKitBonusFields(body string, fm map[string]any) {
	for label, field := range kitBonusFields {
		if v := extractField(body, label); v != "" {
			fm[field] = v
		}
	}
}

// extractKitEquipmentText finds the paragraph after a "##### Equipment" heading.
func extractKitEquipmentText(body string, fm map[string]any) {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Match "##### Equipment" or "###### Equipment" headings
		if strings.HasSuffix(trimmed, "Equipment") && strings.HasPrefix(trimmed, "#") {
			// Find the next non-empty line as the equipment text
			for j := i + 1; j < len(lines); j++ {
				text := strings.TrimSpace(lines[j])
				if text != "" {
					// Stop if we hit another heading or bonus field
					if strings.HasPrefix(text, "#") {
						return
					}
					fm["equipment_text"] = text
					return
				}
			}
			return
		}
	}
	// Fallback: try extractField for "Equipment:" pattern
	if v := extractField(body, "Equipment"); v != "" {
		fm["equipment_text"] = v
	}
}

// splitTableCells splits a markdown table row by "|" and returns trimmed cells.
func splitTableCells(row string) []string {
	row = strings.Trim(row, "|")
	parts := strings.Split(row, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}
