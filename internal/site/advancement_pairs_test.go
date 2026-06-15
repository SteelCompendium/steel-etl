package site

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAdvancementPairNavOrder(t *testing.T) {
	// Base-first, paired, index.md first — regardless of input file order.
	files := []string{"wolf-advancement-features.md", "boar-advancement-features.md", "wolf.md", "boar.md"}
	order, ok := advancementPairNavOrder(files, nil)
	if !ok {
		t.Fatal("expected ok=true for a pair dir")
	}
	want := []string{"index.md", "boar.md", "boar-advancement-features.md", "wolf.md", "wolf-advancement-features.md"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("nav order = %v, want %v", order, want)
	}

	// Not a pair dir → ok=false (caller writes a plain title-only .nav.yml).
	if _, ok := advancementPairNavOrder([]string{"cutter.md"}, nil); ok {
		t.Error("expected ok=false for a dir with no advancement-features leaves")
	}
	// Stray subdir → ok=false.
	if _, ok := advancementPairNavOrder([]string{"wolf.md", "wolf-advancement-features.md"}, []string{"x"}); ok {
		t.Error("expected ok=false when subdirs are present")
	}
}

func TestBuildAdvancementPairContent(t *testing.T) {
	dir := t.TempDir()
	write := func(name, fm string) {
		os.WriteFile(filepath.Join(dir, name), []byte("---\n"+fm+"\n---\n\nbody"), 0644)
	}
	write("wolf.md", "name: Wolf\ntype: feature-group")
	write("wolf-advancement-features.md", "name: Wolf Advancement Features\ntype: featureblock")
	write("boar.md", "name: Boar\ntype: feature-group")
	write("boar-advancement-features.md", "name: Boar Advancement Features\ntype: featureblock")

	files := []string{"boar.md", "boar-advancement-features.md", "wolf.md", "wolf-advancement-features.md"}
	out, ok := buildAdvancementPairContent(filepath.Join("monster/companion/beastheart"), "beastheart", files, nil)
	if !ok {
		t.Fatal("expected ok=true for a dir with advancement pairs")
	}

	// 2-column pair grid wrapper.
	if !strings.Contains(out, `class="sc-cards sc-cards--pairs"`) {
		t.Errorf("missing sc-cards--pairs wrapper:\n%s", out)
	}
	// Each base is immediately followed by its advancement (base-first ordering).
	wolfBase := strings.Index(out, `href="wolf/"`)
	wolfAdv := strings.Index(out, `href="wolf-advancement-features/"`)
	if wolfBase < 0 || wolfAdv < 0 || wolfBase > wolfAdv {
		t.Errorf("expected base wolf card before its advancement card; base=%d adv=%d", wolfBase, wolfAdv)
	}
	// Companion crest + distinguishing eyebrows.
	if !strings.Contains(out, ">Companion<") || !strings.Contains(out, ">Advancement Features<") {
		t.Errorf("expected Companion and Advancement Features eyebrows:\n%s", out)
	}
}

func TestBuildAdvancementPairContent_NoPairs(t *testing.T) {
	if _, ok := buildAdvancementPairContent("monster/goblins", "goblins", []string{"cutter.md"}, nil); ok {
		t.Error("expected ok=false when no advancement-features leaves are present")
	}
}

func TestBuildAdvancementPairContent_Fixture(t *testing.T) {
	dir := t.TempDir()
	write := func(name, fm string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("---\n"+fm+"\n---\n\nbody"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("the-boil.md", "name: The Boil\ntype: featureblock")
	write("the-boil-advancement-features.md", "name: The Boil Advancement Features\ntype: featureblock")

	files := []string{"the-boil.md", "the-boil-advancement-features.md"}
	out, ok := buildAdvancementPairContent("monster/fixture/demon", "demon", files, nil)
	if !ok {
		t.Fatal("expected ok=true for a fixture dir with advancement pairs")
	}
	if !strings.Contains(out, ">Fixture<") {
		t.Errorf("expected Fixture eyebrow for a fixture dir:\n%s", out)
	}
	base := strings.Index(out, `href="the-boil/"`)
	adv := strings.Index(out, `href="the-boil-advancement-features/"`)
	if base < 0 || adv < 0 || base > adv {
		t.Errorf("expected base fixture card before its advancement card; base=%d adv=%d", base, adv)
	}
}

func TestBuildAdvancementPairContent_SubdirsFallThrough(t *testing.T) {
	// A dir with advancement pairs AND a stray subdir should fall through (ok=false).
	if _, ok := buildAdvancementPairContent("monster/fixture/demon", "demon",
		[]string{"the-boil.md", "the-boil-advancement-features.md"}, []string{"extra"}); ok {
		t.Error("expected ok=false when subdirs are present")
	}
}

func TestBuildAdvancementPairContent_CompanionPreview(t *testing.T) {
	dir := t.TempDir()
	base := "---\nname: Panther\nscc: mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther\ntype: feature-group\n---\n\n# Panther\n\n<div class=\"sb-wrap\">…</div>\n"
	adv := "---\nname: Panther\ntype: featureblock\n---\n\n# Panther\n"
	if err := os.WriteFile(filepath.Join(dir, "panther.md"), []byte(base), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "panther-advancement-features.md"), []byte(adv), 0644); err != nil {
		t.Fatal(err)
	}
	companionStatblockCache = map[string]sbIsland{
		"mcdm.beastheart.v1/monster.companion.beastheart.statblock/panther": {
			Name: "Panther", ID: "panther", Role: "Companion", RoleKey: "leader", Level: "1",
		},
	}
	out, ok := buildAdvancementPairContent(dir, "beastheart",
		[]string{"panther.md", "panther-advancement-features.md"}, nil)
	if !ok {
		t.Fatal("ok=false")
	}
	if !strings.Contains(out, "sb-prev") {
		t.Errorf("expected a .sb-prev companion preview, got:\n%s", out)
	}
	if !strings.Contains(out, "sb-cards") || !strings.Contains(out, `data-sbprev-stats="on"`) {
		t.Errorf("expected sb-cards grid with zone defaults, got:\n%s", out)
	}
	if !strings.Contains(out, "Advancement Features") {
		t.Errorf("advancement card missing:\n%s", out)
	}
}
