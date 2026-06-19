package cli

import "testing"

func TestIsLevelGroupingFeatureID(t *testing.T) {
	grouping := []string{"1st-level-features", "2nd-level-features", "10th-level-features"}
	for _, id := range grouping {
		if !isLevelGroupingFeatureID(id) {
			t.Errorf("%q should be detected as a level-grouping feature id", id)
		}
	}
	// Intentional features (lookup containers) and normal features must NOT match.
	keep := []string{
		"1st-level-circle-features", // has "circle"
		"5th-level-circle-feature",  // singular, has "circle"
		"summoners-dominion",
		"perk",
		"basics",
	}
	for _, id := range keep {
		if isLevelGroupingFeatureID(id) {
			t.Errorf("%q must NOT be flagged as a level-grouping feature id", id)
		}
	}
}
