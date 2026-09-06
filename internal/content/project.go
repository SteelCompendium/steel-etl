package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ProjectParser handles @type: project sections (downtime projects).
type ProjectParser struct{}

func (p *ProjectParser) Type() string { return "project" }

func (p *ProjectParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	fm := map[string]any{
		"name": name,
		"type": "project",
	}

	// Read only this section's own fields: a delegating project must not
	// acquire the first child project's prerequisites or goal.
	for key, value := range ProjectFields(section.BodySource) {
		fm[key] = value
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    []string{"project"},
		ItemID:      id,
	}, nil
}

// ProjectFields extracts the shared four-field downtime-project ledger. Values
// retain inline links, matching TreasureParser and the published data contract.
func ProjectFields(body string) map[string]string {
	fields := map[string]string{}
	for label, key := range map[string]string{
		"Item Prerequisite":           "item_prerequisite",
		"Project Source":              "project_source",
		"Project Roll Characteristic": "project_roll_characteristic",
		"Project Goal":                "project_goal",
	} {
		if value := extractField(body, label); value != "" {
			fields[key] = value
		}
	}
	return fields
}
