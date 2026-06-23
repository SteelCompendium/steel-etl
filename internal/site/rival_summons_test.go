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
scc: mcdm.summoner.v1/monster.rival.2nd-echelon.summoner.minion/skeleton
`

const summonGraveKnightFM = `keywords:
    - Undead
name: Grave Knight
organization: Minion
role: Brute
size: 1M
speed: 6
type: statblock
scc: mcdm.summoner.v1/monster.rival.2nd-echelon.summoner.minion/grave-knight
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
		`class="sb-cards"`,
		`class="sb-prev__link" href="../summoner/minion/skeleton/"`,
		`class="sb-prev__link" href="../summoner/minion/grave-knight/"`,
		`<h2 class="sc-head__slot sc-head__left-primary sc-head__slot--line">Skeleton</h2>`,
		`<h2 class="sc-head__slot sc-head__left-primary sc-head__slot--line">Grave Knight</h2>`,
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

// rivalSummonerFM is a summoner-book conjurer (organization Elite, not Minion).
const rivalSummonerFM = `name: Rival Summoner
organization: Elite
role: Controller
type: statblock
scc: mcdm.summoner.v1/monster.rival.2nd-echelon.statblock/rival-summoner
`

// rivalFuryFM is a co-located Monsters-book rival — must NOT get a summons block.
const rivalFuryFM = `name: Rival Fury
organization: Solo
role: Brute
type: statblock
scc: mcdm.monsters.v1/monster.rival.2nd-echelon.statblock/rival-fury
`

func writeStatblockPage(t *testing.T, path, frontmatter, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	body := "---\n" + frontmatter + "---\n\n# " + name + "\n\n---\n\n" +
		`<div class="sb-wrap" data-creature="x"><article class="sb">stats</article></div>` + "\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestAugmentRivalSummonerPages(t *testing.T) {
	sec := t.TempDir()
	ech := filepath.Join(sec, "monster", "rival", "2nd-echelon")
	writeStatblockPage(t, filepath.Join(ech, "rival-summoner.md"), rivalSummonerFM, "Rival Summoner")
	writeStatblockPage(t, filepath.Join(ech, "rival-fury.md"), rivalFuryFM, "Rival Fury")
	minion := filepath.Join(ech, "summoner", "minion")
	writeStatblockPage(t, filepath.Join(minion, "skeleton.md"), summonSkeletonFM, "Skeleton")
	writeStatblockPage(t, filepath.Join(minion, "grave-knight.md"), summonGraveKnightFM, "Grave Knight")

	n, errs := augmentRivalSummonerPages(sec)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if n != 3 { // 1 forward (rival-summoner) + 2 back-links (skeleton, grave-knight)
		t.Errorf("augment count = %d, want 3", n)
	}

	rs := readFile(filepath.Join(ech, "rival-summoner.md"))
	for _, want := range []string{
		"## Summons",
		`href="../summoner/minion/skeleton/"`,
		`href="../summoner/minion/grave-knight/"`,
	} {
		if !strings.Contains(rs, want) {
			t.Errorf("rival-summoner page missing %q", want)
		}
	}

	// Monsters-book rival must be untouched.
	if rf := readFile(filepath.Join(ech, "rival-fury.md")); strings.Contains(rf, "## Summons") {
		t.Errorf("rival-fury must not get a Summons block:\n%s", rf)
	}

	// Each summon page gets exactly one back-link to the Rival Summoner.
	sk := readFile(filepath.Join(minion, "skeleton.md"))
	if c := strings.Count(sk, "sb-backlink"); c != 1 {
		t.Errorf("skeleton sb-backlink count = %d, want 1", c)
	}
	if !strings.Contains(sk, `href="../../../rival-summoner/"`) {
		t.Errorf("skeleton missing back-link href:\n%s", sk)
	}
	if !strings.Contains(sk, "Summoned by") {
		t.Errorf("skeleton missing back-link label:\n%s", sk)
	}
	// Back-link sits before the statblock card.
	if strings.Index(sk, "sb-backlink") > strings.Index(sk, `<div class="sb-wrap"`) {
		t.Errorf("back-link should precede the sb-wrap card:\n%s", sk)
	}

	// Idempotent: a second run adds nothing.
	n2, _ := augmentRivalSummonerPages(sec)
	if n2 != 0 {
		t.Errorf("second run count = %d, want 0 (idempotent)", n2)
	}
	if c := strings.Count(readFile(filepath.Join(ech, "rival-summoner.md")), "## Summons"); c != 1 {
		t.Errorf("Summons block duplicated on re-run (count=%d)", c)
	}
}

func TestAugmentRivalSummonerPages_NoTree(t *testing.T) {
	// No monster/rival dir → no-op, no error.
	n, errs := augmentRivalSummonerPages(t.TempDir())
	if n != 0 || len(errs) != 0 {
		t.Errorf("expected no-op, got n=%d errs=%v", n, errs)
	}
}
