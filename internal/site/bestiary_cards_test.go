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
	got := statblockCard(goblinWarriorFM, "", "goblin-warrior.md", "Goblin Warrior")
	for _, want := range []string{
		`class="sc-card sc-fil"`,
		`href="goblin-warrior/"`,
		`Horde Harrier`, // organization + role type label
		`<div class="sc-card__name">Goblin Warrior</div>`,
		`<span class="sc-tag">Goblin</span>`,
		`<span class="sc-tag">Humanoid</span>`,
		`>EV</div>`, `>Level</div>`, `>Size</div>`, `>Speed</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("statblockCard missing %q in:\n%s", want, got)
		}
	}
}

func TestBestiarySourceMarking(t *testing.T) {
	// Summoner-book statblocks (scc prefix mcdm.summoner.) are marked
	// "Summoner · <label>"; Monsters-book statblocks (no/other prefix) are not.
	summonerFM := goblinWarriorFM + "scc: mcdm.summoner.v1/minion.demon.statblock/hulking-chimor\n"
	if got := bestiarySource(summonerFM); got != "Summoner" {
		t.Errorf("bestiarySource(summoner) = %q, want Summoner", got)
	}
	if got := bestiarySource(goblinWarriorFM); got != "" {
		t.Errorf("bestiarySource(monster) = %q, want empty", got)
	}
	if got := withSource(summonerFM, "Minion Brute"); got != "Summoner · Minion Brute" {
		t.Errorf("withSource = %q", got)
	}
	// The marker reaches the rendered card.
	card := statblockCard(summonerFM, "", "hulking-chimor.md", "Hulking Chimor")
	if !strings.Contains(card, "Summoner · Horde Harrier") {
		t.Errorf("summoner statblock card not marked:\n%s", card)
	}
	if strings.Contains(statblockCard(goblinWarriorFM, "", "goblin-warrior.md", "Goblin Warrior"), "Summoner ·") {
		t.Error("monster statblock card must not be marked Summoner")
	}
}

func TestIsBestiaryGroupDir(t *testing.T) {
	for _, tc := range []struct {
		dir  string
		want bool
	}{
		{"monster/goblins", true},
		{"minion/demon", true},
		{"fixture/elemental", true},
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

func TestStatblockTypeLabel(t *testing.T) {
	for _, tc := range []struct{ org, role, want string }{
		{"Horde", "Harrier", "Horde Harrier"},
		{"Horde", "", "Horde"},
		{"", "Harrier", "Harrier"},
		{"", "", "Statblock"},
	} {
		fm := "organization: " + tc.org + "\nrole: " + tc.role + "\n"
		if got := statblockTypeLabel(fm); got != tc.want {
			t.Errorf("statblockTypeLabel(org=%q role=%q) = %q, want %q", tc.org, tc.role, got, tc.want)
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

func TestRetainerCard(t *testing.T) {
	got := retainerCard(hopperFM, "", "angulotl-hopper.md", "Angulotl Hopper")
	for _, want := range []string{`href="angulotl-hopper/"`, `Retainer Harrier`,
		`<span class="sc-tag">Angulotl</span>`, `Poison 2`, `>Level</div>`, `>Size</div>`} {
		if !strings.Contains(got, want) {
			t.Errorf("retainerCard missing %q in:\n%s", want, got)
		}
	}
}

func TestRetainerCardNoRole(t *testing.T) {
	got := retainerCard("level: 1\nname: Plain\n", "", "plain.md", "Plain")
	if !strings.Contains(got, `<div class="sc-card__type">Retainer</div>`) {
		t.Errorf("roleless retainer should label \"Retainer\" with no trailing space:\n%s", got)
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
		{"monster/goblins/goblin-malice.md", "monster/goblins/goblin-malice.md"},         // featureblock untouched
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

	// Retainers are a flat leaf dir (statblock/ hoisted away) → retainer cards.
	rt := filepath.Join(root, "retainer")
	writeMD(t, filepath.Join(rt, "angulotl-hopper.md"), hopperFM)
	got, ok := buildCardsContent(rt, "retainer", []string{"angulotl-hopper.md"}, nil)
	if !ok || !strings.Contains(got, "Retainer Harrier") {
		t.Errorf("retainer leaf should route to retainerCard:\n%s", got)
	}

	dt := filepath.Join(root, "dynamic-terrain", "mechanisms")
	writeMD(t, filepath.Join(dt, "pillar.md"), pillarFM)
	got, ok = buildCardsContent(dt, "mechanisms", []string{"pillar.md"}, nil)
	if !ok || !strings.Contains(got, "Dynamic Terrain") {
		t.Errorf("terrain leaf wrong:\n%s", got)
	}
}
