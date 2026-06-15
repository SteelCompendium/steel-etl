package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const summonSkeletonFM = `keywords:
    - Undead
level: 1
name: Skeleton
organization: Minion
role: Harrier
size: 1S
speed: 6
type: statblock
scc: mcdm.summoner.v1/monster.rivals.2nd-echelon.summoner.minion/skeleton
`

const summonGraveKnightFM = `keywords:
    - Undead
name: Grave Knight
organization: Minion
role: Brute
size: 1M
speed: 6
type: statblock
scc: mcdm.summoner.v1/monster.rivals.2nd-echelon.summoner.minion/grave-knight
`

func TestRivalSummonsCards(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "skeleton.md"), []byte("---\n"+summonSkeletonFM+"---\n\nbody\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "grave-knight.md"), []byte("---\n"+summonGraveKnightFM+"---\n\nbody\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := rivalSummonsCards(dir, "../summoner/minion", []string{"skeleton.md", "grave-knight.md"})
	for _, want := range []string{
		`<div class="sc-cards">`,
		`href="../summoner/minion/skeleton/"`,      // href base applied, .md → dir URL
		`href="../summoner/minion/grave-knight/"`,
		`<div class="sc-card__name">Skeleton</div>`,
		`<div class="sc-card__name">Grave Knight</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rivalSummonsCards missing %q in:\n%s", want, got)
		}
	}
	// natural sort: grave-knight before skeleton.
	if strings.Index(got, "Grave Knight") > strings.Index(got, "Skeleton") {
		t.Errorf("cards not natural-sorted:\n%s", got)
	}
}
