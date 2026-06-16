package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goblinWarriorFM = `agility: 2
ev: "3"
keywords:
    - Goblin
    - Humanoid
level: 1
name: Goblin Warrior
organization: Horde
role: Harrier
size: 1S
speed: 6
type: statblock
`

func TestStatblockCard(t *testing.T) {
	got := statblockPreviewCard(goblinWarriorFM, "", "goblin-warrior.md", "Goblin Warrior")
	for _, want := range []string{
		`class="sb-wrap sb-prev"`,
		`class="sb-prev__link" href="goblin-warrior/"`,
		`<h2 class="sb__name">Goblin Warrior</h2>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("statblockPreviewCard missing %q in:\n%s", want, got)
		}
	}
}

func TestBestiarySourceMarking(t *testing.T) {
	summonerFM := goblinWarriorFM + "scc: mcdm.summoner.v1/minion.demon.statblock/hulking-chimor\n"
	card := statblockPreviewCard(summonerFM, "", "hulking-chimor.md", "Hulking Chimor")
	if !strings.Contains(card, `class="sb-prev__src">Summoner<`) {
		t.Errorf("summoner statblock should carry a Summoner source chip:\n%s", card)
	}
	if strings.Contains(statblockPreviewCard(goblinWarriorFM, "", "goblin-warrior.md", "Goblin Warrior"), "sb-prev__src") {
		t.Errorf("monster-book statblock must have no source chip")
	}
}

func TestIsBestiaryGroupDir(t *testing.T) {
	for _, tc := range []struct {
		dir  string
		want bool
	}{
		{"monster/goblins", true},
		{"minion/demon", true},
		{"monster/fixture/elemental", true}, // Plan 5c: fixtures now under monster/fixture/
		{"rival/summoner", true},
		{"retainer/summoner", true},
		{"monster", false}, // a type root, not a group dir
		{"minion", false},
		{"feature/ability", false},
	} {
		if got := isBestiaryGroupDir(tc.dir); got != tc.want {
			t.Errorf("isBestiaryGroupDir(%q) = %v, want %v", tc.dir, got, tc.want)
		}
	}
}

const pillarFM = `ev: "3"
level: "2"
name: Pillar
size: One square that can't be moved through
`

func TestTerrainCard(t *testing.T) {
	got := terrainCard(pillarFM, "This stone pillar can be toppled.", "pillar.md", "Pillar")
	for _, want := range []string{`href="pillar/"`, `Dynamic Terrain`,
		`<div class="sc-card__name">Pillar</div>`, `>EV</div>`, `>Level</div>`, `>Size</div>`,
		`<div class="sc-card__flavor">`, `stone pillar`} {
		if !strings.Contains(got, want) {
			t.Errorf("terrainCard missing %q in:\n%s", want, got)
		}
	}
}

const hopperFM = `ev: '-'
immunities:
    - Poison 2
keywords:
    - Angulotl
    - Humanoid
level: 1
name: Angulotl Hopper
role: Harrier
size: 1S
`

func TestRetainerPreviewCard(t *testing.T) {
	// Retainer leaves now render as rich .sb-prev cards (same as monster group landings).
	got := statblockPreviewCard(hopperFM, "", "angulotl-hopper.md", "Angulotl Hopper")
	for _, want := range []string{
		`class="sb-wrap sb-prev"`,
		`class="sb-prev__link" href="angulotl-hopper/"`,
		`<h2 class="sb__name">Angulotl Hopper</h2>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("retainer statblockPreviewCard missing %q in:\n%s", want, got)
		}
	}
}

func writeMD(t *testing.T, path, fm string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\n"+fm+"---\n\n# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

const goblinMaliceFM = "name: Goblin Malice\ntype: featureblock\n"

func TestMonsterGroupContent_Flat(t *testing.T) {
	root := t.TempDir()
	grp := filepath.Join(root, "monster", "goblins")
	// statblock/ is hoisted away: statblock + malice are sibling files in the group dir.
	writeMD(t, filepath.Join(grp, "goblin-malice.md"), goblinMaliceFM)
	writeMD(t, filepath.Join(grp, "goblin-warrior.md"), goblinWarriorFM)

	got, ok := buildMonsterGroupContent(grp, "goblins",
		[]string{"goblin-malice.md", "goblin-warrior.md"}, nil)
	if !ok {
		t.Fatal("expected goblins to be a monster group")
	}
	for _, want := range []string{
		"# Goblins\n\n---\n\n",  // strippable head for mergeGroupLanding
		`href="goblin-malice/"`, // featureblock card
		`>Goblin Malice<`,
		`href="goblin-warrior/"`, // statblock card — hoisted href (no statblock/ segment)
		`Horde Harrier`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("flat group missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "statblock/") {
		t.Errorf("hoisted group landing should carry no statblock/ hrefs:\n%s", got)
	}
}

func TestMonsterGroupContent_Echelon(t *testing.T) {
	root := t.TempDir()
	grp := filepath.Join(root, "monster", "demons")
	// statblock/ hoisted: statblock + malice are siblings inside each echelon dir.
	writeMD(t, filepath.Join(grp, "1st-echelon", "demon-malice-level-1.md"), "name: Demon Malice (Level 1)\ntype: featureblock\n")
	writeMD(t, filepath.Join(grp, "1st-echelon", "spite.md"), goblinWarriorFM)
	writeMD(t, filepath.Join(grp, "2nd-echelon", "wrath.md"), goblinWarriorFM)

	// subdirs deliberately passed out of order to exercise the natural sort.
	got, ok := buildMonsterGroupContent(grp, "demons", nil, []string{"2nd-echelon", "1st-echelon"})
	if !ok {
		t.Fatal("expected demons to be a monster group")
	}
	for _, want := range []string{
		"## 1st Echelon", // per-echelon sub-header
		"## 2nd Echelon",
		`href="1st-echelon/demon-malice-level-1/"`, // echelon-relative featureblock href
		`href="1st-echelon/spite/"`,                // echelon-relative statblock href (hoisted)
		`href="2nd-echelon/wrath/"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("echelon group missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "/statblock/") {
		t.Errorf("hoisted echelon landing should carry no statblock/ hrefs:\n%s", got)
	}
	// natural sort: 1st Echelon header must precede 2nd Echelon header.
	if strings.Index(got, "## 1st Echelon") > strings.Index(got, "## 2nd Echelon") {
		t.Errorf("echelons not natural-sorted (1st should precede 2nd):\n%s", got)
	}
}

func TestIsBestiaryEchelonDir(t *testing.T) {
	for _, tc := range []struct {
		dir  string
		want bool
	}{
		{"monster/demons/1st-echelon", true},
		{"monster/war-dogs/4th-echelon", true},
		{"monster/rival/2nd-echelon", true},
		{"monster/demons", false},  // the group dir itself
		{"monster", false},         // type root
		{"monster/goblins", false}, // a non-echelon group
		{"feature/ability/level-1", false},
	} {
		if got := isBestiaryEchelonDir(tc.dir); got != tc.want {
			t.Errorf("isBestiaryEchelonDir(%q) = %v, want %v", tc.dir, got, tc.want)
		}
	}
}

// An echelon SUB-DIRECTORY index page (monster/demons/1st-echelon/index.md) must
// render its own statblock + featureblock preview cards — not fall through to the
// old browse-index flat list. Files sit directly in the dir (relPrefix ""), so
// hrefs are bare (no echelon prefix).
func TestMonsterGroupContent_EchelonSubdir(t *testing.T) {
	root := t.TempDir()
	ech := filepath.Join(root, "monster", "demons", "1st-echelon")
	writeMD(t, filepath.Join(ech, "demon-malice-level-1.md"), "name: Demon Malice (Level 1)\ntype: featureblock\n")
	writeMD(t, filepath.Join(ech, "spite.md"), goblinWarriorFM)

	got, ok := buildMonsterGroupContent(ech, "1st-echelon",
		[]string{"demon-malice-level-1.md", "spite.md"}, nil)
	if !ok {
		t.Fatal("expected an echelon subdir to render as a monster group landing")
	}
	for _, want := range []string{
		"# 1st Echelon\n\n---\n\n",     // strippable head
		`class="sb-cards"`,             // statblock preview grid
		`class="sb-wrap sb-prev"`,      // rich statblock card
		`href="spite/"`,                // bare (no echelon prefix) statblock href
		`href="demon-malice-level-1/"`, // bare featureblock href
		`>Demon Malice (Level 1)<`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("echelon subdir landing missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "browse-index") {
		t.Errorf("echelon subdir must not render the old browse-index list:\n%s", got)
	}
}

func TestSplitByType(t *testing.T) {
	dir := t.TempDir()
	writeMD(t, filepath.Join(dir, "goblin-warrior.md"), goblinWarriorFM) // type: statblock
	writeMD(t, filepath.Join(dir, "goblin-malice.md"), goblinMaliceFM)   // type: featureblock
	sb, feat := splitByType(dir, "", []string{"goblin-malice.md", "goblin-warrior.md"})
	if len(sb) != 1 || sb[0] != "goblin-warrior.md" {
		t.Errorf("statblocks = %v, want [goblin-warrior.md]", sb)
	}
	if len(feat) != 1 || feat[0] != "goblin-malice.md" {
		t.Errorf("features = %v, want [goblin-malice.md]", feat)
	}
}

func TestHoistStatblockPath(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"monster/goblins/statblock/goblin-warrior.md", "monster/goblins/goblin-warrior.md"},
		{"monster/demons/1st-echelon/statblock/spite.md", "monster/demons/1st-echelon/spite.md"},
		{"retainer/statblock/angulotl-hopper.md", "retainer/angulotl-hopper.md"},
		// Fixtures (Plan 5c): the base drops its non-leaf featureblock/ segment;
		// the advancement-features/ subfolder is kept (mirrors companions).
		{"monster/fixture/demon/featureblock/the-boil.md", "monster/fixture/demon/the-boil.md"},
		{"monster/fixture/demon/advancement-features/the-boil.md", "monster/fixture/demon/advancement-features/the-boil.md"},
		// featureblock hoist is fixture-scoped: a featureblock/ segment in any
		// other bestiary tree is left intact.
		{"retainer/summoner/featureblock/x.md", "retainer/summoner/featureblock/x.md"},
		{"monster/goblins/goblin-malice.md", "monster/goblins/goblin-malice.md"},         // malice leaf untouched
		{"dynamic-terrain/mechanisms/pillar.md", "dynamic-terrain/mechanisms/pillar.md"}, // not bestiary statblock
		{"class/fury.md", "class/fury.md"},                                               // unrelated tree untouched
	} {
		if got := hoistStatblockPath(tc.in); got != tc.want {
			t.Errorf("hoistStatblockPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFeatureblockLabel(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"Goblin Malice", "Malice"},
		{"Tactical Stance", "Tactical Stance"},
		{"Something Else", "Feature"},
	} {
		if got := featureblockLabel(tc.name); got != tc.want {
			t.Errorf("featureblockLabel(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestMonsterGroupContent_NotAGroup(t *testing.T) {
	// the monster/ root itself and a statblock/ leaf are NOT groups
	if _, ok := buildMonsterGroupContent("/x/monster", "monster", nil, []string{"goblins"}); ok {
		t.Error("monster root should not be a group")
	}
	if _, ok := buildMonsterGroupContent("/x/monster/goblins/statblock", "statblock", []string{"a.md"}, nil); ok {
		t.Error("statblock leaf should not be a group")
	}
}

// terrainStatsFM uses the NEW stats[] shape (no scalar ev:/size:).
const terrainStatsFM = `name: Angry Beehive
type: dynamic-terrain
level: 2
terrain_type: Hazard
role: Hexer
stats:
    - name: EV
      value: "2"
    - name: Stamina
      value: "3"
    - name: Size
      value: 1S
`

func TestTerrainStat(t *testing.T) {
	if got := terrainStat(terrainStatsFM, "EV"); got != "2" {
		t.Errorf("terrainStat EV = %q, want 2", got)
	}
	if got := terrainStat(terrainStatsFM, "Size"); got != "1S" {
		t.Errorf("terrainStat Size = %q, want 1S", got)
	}
	if got := terrainStat(terrainStatsFM, "Stamina"); got != "3" {
		t.Errorf("terrainStat Stamina = %q, want 3", got)
	}
	if got := terrainStat(terrainStatsFM, "Missing"); got != "" {
		t.Errorf("terrainStat Missing = %q, want empty", got)
	}
}

func TestTerrainCard_StatsShape(t *testing.T) {
	got := terrainCard(terrainStatsFM, "An angry beehive hovers nearby.", "angry-beehive.md", "Angry Beehive")
	for _, want := range []string{
		`href="angry-beehive/"`,
		`Dynamic Terrain`,
		`<div class="sc-card__name">Angry Beehive</div>`,
		`>2<`,  // EV value
		`>1S<`, // Size value
		`>EV</div>`, `>Level</div>`, `>Size</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("terrainCard (stats shape) missing %q in:\n%s", want, got)
		}
	}
}

func TestUsesFolderIndex_Bestiary(t *testing.T) {
	for _, dir := range []string{"/x/monster", "/x/dynamic-terrain", "/x/retainer"} {
		if !usesFolderIndex(dir) {
			t.Errorf("usesFolderIndex(%q) = false, want true", dir)
		}
	}
}

func TestBuildFolderIndex_Monster(t *testing.T) {
	out := buildFolderIndex("/x/monster", "monster", []string{"goblins", "dragons"})
	for _, want := range []string{`class="sc-folder"`, `>Goblins</h3>`, `>Dragons</h3>`} {
		if !strings.Contains(out, want) {
			t.Errorf("monster folder index missing %q in:\n%s", want, out)
		}
	}
}

// The echelon group dir (files=0, subdirs=echelons) must NOT be captured by the
// folder-index branch — it must fall through to buildMonsterGroupContent.
func TestFeatureIndex_SkipsMonsterGroup(t *testing.T) {
	if _, ok := buildFeatureIndexContent("/x/monster/demons", "demons", nil, []string{"1st-echelon"}); ok {
		t.Error("monster group dir should NOT be handled by buildFeatureIndexContent's folder branch")
	}
	// but the monster ROOT (a non-group index-of-indexes) SHOULD be:
	if _, ok := buildFeatureIndexContent("/x/monster", "monster", nil, []string{"goblins"}); !ok {
		t.Error("monster root SHOULD render as folder index")
	}
}

func TestBuildCardsContent_Bestiary(t *testing.T) {
	root := t.TempDir()
	// Monster statblocks are NOT a leaf-dir card type — they render on the group
	// landing (buildMonsterGroupContent), so buildCardsContent declines them.
	sb := filepath.Join(root, "monster", "goblins")
	writeMD(t, filepath.Join(sb, "goblin-warrior.md"), goblinWarriorFM)
	if _, ok := buildCardsContent(sb, "goblins", []string{"goblin-warrior.md"}, nil); ok {
		t.Error("a monster group dir should not be handled by buildCardsContent")
	}

	// Retainers are a flat leaf dir (statblock/ hoisted away) → rich .sb-prev cards
	// inside an .sb-cards grid (same as monster group landings, not generic .sc-card).
	rt := filepath.Join(root, "retainer")
	writeMD(t, filepath.Join(rt, "angulotl-hopper.md"), hopperFM)
	got, ok := buildCardsContent(rt, "retainer", []string{"angulotl-hopper.md"}, nil)
	if !ok {
		t.Fatal("retainer leaf dir should be handled by buildCardsContent")
	}
	for _, want := range []string{
		`class="sb-cards"`,
		`class="sb-wrap sb-prev"`,
		`<h2 class="sb__name">Angulotl Hopper</h2>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("retainer leaf card missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `class="sc-card`) {
		t.Errorf("retainer leaf must not render as .sc-card (got legacy card):\n%s", got)
	}

	dt := filepath.Join(root, "dynamic-terrain", "mechanisms")
	writeMD(t, filepath.Join(dt, "pillar.md"), pillarFM)
	got, ok = buildCardsContent(dt, "mechanisms", []string{"pillar.md"}, nil)
	if !ok || !strings.Contains(got, "Dynamic Terrain") {
		t.Errorf("terrain leaf wrong:\n%s", got)
	}
}
