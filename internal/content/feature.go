package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// FeatureGroupParser handles @type: feature-group sections.
// Container that provides level context to children.
type FeatureGroupParser struct{}

func (p *FeatureGroupParser) Type() string { return "feature-group" }

func (p *FeatureGroupParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	fm := map[string]any{
		"name": section.Heading,
		"type": "feature-group",
	}

	if level, ok := section.Annotation["level"]; ok {
		fm["level"] = level
	}

	// feature-group is a container -- not classified with its own SCC
	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.BodySource,
	}, nil
}

// FeatureParser handles @type: feature sections.
// Non-ability class features (Growing Ferocity, etc.)
type FeatureParser struct{}

func (p *FeatureParser) Type() string { return "feature" }

func (p *FeatureParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	cleanName := CleanHeading(section.Heading)

	id := section.ID()
	if id == "" {
		id = Slugify(cleanName)
	}

	// Look up parent class from context by walking ancestors
	classID := findAncestorID(ctx, section.HeadingLevel, "class")

	// Look up parent kit from context (for stormwight kits etc.)
	kitID := findAncestorID(ctx, section.HeadingLevel, "kit")

	fm := map[string]any{
		"name": cleanName,
		"type": "feature",
	}

	// Look up level from context (set by parent feature-group)
	if level, ok := ctx.Lookup(section.HeadingLevel, "level"); ok {
		fm["level"] = level
		// Append level to ID for disambiguation (e.g., "perk" → "perk-2")
		id = id + "-" + level
	}

	// Append kit ID for disambiguation (e.g., "kit-bonuses-1" → "kit-bonuses-1-boren")
	if kitID != "" {
		fm["kit"] = kitID
		id = id + "-" + kitID
	}

	typePath := []string{"features"}
	if classID != "" {
		fm["class"] = classID
		typePath = []string{"features", classID}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        section.BodySource,
		TypePath:    typePath,
		ItemID:      id,
	}, nil
}
