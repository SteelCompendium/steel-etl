package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
