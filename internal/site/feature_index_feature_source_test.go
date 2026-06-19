package site

import "testing"

func TestExtractPreviewItem_FeatureSource(t *testing.T) {
	fm := "name: Summoner's Dominion\ntype: feature\nclass: summoner\nlevel: 2\nfeature_source: circle\n"
	it := extractPreviewItem(fm, "", "feature", "summoner")
	if it.FeatureSource != "circle" {
		t.Errorf("FeatureSource = %q, want circle", it.FeatureSource)
	}
}

func TestExtractPreviewItem_FeatureSourceAbsent(t *testing.T) {
	fm := "name: Growing Ferocity\ntype: feature\nclass: fury\nlevel: 1\n"
	it := extractPreviewItem(fm, "", "feature", "fury")
	if it.FeatureSource != "" {
		t.Errorf("FeatureSource = %q, want empty", it.FeatureSource)
	}
}
