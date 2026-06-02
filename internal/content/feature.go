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

	result := &ParsedContent{
		Frontmatter: fm,
		Body:        section.FullBodySource(),
	}

	// Companion species containers (beastheart) are first-class: classify them
	// as feature-group.companion/{species}. Plain feature-groups stay unclassified.
	if companion, ok := section.Annotation["companion"]; ok && companion != "" {
		fm["companion"] = companion
		result.TypePath = []string{"feature-group", "companion"}
		result.ItemID = companion
	}

	return result, nil
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

	// Look up parent ancestry from context
	ancestryID := findAncestorID(ctx, section.HeadingLevel, "ancestry")

	// Companion species (beastheart book) takes precedence over class in the path.
	companionID, _ := ctx.Lookup(section.HeadingLevel, "companion")

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
	if ancestryID != "" {
		fm["ancestry"] = ancestryID
	}
	if companionID != "" {
		fm["companion"] = companionID
	}
	// Subclass is reference metadata only — surfaced in frontmatter, never in the path.
	if v, ok := section.Annotation["subclass"]; ok && v != "" {
		fm["subclass"] = parseSubclass(v)
	}

	// Build type path: feature.trait.{parent}.level-{N}[.{kit}]
	// Companion traits use feature.trait.companion.{species}.level-{N}.
	typePath := []string{"feature", "trait"}
	if companionID != "" {
		typePath = append(typePath, "companion", companionID)
	} else if classID != "" {
		typePath = append(typePath, classID)
	} else if ancestryID != "" {
		typePath = append(typePath, ancestryID)
	} else if kitID == "" {
		groupID := findAncestorID(ctx, section.HeadingLevel, "feature-group")
		typePath = append(typePath, "common")
		if groupID != "" {
			typePath = append(typePath, groupID)
		}
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

	// Embed a single child ability as a structured nested object for the SDK
	// trait schema (which has a singular `ability` field). This only applies to
	// single-ability traits (e.g. "Faithful Friend"). Multi-ability containers
	// (e.g. "Censor Abilities") do NOT get a singular embed; their abilities are
	// rendered on the page via PageBody/RenderSubtree, not the structured Body.
	abilityChildren := collectAbilityChildren(section)
	if len(abilityChildren) == 1 {
		abilityParser := &AbilityParser{}
		parsed, err := abilityParser.Parse(context.NewContextStack(nil), abilityChildren[0])
		if err == nil {
			result.Children = map[string]*ParsedContent{
				"ability": parsed,
			}
		}
	}

	return result, nil
}

// collectAbilityChildren returns all @type: ability descendants of a section, in
// document order, recursing through unannotated intermediaries (sub-headings like
// "Signature Ability") but not descending into other annotated children.
func collectAbilityChildren(section *parser.Section) []*parser.Section {
	var out []*parser.Section
	for _, child := range section.Children {
		switch child.Type() {
		case "ability":
			out = append(out, child)
		case "":
			out = append(out, collectAbilityChildren(child)...)
		}
	}
	return out
}
