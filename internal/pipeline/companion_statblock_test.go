package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A beastheart companion is a real statblock in every format (SC-390): its coded
// Level-1 members ride along as features[] in book order, the "<Species>
// Advancement Features" featureblock stays its own entity (not folded in), and no
// SCC code moves.
func TestPipeline_CompanionStatblockFormats(t *testing.T) {
	inputPath := "../../testdata/fixtures/companion_statblock.md"
	baseDir := t.TempDir()
	registryPath := filepath.Join(t.TempDir(), "classification.json")

	cfg := &Config{
		Input:  inputPath,
		Locale: "en",
		Output: OutputConfig{
			BaseDir:  baseDir,
			Formats:  []string{"md", "json", "yaml"},
			Variants: VariantsConfig{Linked: true, DSE: true, DSELinked: true},
		},
		Classification: ClassificationConfig{Registry: registryPath},
	}
	if _, err := RunWithConfig(cfg, inputPath, "", registryPath); err != nil {
		t.Fatalf("pipeline run: %v", err)
	}

	reg, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	for _, code := range []string{
		"monster.companion.beastheart.statblock/panther", // same code as when it was a feature-group
		"monster.companion.beastheart.advancement-features/panther",
		"feature.ability.companion.beastheart.panther.level-1/pounce",
		"feature.companion.beastheart.panther.level-1/mighty-spring",
		"feature.companion.beastheart.panther.level-3/cat-and-mouse",
	} {
		if !strings.Contains(string(reg), code) {
			t.Errorf("registry missing %q", code)
		}
	}

	read := func(format, ext string) string {
		t.Helper()
		var found string
		_ = filepath.Walk(baseDir, func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(filepath.ToSlash(p), "/"+format+"/monster/companion/beastheart/statblock/panther"+ext) {
				found = p
			}
			return nil
		})
		if found == "" {
			t.Fatalf("no %s output for the companion under %s", format, baseDir)
		}
		b, err := os.ReadFile(found)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	for _, f := range []struct{ format, ext string }{
		{"json", ".json"}, {"yaml", ".yaml"}, {"md", ".md"}, {"md-linked", ".md"},
		{"md-dse", ".md"}, {"md-dse-linked", ".md"},
	} {
		out := read(f.format, f.ext)
		if strings.Contains(out, "feature-group") {
			t.Errorf("%s: still a feature-group:\n%s", f.format, out)
		}
		if !strings.Contains(out, "statblock") {
			t.Errorf("%s: not a statblock:\n%s", f.format, out)
		}
	}

	// json: structured stats + the two members in book order, advancement excluded.
	js := read("json", ".json")
	for _, want := range []string{`"free_strike": "1 + M"`, `"stamina": "= yours"`, `"organization": "Companion"`, `"Sneak"`} {
		if !strings.Contains(js, want) {
			t.Errorf("json missing %s:\n%s", want, js)
		}
	}
	if strings.Contains(js, "skill.intrigue/sneak") {
		t.Errorf("the skills list must be link-free:\n%s", js)
	}
	pounce, spring := strings.Index(js, `"name": "Pounce"`), strings.Index(js, `"name": "Mighty Spring"`)
	if pounce < 0 || spring < pounce {
		t.Errorf("features not Pounce then Mighty Spring (indexes %d, %d):\n%s", pounce, spring, js)
	}
	if !strings.Contains(js, `"ability_type": "Signature Ability"`) {
		t.Errorf("signature ability not mapped to Signature Ability:\n%s", js)
	}
	if strings.Contains(js, "Cat and Mouse") || strings.Contains(js, "Advancement Features") {
		t.Errorf("advancement features must not be folded into the statblock:\n%s", js)
	}

	// md-dse: a ds-sb fence carrying the same features.
	dse := read("md-dse", ".md")
	if !strings.Contains(dse, "```ds-sb\n") || !strings.Contains(dse, "name: Pounce") || !strings.Contains(dse, "name: Mighty Spring") {
		t.Errorf("md-dse missing the ds-sb block with members:\n%s", dse)
	}
	if strings.Contains(dse, "Cat and Mouse") {
		t.Errorf("md-dse folded in the advancement feature:\n%s", dse)
	}

	// md-linked keeps the book-order page body (the site parses it), now with stat
	// frontmatter: the page-body contract is unchanged.
	linked := read("md-linked", ".md")
	for _, want := range []string{"type: statblock", "<br>Free Strike", "## Pounce", "## Mighty Spring"} {
		if !strings.Contains(linked, want) {
			t.Errorf("md-linked missing %q:\n%s", want, linked)
		}
	}
}
