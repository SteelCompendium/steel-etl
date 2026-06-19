package output

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

func metaFromTransform(t *testing.T, fm map[string]any) map[string]any {
	t.Helper()
	out := TransformToSDKFormat("mcdm.summoner.v1/feature.summoner.level-2/x", &content.ParsedContent{Frontmatter: fm, Body: "Body."})
	meta, _ := out["metadata"].(map[string]any)
	if meta == nil {
		t.Fatalf("no metadata in transform output: %v", out)
	}
	return meta
}

func TestSDKMetadata_FeatureSource_Trait(t *testing.T) {
	meta := metaFromTransform(t, map[string]any{"name": "Summoner's Dominion", "type": "feature", "feature_source": "circle"})
	if meta["feature_source"] != "circle" {
		t.Errorf("trait metadata feature_source = %v, want circle", meta["feature_source"])
	}
}

func TestSDKMetadata_FeatureSource_Ability(t *testing.T) {
	meta := metaFromTransform(t, map[string]any{"name": "X", "type": "ability", "feature_source": "summoner"})
	if meta["feature_source"] != "summoner" {
		t.Errorf("ability metadata feature_source = %v, want summoner", meta["feature_source"])
	}
}

func TestSDKMetadata_FeatureSource_AbsentWhenUnset(t *testing.T) {
	meta := metaFromTransform(t, map[string]any{"name": "Growing Ferocity", "type": "feature"})
	if _, ok := meta["feature_source"]; ok {
		t.Errorf("feature_source must be absent when unset, got %v", meta["feature_source"])
	}
}
