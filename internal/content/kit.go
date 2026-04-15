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

	// Extract individual stat bonus fields from table
	extractKitBonusFields(body, fm)

	// Extract equipment text as raw string
	if v := extractField(body, "Equipment"); v != "" {
		fm["equipment_text"] = v
	}

	// Extract kit type from annotation
	if ann := section.Annotation; ann != nil {
		if v, ok := ann["kit-type"]; ok {
			fm["kit_type"] = v
		}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"kit"},
		ItemID:      id,
	}, nil
}

// kitBonusMapping maps slugified table header names to their schema field names.
var kitBonusMapping = map[string]string{
	"stamina":          "stamina_bonus",
	"speed":            "speed_bonus",
	"stability":        "stability_bonus",
	"melee-damage":     "melee_damage_bonus",
	"ranged-damage":    "ranged_damage_bonus",
	"melee-distance":   "melee_distance_bonus",
	"ranged-distance":  "ranged_distance_bonus",
	"disengage":        "disengage_bonus",
	"damage":           "melee_damage_bonus",
	"distance":         "melee_distance_bonus",
}

// extractKitBonusFields parses a kit's stat bonus table and sets individual
// bonus fields on the frontmatter map (e.g., stamina_bonus, speed_bonus).
func extractKitBonusFields(body string, fm map[string]any) {
	lines := strings.Split(body, "\n")
	var tableRows []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && !strings.Contains(trimmed, "---") {
			tableRows = append(tableRows, trimmed)
		}
	}

	if len(tableRows) < 2 {
		return
	}

	// First row = headers, second row = values
	headers := splitTableCells(tableRows[0])
	values := splitTableCells(tableRows[1])

	for i, header := range headers {
		if i >= len(values) {
			break
		}
		h := strings.TrimSpace(strings.ReplaceAll(header, "**", ""))
		v := strings.TrimSpace(strings.ReplaceAll(values[i], "**", ""))
		if h == "" || v == "" || v == "-" || v == "—" {
			continue
		}

		slug := Slugify(h)
		if fieldName, ok := kitBonusMapping[slug]; ok {
			fm[fieldName] = v
		}
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
