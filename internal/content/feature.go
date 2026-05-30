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

	// Look up parent ancestry from context
	ancestryID := findAncestorID(ctx, section.HeadingLevel, "ancestry")

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

	// Build type path: feature.trait.{parent}.level-{N}[.{kit}]
	typePath := []string{"feature", "trait"}
	if classID != "" {
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

	// Embed child ability sections (annotated with @type: ability).
	//
	// Two shapes exist:
	//   - Single-ability traits (e.g. "Faithful Friend"): exactly one ability
	//     child. We embed it as a structured nested object (Children["ability"]),
	//     mirroring the kit signature-ability pattern, for the SDK trait schema
	//     which has a singular `ability` field.
	//   - Multi-ability containers (e.g. "Censor Abilities", "Fury Abilities"):
	//     many abilities organized under sub-headings. A singular embed makes no
	//     sense here, so we skip the structured field and instead re-render the
	//     body so every ability appears inline, in document order, under its
	//     sub-heading.
	//
	// In both cases the body must include the ability content, since
	// FullBodySource() omits annotated children.
	abilityChildren := collectAbilityChildren(section)
	if len(abilityChildren) > 0 {
		result.Body = section.FullBodySourceWithAbilities()
	}
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
