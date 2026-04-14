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

	// Extract stat bonuses from table if present
	statBonuses := extractKitStatBonuses(body)
	if len(statBonuses) > 0 {
		fm["stat_bonuses"] = statBonuses
	}

	// Extract equipment list
	equipment := extractListField(body, "Equipment")
	if len(equipment) > 0 {
		fm["equipment"] = equipment
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

// extractKitStatBonuses parses a kit's stat bonus table.
// Kit tables typically have headers like: Stamina | Speed | Melee Damage | ...
func extractKitStatBonuses(body string) map[string]string {
	lines := strings.Split(body, "\n")
	var tableRows []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && !strings.Contains(trimmed, "---") {
			tableRows = append(tableRows, trimmed)
		}
	}

	if len(tableRows) < 2 {
		return nil
	}

	// First row = headers, second row = values
	headers := splitTableCells(tableRows[0])
	values := splitTableCells(tableRows[1])

	bonuses := make(map[string]string)
	for i, header := range headers {
		if i < len(values) {
			h := strings.TrimSpace(strings.ReplaceAll(header, "**", ""))
			v := strings.TrimSpace(strings.ReplaceAll(values[i], "**", ""))
			if h != "" && v != "" && v != "-" && v != "—" {
				bonuses[Slugify(h)] = v
			}
		}
	}

	return bonuses
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
