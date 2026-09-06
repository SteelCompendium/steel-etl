package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SC-306: buildMaliceBandCache renders each family's Malice band straight from
// parseStatblockIslandFeatures — a third, easy-to-miss renderStatblockFeature
// call site (statblock_card.go's own loop and renderFbFeats are the other
// two). This band is spliced directly into a statblock's own canonical Browse
// page (augmentMonsterMaliceBands → spliceMaliceBand), never through the
// embed_cards stripFeatIDs path, so its cards need real sc-feat- ids or
// Material's indexer glues every family Malice feature's name onto the band's
// own title. Two same-named features also lock the per-band dedupe suffix.
func TestBuildMaliceBandCache_FeatureIDs(t *testing.T) {
	dir := t.TempDir()
	const page = `---
name: Devil Malice
type: featureblock
kind: malice
scc: mcdm.monsters.v1/monster.devil/devil-malice
flavor: At the start of any devil's turn, you can spend Malice to activate one of the following features.
---

> 👿 **Devilish Suggestion**
>
> A devil plants a suggestion in a nearby creature's mind.

> 👿 **Devilish Suggestion**
>
> A second devil plants the same suggestion.
`
	path := filepath.Join(dir, "devil-malice.md")
	if err := os.WriteFile(path, []byte(page), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{}
	entries := []sourceEntry{{relPath: "devil-malice.md", absPath: path, sourceDir: dir}}
	bands := buildMaliceBandCache(cfg, entries)

	key, ok := maliceKey("mcdm.monsters.v1/monster.devil/devil-malice")
	if !ok {
		t.Fatal("maliceKey did not resolve for the test fixture's scc")
	}
	candidates, ok := bands[key]
	if !ok || len(candidates) != 1 {
		t.Fatalf("expected exactly one malice candidate for %q, got %v", key, candidates)
	}
	html := candidates[0].html

	for _, want := range []string{
		` id="sc-feat-malice-devilish-suggestion"`,
		` id="sc-feat-malice-devilish-suggestion-2"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in malice band html:\n%s", want, html)
		}
	}
}
