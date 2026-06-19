package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// summonerCtx builds a context stack rooted in the Summoner book with a
// level-N feature-group at H3, optionally carrying @feature_source on the group.
func summonerCtx(level, groupSource string) *context.ContextStack {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.summoner.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "summoner"})
	groupMeta := context.Metadata{"type": "feature-group", "level": level}
	if groupSource != "" {
		groupMeta["feature_source"] = groupSource
	}
	ctx.Push(3, groupMeta)
	return ctx
}

func TestFeatureSource_UnmarkedSummonerFeature(t *testing.T) {
	sec := &parser.Section{Heading: "Perk", HeadingLevel: 4, Annotation: map[string]string{"type": "feature"}}
	res, _ := (&FeatureParser{}).Parse(summonerCtx("2", ""), sec)
	if res.Frontmatter["feature_source"] != "summoner" {
		t.Errorf("feature_source = %v, want summoner", res.Frontmatter["feature_source"])
	}
}

func TestFeatureSource_ExplicitCircleFeature(t *testing.T) {
	sec := &parser.Section{Heading: "Summoner's Dominion", HeadingLevel: 4,
		Annotation: map[string]string{"type": "feature", "feature_source": "circle"}}
	res, _ := (&FeatureParser{}).Parse(summonerCtx("2", ""), sec)
	if res.Frontmatter["feature_source"] != "circle" {
		t.Errorf("feature_source = %v, want circle", res.Frontmatter["feature_source"])
	}
}

func TestFeatureSource_InheritedFromContainer(t *testing.T) {
	// A pick-child under a @feature_source: circle container (e.g. the
	// 1st-level-circle-features container) inherits circle.
	sec := &parser.Section{Heading: "Channel", HeadingLevel: 4, Annotation: map[string]string{"type": "feature"}}
	res, _ := (&FeatureParser{}).Parse(summonerCtx("1", "circle"), sec)
	if res.Frontmatter["feature_source"] != "circle" {
		t.Errorf("inherited feature_source = %v, want circle", res.Frontmatter["feature_source"])
	}
}

func TestFeatureSource_AbilityInherits(t *testing.T) {
	sec := &parser.Section{Heading: "Some Circle Ability", HeadingLevel: 4, Annotation: map[string]string{"type": "ability"}}
	res, _ := (&AbilityParser{}).Parse(summonerCtx("2", "circle"), sec)
	if res.Frontmatter["feature_source"] != "circle" {
		t.Errorf("ability feature_source = %v, want circle", res.Frontmatter["feature_source"])
	}
}

func TestFeatureSource_NonSummonerBookOmitted(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "fury"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "level": "1"})
	sec := &parser.Section{Heading: "Growing Ferocity", HeadingLevel: 4, Annotation: map[string]string{"type": "feature"}}
	res, _ := (&FeatureParser{}).Parse(ctx, sec)
	if _, ok := res.Frontmatter["feature_source"]; ok {
		t.Errorf("non-Summoner feature must omit feature_source, got %v", res.Frontmatter["feature_source"])
	}
}
