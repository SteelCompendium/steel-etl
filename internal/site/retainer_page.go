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
