package site

import (
	"strings"
	"testing"
)

func TestApplySearchBoost(t *testing.T) {
	classPage := "---\nname: Fury\ntype: class\n---\n\n# Fury\n"
	got := string(applySearchBoost([]byte(classPage)))
	if !strings.HasPrefix(got, "---\nsearch:\n  boost: 4\nname: Fury\n") {
		t.Errorf("class page: boost not injected after opening ---:\n%s", got)
	}
	if !strings.Contains(got, "# Fury") {
		t.Errorf("class page: body lost")
	}

	sb := "---\nname: Goblin Warrior\ntype: statblock\n---\nbody\n"
	got = string(applySearchBoost([]byte(sb)))
	if !strings.Contains(got, "search:\n  boost: 0.6\n") {
		t.Errorf("statblock: want boost 0.6, got:\n%s", got)
	}

	ability := "---\nname: Brutal Slam\ntype: ability\n---\nbody\n"
	if got := string(applySearchBoost([]byte(ability))); got != ability {
		t.Errorf("ability: must be unchanged (default boost), got:\n%s", got)
	}

	noFM := "# Plain page\n"
	if got := string(applySearchBoost([]byte(noFM))); got != noFM {
		t.Errorf("page without frontmatter must be unchanged")
	}
}

func TestSearchExcluded(t *testing.T) {
	if !searchExcluded([]string{"Read"}, "Read") {
		t.Error("Read should be excluded")
	}
	if searchExcluded([]string{"Read"}, "Browse") {
		t.Error("Browse should not be excluded")
	}
	if searchExcluded(nil, "Browse") {
		t.Error("nil list excludes nothing")
	}
}
