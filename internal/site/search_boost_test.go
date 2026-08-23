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

// SC-179: universal actions/abilities rank with the rules glossary, not with the
// thousands of class-specific features they'd otherwise tie with.
func TestApplySearchBoostCommonActions(t *testing.T) {
	boosted := []struct{ name, page string }{
		{
			"common main-action rule page",
			"---\nname: Free Strike\nscc: mcdm.heroes.v1/feature.common.main-actions/free-strike\ntype: feature\n---\nbody\n",
		},
		{
			"common maneuver rule page",
			"---\nname: Grab\nscc: mcdm.heroes.v1/feature.common.maneuvers/grab\ntype: feature\n---\nbody\n",
		},
		{
			"common move-action rule page",
			"---\nname: Advance\nscc: mcdm.heroes.v1/feature.common.move-actions/advance\ntype: feature\n---\nbody\n",
		},
		{
			"common ability card",
			"---\nname: Melee Weapon Free Strike\nscc: mcdm.heroes.v1/feature.ability.common/melee-weapon-free-strike\ntype: ability\n---\nbody\n",
		},
	}
	for _, tc := range boosted {
		got := string(applySearchBoost([]byte(tc.page)))
		if !strings.HasPrefix(got, "---\nsearch:\n  boost: 3\n") {
			t.Errorf("%s: want boost 3, got:\n%s", tc.name, got)
		}
	}

	// Class-specific features/abilities keep the default 1×.
	unboosted := []string{
		"---\nname: Gouge\nscc: mcdm.heroes.v1/feature.ability.fury.level-1/gouge\ntype: ability\n---\nbody\n",
		"---\nname: Ferocity\nscc: mcdm.heroes.v1/feature.fury.level-1/ferocity\ntype: feature\n---\nbody\n",
		// A `feature`/`ability` page with no scc must not panic or match.
		"---\nname: Orphan\ntype: feature\n---\nbody\n",
	}
	for _, page := range unboosted {
		if got := string(applySearchBoost([]byte(page))); got != page {
			t.Errorf("page must be unchanged, got:\n%s", got)
		}
	}

	// A boosted TYPE still wins where the SCC rule doesn't apply.
	rule := "---\nname: Creature Free Strike\nscc: mcdm.monsters.v1/rule.monster/creature-free-strike\ntype: rule\n---\nbody\n"
	if got := string(applySearchBoost([]byte(rule))); !strings.Contains(got, "boost: 3\n") {
		t.Errorf("rule page: want boost 3, got:\n%s", got)
	}
}

func TestSCCTypePath(t *testing.T) {
	cases := []struct{ code, want string }{
		{"mcdm.heroes.v1/feature.ability.common/grab", "feature.ability.common"},
		{"mcdm.heroes.v1/feature.common.main-actions/free-strike", "feature.common.main-actions"},
		{"mcdm.heroes.v1/class/fury", "class"},
		{"mcdm.heroes.v1/feature.common.main-actions", ""}, // two segments only
		{"not-an-scc", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := sccTypePath(tc.code); got != tc.want {
			t.Errorf("sccTypePath(%q) = %q, want %q", tc.code, got, tc.want)
		}
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
