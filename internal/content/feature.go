package content

import (
	"strings"

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
		Body:        section.FullBodySource(),
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
		"type": "trait",
	}

	// Look up level from context (set by parent feature-group)
	levelStr := ""
	if level, ok := ctx.Lookup(section.HeadingLevel, "level"); ok {
		fm["level"] = level
		levelStr = level
	}

	if classID != "" {
		fm["class"] = classID
	}
	if kitID != "" {
		fm["kit"] = kitID
	}

	// Build type path: feature.trait.{class}.level-{N}[.{kit}]
	typePath := []string{"feature", "trait"}
	if classID != "" {
		typePath = append(typePath, classID)
	}
	if levelStr != "" {
		typePath = append(typePath, "level-"+levelStr)
	}
	if kitID != "" {
		typePath = append(typePath, kitID)
	}

	result := &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
		TypePath:    typePath,
		ItemID:      id,
	}

	// Find child ability sections (annotated with @type: ability) and embed them.
	// This mirrors the kit pattern where signature abilities are embedded in Children.
	if abilityChild := findAbilityChild(section); abilityChild != nil {
		abilityParser := &AbilityParser{}
		parsed, err := abilityParser.Parse(context.NewContextStack(nil), abilityChild)
		if err == nil {
			result.Children = map[string]*ParsedContent{
				"ability": parsed,
			}
			// Append the ability heading and body so markdown output includes it.
			// FullBodySource() skips annotated children, so without this the
			// markdown ends at "You have the following ability."
			sigHeading := strings.Repeat("#", abilityChild.HeadingLevel) + " " + abilityChild.Heading
			sigBody := abilityChild.FullBodySource()
			if sigBody != "" {
				result.Body = result.Body + "\n\n" + sigHeading + "\n\n" + sigBody
			}
		}
	}

	return result, nil
}

// findAbilityChild searches a section's children (recursively through unannotated
// intermediaries) for a child with @type: ability.
func findAbilityChild(section *parser.Section) *parser.Section {
	for _, child := range section.Children {
		if child.Type() == "ability" {
			return child
		}
		// Recurse through unannotated children
		if child.Type() == "" {
			if found := findAbilityChild(child); found != nil {
				return found
			}
		}
	}
	return nil
}
