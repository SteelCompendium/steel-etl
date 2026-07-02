package site

// Per-type search ranking boosts (Material's `search: boost:` page
// frontmatter). Canonical reference pages outrank the 555 monster statblocks
// for their own names ("fury" should find the Fury class, not four Rival Fury
// statblocks). Injected in buildSection for non-search-excluded sections only —
// Read pages get `search: exclude` later (applySearchExclusion) and MUST NOT
// carry a second `search:` YAML key.
// See workspace docs/superpowers/specs/2026-07-01-v2-ux-analysis.md §2.7.

import "strings"

var searchBoostByType = map[string]string{
	"class":           "4",
	"ancestry":        "3",
	"condition":       "3",
	"rule":            "3",
	"movement":        "3",
	"negotiation":     "3",
	"skill":           "2",
	"kit":             "2",
	"culture":         "2",
	"career":          "2",
	"perk":            "2",
	"title":           "2",
	"complication":    "2",
	"project":         "2",
	"god":             "2",
	"saint":           "2",
	"treasure":        "2",
	"statblock":       "0.6",
	"featureblock":    "0.6",
	"dynamic-terrain": "0.7",
}

// applySearchBoost injects `search:\n  boost: <n>` at the top of the page
// frontmatter when the page type has a boost mapping. Pages without
// frontmatter, or with an unmapped type (feature/ability/chapter/…, which
// keep the default 1×), pass through unchanged.
func applySearchBoost(data []byte) []byte {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return data
	}
	fm, _ := splitFrontmatter(content)
	typ := strings.TrimSpace(parseFrontmatterField(fm, "type"))
	boost, ok := searchBoostByType[typ]
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
