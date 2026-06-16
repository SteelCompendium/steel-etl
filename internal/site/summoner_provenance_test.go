package site

import "testing"

func TestSummonerProvenanceEyebrow(t *testing.T) {
	cases := []struct {
		name string
		scc  string
		want string
	}{
		{
			name: "rival minion",
			scc:  "mcdm.summoner.v1/monster.rival.4th-echelon.summoner.minion/zombie-titan",
			want: "Rival Summoner Summon · Echelon 4",
		},
		{
			name: "rival elite",
			scc:  "mcdm.summoner.v1/monster.rival.1st-echelon.statblock/rival-summoner",
			want: "Rival Summoner · Echelon 1",
		},
		{
			name: "portfolio minion",
			scc:  "mcdm.summoner.v1/monster.minion.summoner.undead.statblock/skeleton",
			want: "Summoner Minion · Undead",
		},
		{
			name: "champion",
			scc:  "mcdm.summoner.v1/monster.champion.summoner.demon.statblock/demon-lords-aspect",
			want: "Summoner Champion · Demon",
		},
		{
			// CRITICAL non-match: Monsters-book rivals share the shape but are a
			// different book and must be left alone.
			name: "monsters-book rival is not matched",
			scc:  "mcdm.monsters.v1/monster.rival.4th-echelon.statblock/rival-fury",
			want: "",
		},
		{
			name: "unrelated summoner code is not matched",
			scc:  "mcdm.summoner.v1/class/summoner",
			want: "",
		},
		{
			name: "empty",
			scc:  "",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := summonerProvenanceEyebrow(tc.scc); got != tc.want {
				t.Errorf("summonerProvenanceEyebrow(%q) = %q, want %q", tc.scc, got, tc.want)
			}
		})
	}
}
