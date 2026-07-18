package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwningClass(t *testing.T) {
	cases := []struct {
		name     string
		scc      string
		wantSlug string
		wantName string
	}{
		{
			name:     "beastheart companion base",
			scc:      "mcdm.beastheart.v1/monster.companion.beastheart.statblock/wolf",
			wantSlug: "beastheart",
			wantName: "Beastheart",
		},
		{
			name:     "beastheart companion advancement-features",
			scc:      "mcdm.beastheart.v1/monster.companion.beastheart.advancement-features/wolf",
			wantSlug: "beastheart",
			wantName: "Beastheart",
		},
		{
			name:     "summoner fixture base",
			scc:      "mcdm.summoner.v1/monster.fixture.demon.featureblock/the-boil",
			wantSlug: "summoner",
			wantName: "Summoner",
		},
		{
			name:     "summoner fixture advancement-features",
			scc:      "mcdm.summoner.v1/monster.fixture.demon.advancement-features/the-boil",
			wantSlug: "summoner",
			wantName: "Summoner",
		},
		{
			// CRITICAL non-match: the summoner retainer (Devil Detective) sits under
			// monster.retainer.statblock, a different type-path shape — FOLLOWUPS #15
			// scope decision defers it rather than force-matching.
			name: "summoner retainer is not matched",
			scc:  "mcdm.summoner.v1/monster.retainer.statblock/devil-detective",
		},
		{
			name: "unrelated code",
			scc:  "mcdm.monsters.v1/monster.rival.4th-echelon.statblock/rival-fury",
		},
		{
			name: "empty",
			scc:  "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotSlug, gotName := owningClass(tc.scc)
			if gotSlug != tc.wantSlug || gotName != tc.wantName {
				t.Errorf("owningClass(%q) = (%q, %q), want (%q, %q)", tc.scc, gotSlug, gotName, tc.wantSlug, tc.wantName)
			}
		})
	}
}

const wolfCompanionFM = `companion: wolf
name: Wolf
scc: mcdm.beastheart.v1/monster.companion.beastheart.statblock/wolf
type: feature-group
`

const wolfAdvFM = `name: Wolf Advancement Features
scc: mcdm.beastheart.v1/monster.companion.beastheart.advancement-features/wolf
type: featureblock
`

const boilFixtureFM = `name: The Boil
scc: mcdm.summoner.v1/monster.fixture.demon.featureblock/the-boil
type: featureblock
`

const boilAdvFM = `name: The Boil Advancement Features
scc: mcdm.summoner.v1/monster.fixture.demon.advancement-features/the-boil
type: featureblock
`

// A rival summon must be untouched by this pass (it gets its own backlink from
// augmentRivalSummonerPages, not the class-owned one).
const zombieRivalSummonFM = `name: Zombie
organization: Minion
scc: mcdm.summoner.v1/monster.rival.2nd-echelon.summoner.minion/zombie
type: statblock
`

func writeFbPage(t *testing.T, path, frontmatter, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	body := "---\n" + frontmatter + "---\n\n# " + name + "\n\n---\n\n" +
		`<div class="fb-wrap" data-role="support"><article class="fb">stats</article></div>` + "\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestAugmentClassOwnedBackLinks(t *testing.T) {
	sec := t.TempDir()

	companionDir := filepath.Join(sec, "monster", "companion", "beastheart")
	writeStatblockPage(t, filepath.Join(companionDir, "wolf.md"), wolfCompanionFM, "Wolf")
	writeFbPage(t, filepath.Join(companionDir, "wolf-advancement-features.md"), wolfAdvFM, "Wolf Advancement Features")
	// index.md must be left alone (no scc, no card marker).
	if err := os.WriteFile(filepath.Join(companionDir, "index.md"), []byte("# Beastheart Companions\n"), 0644); err != nil {
		t.Fatal(err)
	}

	fixtureDir := filepath.Join(sec, "monster", "fixture", "demon")
	writeFbPage(t, filepath.Join(fixtureDir, "the-boil.md"), boilFixtureFM, "The Boil")
	writeFbPage(t, filepath.Join(fixtureDir, "the-boil-advancement-features.md"), boilAdvFM, "The Boil Advancement Features")

	// A rival summon co-located elsewhere in monster/ — must not gain a class backlink.
	rivalMinionDir := filepath.Join(sec, "monster", "rival", "2nd-echelon", "summoner", "minion")
	writeStatblockPage(t, filepath.Join(rivalMinionDir, "zombie.md"), zombieRivalSummonFM, "Zombie")

	n, errs := augmentClassOwnedBackLinks(sec)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if n != 4 {
		t.Errorf("augment count = %d, want 4 (wolf, wolf-advancement-features, the-boil, the-boil-advancement-features)", n)
	}

	wolf := readFile(filepath.Join(companionDir, "wolf.md"))
	for _, want := range []string{
		`class="sb-backlink"`,
		`href="../../../../class/beastheart/"`,
		"Beastheart",
	} {
		if !strings.Contains(wolf, want) {
			t.Errorf("wolf.md missing %q:\n%s", want, wolf)
		}
	}
	// The backlink must be the sb-wrap div's first child — contiguous with its
	// opening tag, not a preceding page-level sibling (that breaks the
	// h1+hr+card adjacency v2's CSS relies on to hide duplicate title chrome).
	if i := strings.Index(wolf, `<div class="sb-wrap"`); i < 0 {
		t.Fatalf("wolf.md missing sb-wrap card:\n%s", wolf)
	} else {
		tagEnd := strings.IndexByte(wolf[i:], '>') + i + 1
		wantPrefix := wolf[i:tagEnd] + `<p class="sb-backlink">`
		if !strings.HasPrefix(wolf[i:], wantPrefix) {
			t.Errorf("backlink must be contiguous with the sb-wrap card's opening tag (first child), got:\n%s", wolf)
		}
	}

	wolfAdv := readFile(filepath.Join(companionDir, "wolf-advancement-features.md"))
	if !strings.Contains(wolfAdv, `href="../../../../class/beastheart/"`) {
		t.Errorf("wolf-advancement-features.md missing beastheart backlink:\n%s", wolfAdv)
	}
	if i := strings.Index(wolfAdv, `<div class="fb-wrap"`); i < 0 {
		t.Fatalf("wolf-advancement-features.md missing fb-wrap card:\n%s", wolfAdv)
	} else {
		tagEnd := strings.IndexByte(wolfAdv[i:], '>') + i + 1
		wantPrefix := wolfAdv[i:tagEnd] + `<p class="sb-backlink">`
		if !strings.HasPrefix(wolfAdv[i:], wantPrefix) {
			t.Errorf("backlink must be contiguous with the fb-wrap card's opening tag (first child), got:\n%s", wolfAdv)
		}
	}

	boil := readFile(filepath.Join(fixtureDir, "the-boil.md"))
	if !strings.Contains(boil, `href="../../../../class/summoner/"`) {
		t.Errorf("the-boil.md missing summoner backlink:\n%s", boil)
	}
	boilAdv := readFile(filepath.Join(fixtureDir, "the-boil-advancement-features.md"))
	if !strings.Contains(boilAdv, `href="../../../../class/summoner/"`) {
		t.Errorf("the-boil-advancement-features.md missing summoner backlink:\n%s", boilAdv)
	}

	// index.md untouched.
	if idx := readFile(filepath.Join(companionDir, "index.md")); strings.Contains(idx, "sb-backlink") {
		t.Errorf("index.md must not gain a backlink")
	}

	// Rival summon untouched by this pass.
	if z := readFile(filepath.Join(rivalMinionDir, "zombie.md")); strings.Contains(z, "sb-backlink") {
		t.Errorf("rival summon must not gain a class-owned backlink:\n%s", z)
	}

	// Idempotent.
	n2, _ := augmentClassOwnedBackLinks(sec)
	if n2 != 0 {
		t.Errorf("second run count = %d, want 0 (idempotent)", n2)
	}
	if c := strings.Count(readFile(filepath.Join(companionDir, "wolf.md")), "sb-backlink"); c != 1 {
		t.Errorf("wolf.md backlink duplicated on re-run (count=%d)", c)
	}
}

func TestAugmentClassOwnedBackLinks_NoTree(t *testing.T) {
	n, errs := augmentClassOwnedBackLinks(t.TempDir())
	if n != 0 || len(errs) != 0 {
		t.Errorf("expected no-op, got n=%d errs=%v", n, errs)
	}
}
