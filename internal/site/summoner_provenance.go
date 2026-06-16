package site

import "strings"

// summonerProvenanceEyebrow maps a summoner-book creature statblock's SCC code to
// the provenance label shown in the statblock head's sb__kw eyebrow, overriding
// the otherwise-useless keyword line (which today renders "—" or, for portfolio
// minions, the creature's own name). Returns "" for any code that is not a
// summoner-book creature statblock.
//
// The Monsters-book `mcdm.monsters.v1/monster.rival.{ech}.statblock` tree shares
// the `monster.rival.*` shape but is a different book and must stay untouched —
// the `mcdm.summoner.` source-prefix gate is what excludes it.
//
// Recognized type-paths (the segment between source and item):
//
//	monster.rival.{ech}.summoner.minion          → "Rival Summoner Summon · Echelon N"
//	monster.rival.{ech}.statblock                → "Rival Summoner · Echelon N"
//	monster.minion.summoner.{circle}.statblock   → "Summoner Minion · {Circle}"
//	monster.champion.summoner.{circle}.statblock → "Summoner Champion · {Circle}"
func summonerProvenanceEyebrow(scc string) string {
	scc = strings.TrimSpace(scc)
	src, rest, ok := strings.Cut(scc, "/")
	if !ok || !strings.HasPrefix(src, "mcdm.summoner.") {
		return ""
	}
	// Drop the trailing /item to leave the type-path.
	typePath, _, ok := strings.Cut(rest, "/")
	if !ok {
		return ""
	}
	seg := strings.Split(typePath, ".")

	switch {
	// monster.rival.{ech}.summoner.minion
	case len(seg) == 5 && seg[0] == "monster" && seg[1] == "rival" &&
		seg[3] == "summoner" && seg[4] == "minion":
		if n := echelonNum(seg[2]); n != "" {
			return "Rival Summoner Summon · Echelon " + n
		}
	// monster.rival.{ech}.statblock
	case len(seg) == 4 && seg[0] == "monster" && seg[1] == "rival" &&
		seg[3] == "statblock":
		if n := echelonNum(seg[2]); n != "" {
			return "Rival Summoner · Echelon " + n
		}
	// monster.minion.summoner.{circle}.statblock
	case len(seg) == 5 && seg[0] == "monster" && seg[1] == "minion" &&
		seg[2] == "summoner" && seg[4] == "statblock":
		return "Summoner Minion · " + titleCase(seg[3])
	// monster.champion.summoner.{circle}.statblock
	case len(seg) == 5 && seg[0] == "monster" && seg[1] == "champion" &&
		seg[2] == "summoner" && seg[4] == "statblock":
		return "Summoner Champion · " + titleCase(seg[3])
	}
	return ""
}

// echelonNum extracts the leading number N from an "Nth-echelon" segment
// (e.g. "4th-echelon" → "4"). Returns "" if the segment isn't of that shape.
func echelonNum(seg string) string {
	rest, ok := strings.CutSuffix(seg, "-echelon")
	if !ok {
		return ""
	}
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		i++
	}
	if i == 0 {
		return ""
	}
	return rest[:i]
}
