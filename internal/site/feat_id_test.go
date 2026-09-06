package site

import "testing"

func TestFeatID(t *testing.T) {
	seen := map[string]int{}
	if got := featID(seen, "Grasping Appendages"); got != "sc-feat-grasping-appendages" {
		t.Errorf("got %q", got)
	}
	if got := featID(seen, "Free Strike"); got != "sc-feat-free-strike" {
		t.Errorf("got %q", got)
	}
	if got := featID(seen, "Free Strike"); got != "sc-feat-free-strike-2" {
		t.Errorf("duplicate must get a suffix, got %q", got)
	}
	if got := featID(seen, "  "); got != "sc-feat-feature" {
		t.Errorf("empty name falls back, got %q", got)
	}
}
