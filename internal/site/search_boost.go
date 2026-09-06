package site

// Per-type search ranking boosts (Material's `search: boost:` page
// frontmatter). Canonical reference pages outrank statblocks for their own
// names ("fury" should find the Fury class, not four Rival Fury statblocks).
// Statblocks, featureblocks and dynamic terrain sit at the default boost (1):
// the old 0.6/0.7 demotions buried monsters under their own names ("Goblin
// Warrior" lost to "Warrior Priest") — SC-306. Injected in buildSection for
// non-search-excluded sections only — Read pages get `search: exclude` later
// (applySearchExclusion) and MUST NOT carry a second `search:` YAML key.
// See workspace docs/superpowers/specs/2026-07-01-v2-ux-analysis.md §2.7 and
// docs/superpowers/specs/2026-09-06-search-ranking-design.md.

import "strings"

var searchBoostByType = map[string]string{
	"class":        "4",
	"ancestry":     "3",
	"condition":    "3",
	"rule":         "3",
	"movement":     "3",
	"negotiation":  "3",
	"skill":        "2",
	"kit":          "2",
	"culture":      "2",
	"career":       "2",
	"perk":         "2",
	"title":        "2",
	"complication": "2",
	"project":      "2",
	"god":          "2",
	"saint":        "2",
	"treasure":     "2",
}

// commonActionBoost ranks the book's UNIVERSAL actions with the rules glossary
// (3) instead of with the ~3,000 class-specific features they'd otherwise tie
// with. Two SCC type-paths qualify (SC-179):
//
//   - `feature.common.*` (`main-actions` / `maneuvers` / `move-actions`) — the
//     common action rule pages (Free Strike, Charge, Grab, Hide, …), the
//     canonical explanation of each action.
//   - `feature.ability.common` — the common ability cards every creature has
//     (Melee/Ranged Weapon Free Strike, Grab, Knockback, Escape Grab).
//
// Motivation: a reader searching "free strike" was drowned by the hundreds of
// class abilities and statblocks whose text merely mentions it, and never reached
// either the rule page or the two ability cards. These pages are looked up by
// every player at every table; class-specific features are not.
const commonActionBoost = "3"

// sccTypePath returns the type-path segment of an SCC code — the substring
// between the first and second '/' (`mcdm.heroes.v1/feature.ability.common/grab`
// → `feature.ability.common`) — or "" when code has fewer than three
// '/'-separated segments. A code with MORE than three segments still returns
// just that one segment (whatever comes after the second '/' is ignored); every
// registry code today is exactly three segments, so this is unreachable in
// practice, but it isn't the ">3 segments → \"\"" some earlier wording implied.
func sccTypePath(code string) string {
	_, rest, ok := strings.Cut(strings.TrimSpace(code), "/")
	if !ok {
		return ""
	}
	typePath, _, ok := strings.Cut(rest, "/")
	if !ok {
		return ""
	}
	return typePath
}

// searchBoostFor returns the boost for a page's frontmatter, preferring the
// SCC-derived common-action rule over the coarser per-type table.
func searchBoostFor(fm string) (string, bool) {
	typ := strings.TrimSpace(parseFrontmatterField(fm, "type"))
	if typ == "feature" || typ == "ability" {
		tp := sccTypePath(parseFrontmatterField(fm, "scc"))
		// The prefix match is deliberately open-ended: any FUTURE
		// `feature.common.<bucket>` (today: main-actions/maneuvers/move-actions)
		// inherits this tier automatically, with no review gate. Fine while
		// every bucket under `feature.common.` is a universal action; revisit
		// this line if that ever stops being true (INFO-2, SC-179 round 3 review).
		if tp == "feature.ability.common" || strings.HasPrefix(tp, "feature.common.") {
			return commonActionBoost, true
		}
	}
	boost, ok := searchBoostByType[typ]
	return boost, ok
}

// applySearchBoost injects `search:\n  boost: <n>` at the top of the page
// frontmatter when the page earns a boost. Pages without frontmatter, or with an
// unmapped type (feature/ability/chapter/…, which keep the default 1×), pass
// through unchanged.
func applySearchBoost(data []byte) []byte {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return data
	}
	fm, _ := splitFrontmatter(content)
	boost, ok := searchBoostFor(fm)
	if !ok {
		return data
	}
	rest := strings.TrimPrefix(content, "---\n")
	return []byte("---\nsearch:\n  boost: " + boost + "\n" + rest)
}

// searchExcluded reports whether a section name is listed in search_exclude.
func searchExcluded(excluded []string, name string) bool {
	for _, e := range excluded {
		if e == name {
			return true
		}
	}
	return false
}
