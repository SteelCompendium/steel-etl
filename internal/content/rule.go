package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// RuleParser handles @type: rule sections — rules-glossary terms that get a
// grouped, human-readable SCC code (rule.<group>/<id>) so prose can link to
// the rule that defines them. Mirrors GodParser; the optional @group annotation
// adds the second TypePath segment (rule.<group>); with no @group it is flat
// (rule/<id>). RuleParser never accumulates a parent path, so codes stay flat
// within their group even when the annotated heading is nested in a chapter.
type RuleParser struct{}

func (p *RuleParser) Type() string { return "rule" }

func (p *RuleParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	name := CleanHeading(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(name)
	}

	typePath := []string{"rule"}
	if group, ok := section.Annotation["group"]; ok && group != "" {
		typePath = []string{"rule", group}
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": name,
			"type": "rule",
		},
		Body:     section.FullBodySource(),
		TypePath: typePath,
		ItemID:   id,
	}, nil
}
