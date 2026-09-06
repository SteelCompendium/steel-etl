package site

import "fmt"

// featID mints the id for a nested feature card head (a statblock's or
// featureblock's feature). Material's search indexer opens a new index section
// only for headings WITH an id; an id-less heading that shares its tag with the
// enclosing section's heading gets glued onto that section's TITLE instead
// ("Demon Lord's AspectGrasping AppendagesWarping Strike…"), which is what
// buried exact-name searches (SC-306). seen dedupes within one block: a second
// "Free Strike" becomes sc-feat-free-strike-2. Top-level card heads stay
// id-less on purpose — the page H1 already owns that title.
func featID(seen map[string]int, name string) string {
	// Unwrap markdown links first (mdLinkRe, ability_cards.go): a name like
	// "[Solo](../rule/organization/solo.md) Monster" would otherwise slugify
	// the link target into the id too (sc-feat-solo-rule-organization-solo-md-monster).
	name = mdLinkRe.ReplaceAllString(name, "$1")
	base := "sc-feat-" + slugify(name)
	if base == "sc-feat-" {
		base = "sc-feat-feature"
	}
	seen[base]++
	if n := seen[base]; n > 1 {
		return fmt.Sprintf("%s-%d", base, n)
	}
	return base
}
