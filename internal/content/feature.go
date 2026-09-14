package content

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

// featureSource returns the feature_source frontmatter value for a feature or
// ability. It is Summoner-book-only: non-Summoner books get "" (field omitted).
// Within the Summoner book the value comes from the section's own
// @feature_source annotation or an inherited ancestor's (via the context stack,
// mirroring @level propagation); unmarked Summoner features default to
// "summoner". Only FeatureParser/AbilityParser call this, so statblock/
// featureblock/monster-group descendants never inherit it. The value space is
// forward-compatible with Phase-2 "circle-of-<name>" slugs. See
// docs/superpowers/specs/2026-06-18-summoner-feature-source-design.md.
func featureSource(ctx *context.ContextStack, section *parser.Section) string {
	book, _ := ctx.Lookup(section.HeadingLevel, "book")
	if !strings.HasPrefix(book, "mcdm.summoner.") {
		return ""
	}
	// The section's own @feature_source wins (the explicit-mark features such as
	// Summoner's Dominion), then an inherited container's (the circle-lookup
	// containers). Checking the section directly avoids depending on the pipeline
	// having pushed this section's own annotation onto the stack first.
	if section.Annotation != nil {
		if v, ok := section.Annotation["feature_source"]; ok && v != "" {
			return v
		}
	}
	if v, ok := ctx.Lookup(section.HeadingLevel, "feature_source"); ok && v != "" {
		return v
	}
	return "summoner"
}

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

	// Companion species containers (beastheart) are first-class. They are
	// statblock-IDENTITY entities in the monster.companion.<class>.* namespace
	// (mirroring monster.rival.<echelon>.statblock), but keep rendering as a
	// feature-group page (spec 2026-06-13 §5) and still push `companion`
	// context to their member features/abilities.
	if companion, ok := section.Annotation["companion"]; ok && companion != "" {
		fm["companion"] = companion
		classID := findAncestorID(ctx, section.HeadingLevel, "class")
		result.TypePath = compactPath("monster", "companion", classID, "statblock")
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

	// A feature may be granted by a treasure rule page (mirrors AbilityParser's
	// rule-ancestor check, SC-323). Nearest-recognised-ancestor-wins, exactly
	// like AbilityParser's own combined walk: findTreasureRuleAncestor returns
	// non-"" only when the treasure rule is NEARER than any class/kit/
	// ancestry/companion ancestor (it returns "" the moment its own walk hits
	// one of those first), so calling it unconditionally — not gated on
	// whether classID/kitID/ancestryID/companionID are empty, which are
	// whole-tree lookups, not nearest-specific — is what gives a treasure rule
	// nested inside e.g. a class section correct precedence over a class
	// ancestor further out (round-2 review MED-2).
	treasureRuleID := findTreasureRuleAncestor(ctx, section.HeadingLevel)
	if treasureRuleID != "" {
		// The whole-tree lookups above are stale once a NEARER treasure rule
		// wins: clear them so a farther-out class/kit/ancestry/companion
		// doesn't leak into frontmatter or the path. Mirrors AbilityParser,
		// where a treasure-rule win means parentID/parentType are never set at
		// all (single combined walk, only one of the two can be non-empty).
		classID, kitID, ancestryID, companionID = "", "", "", ""
	}

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

	// Capture a cost embedded in the heading's trailing parenthetical — ancestry
	// purchased traits encode their ancestry-point cost there (e.g. "Barbed Tail
	// (1 Point)"). An explicit @cost annotation wins, mirroring ability.go.
	if v, ok := section.Annotation["cost"]; ok && v != "" {
		fm["cost"] = v
	} else if cost := extractCostSuffix(section.Heading); cost != "" {
		fm["cost"] = cost
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
	if fs := featureSource(ctx, section); fs != "" {
		fm["feature_source"] = fs
	}
	if treasureRuleID != "" {
		// Relationship is a frontmatter link, never path nesting (scc-reference
		// "relationships are frontmatter links, never path nesting").
		if book, ok := ctx.Lookup(section.HeadingLevel, "book"); ok && book != "" {
			fm["granted_by"] = scc.Classify(book, []string{"rule", "treasure"}, treasureRuleID)
		}
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
		// Companion features carry the owning class as a subgroup segment
		// (feature.companion.beastheart.wolf.level-N/<id>), mirroring the
		// monster.companion.beastheart.* container. classID is computed above
		// (findAncestorID … "class"); guard against empty so we never emit a
		// double-dot path.
		typePath = append(typePath, "companion")
		if classID != "" {
			typePath = append(typePath, classID)
		}
		typePath = append(typePath, companionID)
	} else if classID != "" {
		typePath = append(typePath, classID)
	} else if ancestryID != "" {
		typePath = append(typePath, ancestryID)
	} else if treasureRuleID != "" {
		// Treasure-granted features are flat under `feature.treasure`, mirroring
		// AbilityParser's `feature.ability.treasure` bucket (SC-323); the
		// granting rule page is carried as the `granted_by` frontmatter link
		// above, not nested into the path. Deliberately flat even when a
		// `feature-group` sits between the feature and the treasure rule —
		// unlike the `common` branch below, this never appends a group id
		// (round-2 review LOW-6: no behaviour change, just documenting it).
		typePath = append(typePath, "treasure")
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
