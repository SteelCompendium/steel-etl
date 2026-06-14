package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBrowseMD(t *testing.T, path, fm string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\n"+fm+"---\n\n# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCollectBestiaryItems(t *testing.T) {
	browse := filepath.Join(t.TempDir(), "Browse")
	// a monster statblock (hoisted: sits directly under the group, no statblock/)
	writeBrowseMD(t, filepath.Join(browse, "monster", "goblins", "goblin-warrior.md"),
		"ev: \"3\"\nkeywords:\n    - Goblin\n    - Humanoid\nlevel: 1\nname: Goblin Warrior\norganization: Horde\nrole: Harrier\nsize: 1S\ntype: statblock\n")
	// a malice featureblock (must be EXCLUDED — type: featureblock)
	writeBrowseMD(t, filepath.Join(browse, "monster", "goblins", "goblin-malice.md"),
		"name: Goblin Malice\ntype: featureblock\n")
	// a dynamic terrain leaf
	writeBrowseMD(t, filepath.Join(browse, "dynamic-terrain", "mechanisms", "pillar.md"),
		"ev: \"3\"\nlevel: \"2\"\nname: Pillar\nsize: One square\ntype: dynamic-terrain\n")
	// a retainer (also type: statblock, but under retainer/ → classified as retainer)
	writeBrowseMD(t, filepath.Join(browse, "retainer", "angulotl-hopper.md"),
		"ev: '-'\nkeywords:\n    - Angulotl\nlevel: 1\nname: Angulotl Hopper\nrole: Harrier\nsize: 1S\ntype: statblock\n")

	items := collectBestiaryItems(browse)
	if len(items) != 3 {
		t.Fatalf("expected 3 searchable items (malice excluded), got %d: %+v", len(items), items)
	}
	byName := map[string]bestiaryItem{}
	for _, it := range items {
		byName[it.Name] = it
	}
	gw, ok := byName["Goblin Warrior"]
	if !ok {
		t.Fatal("Goblin Warrior missing")
	}
	if gw.Type != "statblock" || gw.Level != 1 || gw.EV != "3" || gw.Role != "Harrier" ||
		gw.Organization != "Horde" || gw.Size != "1S" {
		t.Errorf("Goblin Warrior fields wrong: %+v", gw)
	}
	if len(gw.Keywords) != 2 || gw.Keywords[0] != "Goblin" {
		t.Errorf("Goblin Warrior keywords wrong: %v", gw.Keywords)
	}
	if gw.Href != "../Browse/monster/goblins/goblin-warrior/" {
		t.Errorf("Goblin Warrior href wrong: %q", gw.Href)
	}
	if byName["Pillar"].Type != "terrain" {
		t.Errorf("Pillar type wrong: %q", byName["Pillar"].Type)
	}
	if byName["Angulotl Hopper"].Type != "retainer" || byName["Angulotl Hopper"].EV != "-" {
		t.Errorf("retainer wrong: %+v", byName["Angulotl Hopper"])
	}
}

func TestBuildBestiarySearchPage(t *testing.T) {
	docs := t.TempDir()
	writeBrowseMD(t, filepath.Join(docs, "Browse", "monster", "goblins", "goblin-warrior.md"),
		"ev: \"3\"\nlevel: 1\nname: Goblin Warrior\norganization: Horde\nrole: Harrier\nsize: 1S\ntype: statblock\n")

	ok, err := buildBestiarySearchPage(docs)
	if err != nil || !ok {
		t.Fatalf("expected page written, ok=%v err=%v", ok, err)
	}
	out, err := os.ReadFile(filepath.Join(docs, "Bestiary", "index.md"))
	if err != nil {
		t.Fatalf("Bestiary/index.md not written: %v", err)
	}
	s := string(out)
	for _, want := range []string{
		"search:\n  exclude: true",
		`<div class="sc-bestiary-mount">`,
		`<script type="application/json" class="sc-browse-data">`,
		`"name":"Goblin Warrior"`,
		`"href":"../Browse/monster/goblins/goblin-warrior/"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("search page missing %q in:\n%s", want, s)
		}
	}
}

func TestBestiaryItemType_SummonerTrees(t *testing.T) {
	// Summoner portfolio statblocks index as "statblock"; their retainers as
	// "retainer". Group lore / index pages are skipped.
	for _, tc := range []struct {
		rel, fmType, want string
	}{
		{"minion/demon/hulking-chimor.md", "statblock", "statblock"},
		{"champion/fey/celestial-attendant.md", "statblock", "statblock"},
		{"rival/summoner/3rd-echelon/rival-summoner.md", "statblock", "statblock"},
		{"retainer/summoner/devil-detective.md", "statblock", "retainer"},
		{"minion/demon/index.md", "statblock", ""}, // index page skipped
		// Fixtures (Plan 5c) are featureblocks under monster/fixture/: the base
		// indexes as "fixture"; the sibling advancement-features page is excluded.
		{"monster/fixture/elemental/primordial-crystal.md", "featureblock", "fixture"},
		{"monster/fixture/elemental/advancement-features/primordial-crystal.md", "featureblock", ""},
		// Malice/feature featureblocks stay out of the searchable index.
		{"monster/goblins/goblin-malice.md", "featureblock", ""},
	} {
		if got := bestiaryItemType(tc.rel, tc.fmType); got != tc.want {
			t.Errorf("bestiaryItemType(%q) = %q, want %q", tc.rel, got, tc.want)
		}
	}
}

func TestCollectBestiaryItems_SummonerSource(t *testing.T) {
	docs := t.TempDir()
	dir := filepath.Join(docs, "Browse", "minion", "demon")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	page := "---\nname: Hulking Chimor\ntype: statblock\nlevel: 1\nev: \"3\"\n" +
		"scc: mcdm.summoner.v1/minion.demon.statblock/hulking-chimor\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "hulking-chimor.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	items := collectBestiaryItems(filepath.Join(docs, "Browse"))
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1: %+v", len(items), items)
	}
	if items[0].Type != "statblock" || items[0].Source != "Summoner" {
		t.Errorf("got type=%q source=%q, want statblock/Summoner", items[0].Type, items[0].Source)
	}
}

func TestCollectBestiaryItems_TerrainStatsShape(t *testing.T) {
	// Terrain frontmatter uses stats[] not scalar ev:/size: — ensure collectBestiaryItems
	// still picks up the correct values via statField.
	browse := filepath.Join(t.TempDir(), "Browse")
	writeBrowseMD(t, filepath.Join(browse, "dynamic-terrain", "hazards", "angry-beehive.md"),
		"name: Angry Beehive\ntype: dynamic-terrain\nlevel: 2\nstats:\n    - name: EV\n      value: \"2\"\n    - name: Stamina\n      value: \"3\"\n    - name: Size\n      value: 1S\n")

	items := collectBestiaryItems(browse)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d: %+v", len(items), items)
	}
	it := items[0]
	if it.EV != "2" {
		t.Errorf("terrain EV = %q, want 2", it.EV)
	}
	if it.Size != "1S" {
		t.Errorf("terrain Size = %q, want 1S", it.Size)
	}
}

func TestBuildBestiarySearchPage_NoItems(t *testing.T) {
	docs := t.TempDir()
	if err := os.MkdirAll(filepath.Join(docs, "Browse"), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, err := buildBestiarySearchPage(docs)
	if err != nil || ok {
		t.Errorf("expected no-op (ok=false) with no items, got ok=%v err=%v", ok, err)
	}
	if _, err := os.Stat(filepath.Join(docs, "Bestiary", "index.md")); !os.IsNotExist(err) {
		t.Error("no page should be written when there are no items")
	}
}
