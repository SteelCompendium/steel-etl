package site

// High-Fantasy Steel RETAINER advancement cards for the Steel Compendium site.
//
// Retainer statblocks (Goblin Guide, Minotaur Gorer, …) are `type: statblock`,
// but their bodies append advancement abilities under H6 headings
// "###### Level N Retainer Advancement Ability". Those blockquotes used to be
// slurped into the creature JSON island's flat feature list (polluting it). We
// split them out here: the island is built from the pre-advancement BASE body,
// and the advancement abilities re-emit as one Forged Band card with leveled
// .fb__band--adv tiers below the statblock (spec §3, Plan 4). Site-side only —
// no parser/schema/SCC change.

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// retainerAdvHeadingRe matches the advancement separator headings. #{1,6} is
// defensive against a future heading-depth change; "Retainer" (not "Role")
// keeps the chapter's separate Role Advancement pages untouched.
var retainerAdvHeadingRe = regexp.MustCompile(
	`(?im)^#{1,6}[ \t]+Level[ \t]+(\d+)[ \t]+Retainer[ \t]+Advancement[ \t]+Ability[ \t]*$`)

// retainerAdvGroup is one advancement tier: its level plus the blockquote body
// that follows its heading (up to the next heading or end of body).
type retainerAdvGroup struct {
	Level int
	Body  string
}

// splitRetainerAdvancement splits a statblock body into the pre-advancement
// base (fed to the island unchanged) and the ordered advancement groups.
// Returns (body, nil) when there are no advancement headings, so every
// non-retainer statblock is a no-op.
func splitRetainerAdvancement(body string) (string, []retainerAdvGroup) {
	locs := retainerAdvHeadingRe.FindAllStringSubmatchIndex(body, -1)
	if len(locs) == 0 {
		return body, nil
	}
	base := strings.TrimRight(body[:locs[0][0]], "\n")
	groups := make([]retainerAdvGroup, 0, len(locs))
	for i, loc := range locs {
		level, _ := strconv.Atoi(body[loc[2]:loc[3]]) // capture group 1 = the number
		start := loc[1]                               // end of the heading match
		end := len(body)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		groups = append(groups, retainerAdvGroup{
			Level: level,
			Body:  strings.TrimSpace(body[start:end]),
		})
	}
	return base, groups
}

// retainerRoleKey snaps the first word of the first `roles` entry
// ("Harrier Retainer" → "harrier") to a CSS-colored role key, so the Forged
// Band head accents in the retainer's role color. Unknown/absent → "" (the
// card renders in the neutral fallback).
func retainerRoleKey(fm string) string {
	roles := parseFrontmatterList(fm, "roles")
	if len(roles) == 0 {
		return ""
	}
	fields := strings.Fields(roles[0])
	if len(fields) == 0 {
		return ""
	}
	key := strings.ToLower(fields[0])
	if knownRoleKeys[key] {
		return key
	}
	return ""
}

// renderRetainerAdvancement renders the advancement groups as ONE Forged Band
// card (leveled .fb__band--adv tiers via renderFbFeats), to sit below the
// statblock island. Returns "" when there are no groups, so non-retainer
// statblocks add nothing. The leading "\n" separates it from the island div.
func renderRetainerAdvancement(fm string, groups []retainerAdvGroup) string {
	if len(groups) == 0 {
		return ""
	}
	var feats []fbFeature
	for _, g := range groups {
		rfs := content.ParseRichFeatures(g.Body)
		for i := range rfs {
			rfs[i].Level = g.Level // stamp the heading's level (no bold label to detect)
		}
		feats = append(feats, fbFeaturesFromRich(rfs)...)
	}
	if len(feats) == 0 {
		return ""
	}
	eyebrow := ""
	if roles := parseFrontmatterList(fm, "roles"); len(roles) > 0 {
		eyebrow = strings.TrimSpace(roles[0])
	}
	doc := fbDoc{
		Name:     "Advancement Abilities",
		Eyebrow:  eyebrow,
		Role:     retainerRoleKey(fm),
		Features: feats,
	}
	return "\n" + renderFeatureblockCard(doc)
}
