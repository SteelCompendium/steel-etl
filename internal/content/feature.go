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

	// Trait is reserved for the rulebook's trait homes. The only trait home
	// reachable through FeatureParser is an ancestry (monster traits are emitted
	// by statblock_parse.go; companions are NOT a trait home — the Beastheart
	// book calls companion grants "features", never "traits"). Everything else
	// (class/domain/college/kit/companion/common) is a plain feature. See
	// docs/superpowers/specs/2026-06-07-feature-taxonomy-design.md.
	isTrait := ancestryID != ""
	featureKind := "feature"
	if isTrait {
		featureKind = "trait"
	}

	fm := map[string]any{
		"name": cleanName,
		"type": featureKind,
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

	// Build the hub-and-spoke type path. The base case is unmarked; the `trait`
	// marker is inserted only for trait homes (ancestry). Plain features take
	// feature.{entity}.level-{N}[.{kit}]; ability.go handles feature.ability.*.
	// Companion features: feature.companion.{species}.level-{N} (no trait marker).
	typePath := []string{"feature"}
	if isTrait {
		typePath = append(typePath, "trait")
	}
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
	// Named feature-group grouping under a class/ancestry (e.g. the fury's
	// "Stormwight Kits"): a feature sitting directly inside a named feature-group
	// — no level, no nearer kit — takes the group id as a path segment so its
	// siblings collapse into one browse-index group instead of dangling at the
	// class root. Level groups carry @level (not @id) so they never match here;
	// the kit branch below handles kit-scoped features (Boren, Corven, …).
	if levelStr == "" && kitID == "" && (classID != "" || ancestryID != "") {
		if groupID := findAncestorID(ctx, section.HeadingLevel, "feature-group"); groupID != "" {
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
