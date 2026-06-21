package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const detectiveFM = `name: Devil Detective
organization: Retainer
role: Controller
type: statblock
scc: mcdm.summoner.v1/monster.retainer.statblock/devil-detective
`

const retainerRazorFM = `name: Razor
organization: Minion
role: Harrier
type: statblock
scc: mcdm.summoner.v1/monster.retainer.summoner.minion.statblock/razor
`

// A Monsters-book retainer (not summoner) must NOT get a summons/advancement augment.
const monsterRetainerFM = `name: Angulotl Hopper
organization: Retainer
role: Harrier
type: statblock
scc: mcdm.monsters.v1/monster.retainer.statblock/angulotl-hopper
`

func TestAugmentSummonerRetainerPages(t *testing.T) {
	sec := t.TempDir()
	ret := filepath.Join(sec, "monster", "retainer")
	writeStatblockPage(t, filepath.Join(ret, "devil-detective.md"), detectiveFM, "Devil Detective")
	writeStatblockPage(t, filepath.Join(ret, "angulotl-hopper.md"), monsterRetainerFM, "Angulotl Hopper")
	// Advancement featureblock sibling (flattened name) with one leveled member.
	if err := os.WriteFile(filepath.Join(ret, "devil-detective-advancement-features.md"),
		[]byte("---\nname: Devil Detective\ntype: featureblock\nfeatures:\n  - name: Soul Sleuth\n    level: 4\n---\n\n# Devil Detective\n"), 0644); err != nil {
		t.Fatal(err)
	}
	minion := filepath.Join(ret, "summoner", "minion")
	writeStatblockPage(t, filepath.Join(minion, "razor.md"), retainerRazorFM, "Razor")

	n, errs := augmentSummonerRetainerPages(sec)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if n != 2 { // detective page (advancement + summons) + 1 minion back-link
		t.Errorf("augment count = %d, want 2", n)
	}

	dd := readFile(filepath.Join(ret, "devil-detective.md"))
	for _, want := range []string{
		"## Advancement Features",
		"Soul Sleuth", // advancementCardInner lists the member
		`href="../devil-detective-advancement-features/"`,
		"## Summons",
		`href="../summoner/minion/razor/"`,
	} {
		if !strings.Contains(dd, want) {
			t.Errorf("detective page missing %q:\n%s", want, dd)
		}
	}

	// Monsters-book retainer must be untouched.
	if ah := readFile(filepath.Join(ret, "angulotl-hopper.md")); strings.Contains(ah, "## Summons") {
		t.Errorf("angulotl-hopper must not get a Summons block")
	}

	// Minion back-link to the detective.
	rz := readFile(filepath.Join(minion, "razor.md"))
	if c := strings.Count(rz, "sb-backlink"); c != 1 {
		t.Errorf("razor sb-backlink count = %d, want 1", c)
	}
	if !strings.Contains(rz, `href="../../../devil-detective/"`) {
		t.Errorf("razor missing back-link href:\n%s", rz)
	}
	if strings.Index(rz, "sb-backlink") > strings.Index(rz, `<div class="sb-wrap"`) {
		t.Errorf("back-link should precede the sb-wrap card:\n%s", rz)
	}

	// Idempotent.
	n2, _ := augmentSummonerRetainerPages(sec)
	if n2 != 0 {
		t.Errorf("second run count = %d, want 0 (idempotent)", n2)
	}
}

func TestAugmentSummonerRetainerPages_NoTree(t *testing.T) {
	n, errs := augmentSummonerRetainerPages(t.TempDir())
	if n != 0 || len(errs) != 0 {
		t.Errorf("expected no-op, got n=%d errs=%v", n, errs)
	}
}
